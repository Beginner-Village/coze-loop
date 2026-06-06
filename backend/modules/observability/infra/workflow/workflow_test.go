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

// snapshotFixture mirrors three real rows from workflow_meta_all.tsv.
func snapshotFixture() map[int64]*model.ObservabilityWorkflowMeta {
	return map[int64]*model.ObservabilityWorkflowMeta{
		7647058738179735552: {ID: 7647058738179735552, Name: "test", IconURI: "default_icon/plugin_default_icon.png"},
		7647056212059488256: {ID: 7647056212059488256, Name: "tessss", IconURI: "default_icon/default_workflow_icon.png"},
		7647061441349943296: {ID: 7647061441349943296, Name: "tranfer_202601_linan_1", IconURI: ""},
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

func TestWorkflowProvider_ResolvesWorkflowIDFromSnapshot(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:    "trace-a",
			SpanID:     "span-a",
			TagsString: map[string]string{"workflow_id": "7647058738179735552"},
		},
		{
			TraceID:    "trace-b",
			SpanID:     "span-b",
			TagsString: map[string]string{"workflow_id": "7647056212059488256"},
		},
	})
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.JSONEq(t,
		`{"id":7647058738179735552,"name":"test","icon_uri":"default_icon/plugin_default_icon.png"}`,
		got["trace-a-span-a"])
	assert.JSONEq(t,
		`{"id":7647056212059488256,"name":"tessss","icon_uri":"default_icon/default_workflow_icon.png"}`,
		got["trace-b-span-b"])
}

func TestWorkflowProvider_WorkflowIDFromLongTag(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:  "trace-c",
			SpanID:   "span-c",
			TagsLong: map[string]int64{"workflow_id": 7647061441349943296},
		},
	})
	assert.NoError(t, err)
	assert.JSONEq(t,
		`{"id":7647061441349943296,"name":"tranfer_202601_linan_1","icon_uri":""}`,
		got["trace-c-span-c"])
}

func TestWorkflowProvider_UnknownIDStillReturnsID(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:    "trace-d",
			SpanID:     "span-d",
			TagsString: map[string]string{"workflow_id": "111222333444"},
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
			TraceID:    "trace-f",
			SpanID:     "span-f",
			TagsString: map[string]string{"workflow_id": "7647058738179735552"},
		},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"id":7647058738179735552,"name":"","icon_uri":""}`, got["trace-f-span-f"])
}

func TestWorkflowProvider_NilDaoDegrades(t *testing.T) {
	provider := newProviderWithDao(nil)

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{TraceID: "t", SpanID: "s", TagsString: map[string]string{"workflow_id": "7647058738179735552"}},
		{TraceID: "t2", SpanID: "s2", TagsString: map[string]string{"workflow": "legacy"}},
	})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"id":7647058738179735552,"name":"","icon_uri":""}`, got["t-s"])
	assert.Equal(t, "legacy", got["t2-s2"])
}

func TestWorkflowProvider_NoWorkflowTagsSkipped(t *testing.T) {
	provider := newProviderWithDao(&fakeMetaDao{rows: snapshotFixture()})

	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{TraceID: "t", SpanID: "s"},
		nil,
		{TraceID: "t2", SpanID: "s2", TagsString: map[string]string{"workflow_id": "0"}},
		{TraceID: "t3", SpanID: "s3", TagsString: map[string]string{"workflow_id": "not-a-number"}},
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
