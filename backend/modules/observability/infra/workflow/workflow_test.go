// Copyright (c) 2026 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/coze-dev/coze-loop/backend/modules/observability/domain/trace/entity/loop_span"
	"github.com/coze-dev/coze-loop/backend/modules/observability/infra/repo/mysql"
	"github.com/coze-dev/coze-loop/backend/modules/observability/infra/repo/mysql/gorm_gen/model"
)

// fakeMetaDao is a hand-written stand-in for mysql.IWorkflowMetaDao backed by a
// fixed in-memory map seeded from the production workflow_meta snapshot.
type fakeMetaDao struct {
	rows map[int64]*model.ObservabilityWorkflowMeta
	err  error
}

func (f *fakeMetaDao) BatchGet(_ context.Context, ids []int64) (map[int64]*model.ObservabilityWorkflowMeta, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make(map[int64]*model.ObservabilityWorkflowMeta)
	for _, id := range ids {
		if r, ok := f.rows[id]; ok {
			out[id] = r
		}
	}
	return out, nil
}

var _ mysql.IWorkflowMetaDao = (*fakeMetaDao)(nil)

// snapshotFixture mirrors real rows exported from the production workflow_meta
// snapshot. These ids were confirmed present in observability_workflow_meta.
func snapshotFixture() map[int64]*model.ObservabilityWorkflowMeta {
	return map[int64]*model.ObservabilityWorkflowMeta{
		7621100656002072576: {ID: 7621100656002072576, Name: "tranfer_bbw_2603", IconURI: ""},
		7642550560138199040: {ID: 7642550560138199040, Name: "trans_recent_payee_bbw_2603_1", IconURI: ""},
	}
}

func newProviderWithDao(dao mysql.IWorkflowMetaDao) *WorkflowProvider {
	return &WorkflowProvider{metaDao: dao}
}

func TestNewWorkflowProvider(t *testing.T) {
	provider := NewWorkflowProvider(nil)
	assert.NotNil(t, provider)
	assert.Implements(t, (*interface {
		BatchGetWorkflows(context.Context, loop_span.SpanList) (map[string]string, error)
	})(nil), provider)
}

func TestWorkflowProvider_ResolvesSubWorkflowIDFromSnapshot(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	// Production shape: multiple node spans of one workflow carry the same
	// sub_workflow_id as an Int64 in tags_long.
	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:  "trace-a",
			SpanID:   "span-start",
			TagsLong: map[string]int64{"sub_workflow_id": 7621100656002072576},
		},
		{
			TraceID:  "trace-b",
			SpanID:   "span-end",
			TagsLong: map[string]int64{"sub_workflow_id": 7642550560138199040},
		},
	})
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.JSONEq(t,
		`{"id":7621100656002072576,"name":"tranfer_bbw_2603","icon_uri":""}`,
		got["trace-a-span-start"])
	assert.JSONEq(t,
		`{"id":7642550560138199040,"name":"trans_recent_payee_bbw_2603_1","icon_uri":""}`,
		got["trace-b-span-end"])
}

func TestWorkflowProvider_SubWorkflowIDFromStringTag(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:    "trace-c",
			SpanID:     "span-c",
			TagsString: map[string]string{"sub_workflow_id": "7621100656002072576"},
		},
	})
	assert.NoError(t, err)
	assert.JSONEq(t,
		`{"id":7621100656002072576,"name":"tranfer_bbw_2603","icon_uri":""}`,
		got["trace-c-span-c"])
}

func TestWorkflowProvider_EvalTargetSecondarySource(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	// No sub_workflow_id; evaluation span exposes the workflow id as
	// eval_target_id when eval_target_type denotes a workflow.
	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:    "trace-eval",
			SpanID:     "span-eval",
			TagsLong:   map[string]int64{"eval_target_id": 7642550560138199040},
			TagsString: map[string]string{"eval_target_type": "CozeWorkflow"},
		},
		{
			// Prompt eval target must NOT be treated as a workflow.
			TraceID:    "trace-prompt",
			SpanID:     "span-prompt",
			TagsLong:   map[string]int64{"eval_target_id": 7621100656002072576},
			TagsString: map[string]string{"eval_target_type": "LoopPrompt"},
		},
	})
	assert.NoError(t, err)
	assert.JSONEq(t,
		`{"id":7642550560138199040,"name":"trans_recent_payee_bbw_2603_1","icon_uri":""}`,
		got["trace-eval-span-eval"])
	_, ok := got["trace-prompt-span-prompt"]
	assert.False(t, ok)
}

func TestWorkflowProvider_UnknownIDStillReturnsID(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:  "trace-d",
			SpanID:   "span-d",
			TagsLong: map[string]int64{"sub_workflow_id": 111222333444},
		},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"id":111222333444,"name":"","icon_uri":""}`, got["trace-d-span-d"])
}

func TestWorkflowProvider_LegacyWorkflowTagPassthrough(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:    "trace-e",
			SpanID:     "span-e",
			TagsString: map[string]string{"workflow": "already-resolved-payload"},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"trace-e-span-e": "already-resolved-payload"}, got)
}

func TestWorkflowProvider_DaoErrorDegradesToIDOnly(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{err: errors.New("db down")})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:  "trace-f",
			SpanID:   "span-f",
			TagsLong: map[string]int64{"sub_workflow_id": 7621100656002072576},
		},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"id":7621100656002072576,"name":"","icon_uri":""}`, got["trace-f-span-f"])
}

func TestWorkflowProvider_NilDaoDegrades(t *testing.T) {
	provider := newProviderWithDao(nil)

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{TraceID: "t", SpanID: "s", TagsLong: map[string]int64{"sub_workflow_id": 7621100656002072576}},
		{TraceID: "t2", SpanID: "s2", TagsString: map[string]string{"workflow": "legacy"}},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"id":7621100656002072576,"name":"","icon_uri":""}`, got["t-s"])
	assert.Equal(t, "legacy", got["t2-s2"])
}

func TestWorkflowProvider_NoWorkflowTagsSkipped(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{TraceID: "t", SpanID: "s"},
		nil,
		{TraceID: "t2", SpanID: "s2", TagsLong: map[string]int64{"sub_workflow_id": 0}},
		{TraceID: "t3", SpanID: "s3", TagsString: map[string]string{"sub_workflow_id": "not-a-number"}},
	})
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestWorkflowProvider_EmptyAndNilSpans(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{})
	assert.NoError(t, err)
	assert.Empty(t, got)

	got, err = provider.BatchGetWorkflows(context.Background(), nil)
	assert.NoError(t, err)
	assert.Empty(t, got)
}
