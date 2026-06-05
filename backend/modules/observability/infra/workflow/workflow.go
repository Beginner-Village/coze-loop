// Copyright (c) 2026 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package workflow

import (
	"context"
	"fmt"

	"github.com/coze-dev/coze-loop/backend/modules/observability/domain/component/rpc"
	"github.com/coze-dev/coze-loop/backend/modules/observability/domain/trace/entity/loop_span"
)

// workflowTagKey is the span tag/attribute key that carries the workflow info
// payload. It matches the `workflow` field name defined in the observability
// span IDL (idl/.../span.thrift EncryptionInfo.workflow).
const workflowTagKey = "workflow"

type WorkflowProvider struct{}

func NewWorkflowProvider() rpc.IWorkflowProvider {
	return &WorkflowProvider{}
}

// BatchGetWorkflows returns a map keyed by "{traceID}-{spanID}" whose value is
// the workflow info payload for spans that request it (Encryption.NeedWorkflow).
//
// The map key format is intentionally identical to the consumer in
// application/convertor/trace/span.go (SpanDO2DTO), which looks up
// workflowMap[fmt.Sprintf("%s-%s", TraceID, SpanID)] for spans whose
// Encryption.NeedWorkflow is true.
//
// Data source: this is the minimal, self-contained implementation. It extracts
// the workflow payload directly from the span's own tags/attributes (the
// `workflow` key). It does NOT call an external workflow service, because no
// such RPC client exists in this module today (see OPEN QUESTION below). Spans
// that do not request workflow info, or that carry no workflow tag, are skipped
// — no value is fabricated.
//
// OPEN QUESTION: the "full" implementation should resolve workflow_id -> a rich
// workflow info / URL via a dedicated workflow service. That data source and
// contract are not yet defined in this repo.
func (w *WorkflowProvider) BatchGetWorkflows(ctx context.Context, spans loop_span.SpanList) (map[string]string, error) {
	result := make(map[string]string)
	for _, s := range spans {
		if s == nil || !s.Encryption.NeedWorkflow {
			continue
		}
		val, ok := s.GetMetaDataValue(workflowTagKey).(string)
		if !ok || val == "" {
			continue
		}
		key := fmt.Sprintf("%s-%s", s.TraceID, s.SpanID)
		result[key] = val
	}
	return result, nil
}
