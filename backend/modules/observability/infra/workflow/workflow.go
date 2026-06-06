// Copyright (c) 2026 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/coze-dev/coze-loop/backend/infra/db"
	"github.com/coze-dev/coze-loop/backend/modules/observability/domain/component/rpc"
	"github.com/coze-dev/coze-loop/backend/modules/observability/domain/trace/entity/loop_span"
	"github.com/coze-dev/coze-loop/backend/modules/observability/infra/repo/mysql"
	"github.com/coze-dev/coze-loop/backend/modules/observability/infra/repo/mysql/gorm_gen/model"
	"github.com/coze-dev/coze-loop/backend/pkg/logs"
)

const (
	// workflowIDTagKey is the span tag/attribute key carrying the workflow id.
	// Spans emitted by workflow runs only carry this numeric id; the human
	// readable name/icon live in the workflow_meta snapshot table.
	workflowIDTagKey = "workflow_id"
	// workflowTagKey is the legacy tag that may already carry a fully-resolved
	// workflow info payload on the span itself. Used as a fallback so spans that
	// already embed the info keep working.
	workflowTagKey = "workflow"
)

// workflowInfo is the JSON payload returned per span. It is intentionally small
// and self-describing so the frontend can render name + icon without another
// round trip. Value is stored at workflowMap["{traceID}-{spanID}"].
type workflowInfo struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	IconURI string `json:"icon_uri"`
}

type WorkflowProvider struct {
	metaDao mysql.IWorkflowMetaDao
}

// NewWorkflowProvider builds the provider backed by the workflow_meta snapshot
// table. provider may be nil in environments where the table is not
// provisioned; in that case BatchGetWorkflows degrades to the legacy "read
// embedded workflow tag" behavior and never errors.
func NewWorkflowProvider(provider db.Provider) rpc.IWorkflowProvider {
	var metaDao mysql.IWorkflowMetaDao
	if provider != nil {
		metaDao = mysql.NewWorkflowMetaDaoImpl(provider)
	}
	return &WorkflowProvider{metaDao: metaDao}
}

// BatchGetWorkflows returns a map keyed by "{traceID}-{spanID}" whose value is a
// JSON workflow info payload ({"id","name","icon_uri"}).
//
// The key format is identical to the consumer in
// application/convertor/trace/span.go (SpanDO2DTO), which looks up
// workflowMap[fmt.Sprintf("%s-%s", TraceID, SpanID)].
//
// Resolution strategy:
//  1. For every span carrying a numeric "workflow_id" tag, collect the id and
//     batch-query the workflow_meta snapshot table to resolve name/icon. This is
//     the real upgrade: production spans only carry the id, and the snapshot
//     supplies the readable metadata.
//  2. If a span already carries a resolved "workflow" tag (legacy producers),
//     that value is passed through verbatim.
//
// Selection: a span is processed if it has a workflow_id (or legacy workflow)
// tag. Encryption.NeedWorkflow is honored when set but is NOT required, because
// that flag is not set anywhere in the current pipeline; gating on it alone
// would make this a no-op. See OPEN QUESTION.
func (w *WorkflowProvider) BatchGetWorkflows(ctx context.Context, spans loop_span.SpanList) (map[string]string, error) {
	result := make(map[string]string)

	type pending struct {
		key        string
		workflowID int64
	}
	var toResolve []pending
	idSet := make(map[int64]struct{})

	for _, s := range spans {
		if s == nil {
			continue
		}
		key := fmt.Sprintf("%s-%s", s.TraceID, s.SpanID)

		// Legacy: span already carries a resolved workflow payload.
		if legacy, ok := s.GetMetaDataValue(workflowTagKey).(string); ok && legacy != "" {
			result[key] = legacy
			continue
		}

		workflowID, ok := extractWorkflowID(s)
		if !ok {
			continue
		}
		toResolve = append(toResolve, pending{key: key, workflowID: workflowID})
		idSet[workflowID] = struct{}{}
	}

	if len(toResolve) == 0 {
		return result, nil
	}

	metaByID := map[int64]*model.ObservabilityWorkflowMeta{}
	if w.metaDao != nil {
		ids := make([]int64, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}
		var err error
		metaByID, err = w.metaDao.BatchGet(ctx, ids)
		if err != nil {
			// Don't fail the whole trace read because metadata lookup failed;
			// log and fall through with whatever we have (ids without names).
			logs.CtxWarn(ctx, "workflow meta batch get failed: %v", err)
			metaByID = map[int64]*model.ObservabilityWorkflowMeta{}
		}
	}

	for _, p := range toResolve {
		info := workflowInfo{ID: p.workflowID}
		if meta, ok := metaByID[p.workflowID]; ok {
			info.Name = meta.Name
			info.IconURI = meta.IconURI
		}
		payload, err := json.Marshal(info)
		if err != nil {
			continue
		}
		result[p.key] = string(payload)
	}
	return result, nil
}

// extractWorkflowID reads the numeric workflow id from a span's tags. The id may
// arrive as a string or an integer tag depending on the producer.
func extractWorkflowID(s *loop_span.Span) (int64, bool) {
	v := s.GetMetaDataValue(workflowIDTagKey)
	switch val := v.(type) {
	case string:
		if val == "" {
			return 0, false
		}
		id, err := strconv.ParseInt(val, 10, 64)
		if err != nil || id == 0 {
			return 0, false
		}
		return id, true
	case int64:
		if val == 0 {
			return 0, false
		}
		return val, true
	default:
		return 0, false
	}
}
