// Copyright (c) 2026 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package workflow

import (
	"context"
	"testing"

	"github.com/coze-dev/coze-loop/backend/modules/observability/domain/trace/entity/loop_span"
	"github.com/stretchr/testify/assert"
)

func TestNewWorkflowProvider(t *testing.T) {
	provider := NewWorkflowProvider()
	assert.NotNil(t, provider)
	assert.Implements(t, (*interface {
		BatchGetWorkflows(context.Context, loop_span.SpanList) (map[string]string, error)
	})(nil), provider)
}

func TestWorkflowProvider_BatchGetWorkflows(t *testing.T) {
	provider := NewWorkflowProvider()
	ctx := context.Background()

	tests := []struct {
		name     string
		spans    loop_span.SpanList
		wantErr  bool
		wantSize int
	}{
		{
			name:     "empty spans list",
			spans:    loop_span.SpanList{},
			wantErr:  false,
			wantSize: 0,
		},
		{
			name:     "nil spans list",
			spans:    nil,
			wantErr:  false,
			wantSize: 0,
		},
		{
			name: "spans with need workflow",
			spans: loop_span.SpanList{
				{
					TraceID:     "trace-1",
					SpanID:      "span-1",
					WorkspaceID: "ws-1",
					Encryption: loop_span.EncryptionInfo{
						NeedWorkflow: true,
					},
				},
				{
					TraceID:     "trace-2",
					SpanID:      "span-2",
					WorkspaceID: "ws-2",
					Encryption: loop_span.EncryptionInfo{
						NeedWorkflow: true,
					},
				},
			},
			wantErr:  false,
			wantSize: 0, // need workflow 但 span 自身无 workflow tag，跳过
		},
		{
			name: "span with need workflow and workflow tag is extracted",
			spans: loop_span.SpanList{
				{
					TraceID:     "trace-9",
					SpanID:      "span-9",
					WorkspaceID: "ws-9",
					TagsString:  map[string]string{"workflow": "wf-payload-9"},
					Encryption: loop_span.EncryptionInfo{
						NeedWorkflow: true,
					},
				},
				{
					// need workflow 但无 tag -> 跳过
					TraceID:     "trace-10",
					SpanID:      "span-10",
					WorkspaceID: "ws-10",
					Encryption: loop_span.EncryptionInfo{
						NeedWorkflow: true,
					},
				},
				{
					// 有 tag 但不 need workflow -> 跳过
					TraceID:     "trace-11",
					SpanID:      "span-11",
					WorkspaceID: "ws-11",
					TagsString:  map[string]string{"workflow": "wf-payload-11"},
					Encryption: loop_span.EncryptionInfo{
						NeedWorkflow: false,
					},
				},
			},
			wantErr:  false,
			wantSize: 1,
		},
		{
			name: "spans without need workflow",
			spans: loop_span.SpanList{
				{
					TraceID:     "trace-1",
					SpanID:      "span-1",
					WorkspaceID: "ws-1",
					Encryption: loop_span.EncryptionInfo{
						NeedWorkflow: false,
					},
				},
			},
			wantErr:  false,
			wantSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := provider.BatchGetWorkflows(ctx, tt.spans)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tt.wantSize, len(got))
			}
		})
	}
}

func TestWorkflowProvider_BatchGetWorkflows_KeyAndValue(t *testing.T) {
	provider := NewWorkflowProvider()
	got, err := provider.BatchGetWorkflows(context.Background(), loop_span.SpanList{
		{
			TraceID:    "trace-1",
			SpanID:     "span-1",
			TagsString: map[string]string{"workflow": "wf-payload"},
			Encryption: loop_span.EncryptionInfo{NeedWorkflow: true},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"trace-1-span-1": "wf-payload"}, got)
}
