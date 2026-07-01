// Copyright (c) 2025 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coze-dev/coze-loop/backend/modules/evaluation/consts"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/component/rpc"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/entity"
)

// fakeCozeAdapter 是 rpc.ICozeTargetRPCAdapter 的手写测试替身，供 CozeBot/CozeWorkflow operator 测试共用。
type fakeCozeAdapter struct {
	runWorkflowFn func(ctx context.Context, spaceID int64, param *rpc.RunWorkflowParam) (*rpc.RunTargetResult, error)
	runAgentFn    func(ctx context.Context, spaceID int64, param *rpc.RunAgentParam) (*rpc.RunTargetResult, error)
}

func (f *fakeCozeAdapter) RunWorkflow(ctx context.Context, spaceID int64, param *rpc.RunWorkflowParam) (*rpc.RunTargetResult, error) {
	return f.runWorkflowFn(ctx, spaceID, param)
}

func (f *fakeCozeAdapter) RunAgent(ctx context.Context, spaceID int64, param *rpc.RunAgentParam) (*rpc.RunTargetResult, error) {
	return f.runAgentFn(ctx, spaceID, param)
}

func TestCozeWorkflow_EvalType(t *testing.T) {
	op := NewCozeWorkflowSourceEvalTargetServiceImpl(&fakeCozeAdapter{})
	assert.Equal(t, entity.EvalTargetTypeCozeWorkflow, op.EvalType())
}

func TestCozeWorkflow_Execute_Success(t *testing.T) {
	var gotParam *rpc.RunWorkflowParam
	op := NewCozeWorkflowSourceEvalTargetServiceImpl(&fakeCozeAdapter{
		runWorkflowFn: func(ctx context.Context, spaceID int64, param *rpc.RunWorkflowParam) (*rpc.RunTargetResult, error) {
			gotParam = param
			return &rpc.RunTargetResult{Output: "wf-out", TotalTokens: 5}, nil
		},
	})

	out, status, err := op.Execute(context.Background(), 100, &entity.ExecuteEvalTargetParam{
		SourceTargetID:      "7412",
		SourceTargetVersion: "v1",
		Input: &entity.EvalTargetInputData{
			InputFields: map[string]*entity.Content{"question": newTextContent("hi")},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, entity.EvalTargetRunStatusSuccess, status)
	assert.Equal(t, "7412", gotParam.WorkflowID)
	assert.Equal(t, "hi", gotParam.Parameters["question"])
	assert.Equal(t, "wf-out", out.OutputFields[consts.OutputSchemaKey].GetText())
	require.NotNil(t, out.EvalTargetUsage)
	assert.Equal(t, int64(5), out.EvalTargetUsage.TotalTokens)
	require.NotNil(t, out.TimeConsumingMS)
}

func TestCozeWorkflow_Execute_Error(t *testing.T) {
	op := NewCozeWorkflowSourceEvalTargetServiceImpl(&fakeCozeAdapter{
		runWorkflowFn: func(ctx context.Context, spaceID int64, param *rpc.RunWorkflowParam) (*rpc.RunTargetResult, error) {
			return nil, errors.New("boom")
		},
	})
	out, status, err := op.Execute(context.Background(), 1, &entity.ExecuteEvalTargetParam{
		SourceTargetID: "7",
		Input:          &entity.EvalTargetInputData{InputFields: map[string]*entity.Content{}},
	})
	require.Error(t, err)
	assert.Equal(t, entity.EvalTargetRunStatusFail, status)
	require.NotNil(t, out.EvalTargetRunError)
	assert.Contains(t, out.EvalTargetRunError.Message, "boom")
}

func TestCozeWorkflow_BuildBySource(t *testing.T) {
	op := NewCozeWorkflowSourceEvalTargetServiceImpl(&fakeCozeAdapter{})
	do, err := op.BuildBySource(context.Background(), 100, "7412", "v2")
	require.NoError(t, err)
	assert.Equal(t, entity.EvalTargetTypeCozeWorkflow, do.EvalTargetType)
	assert.Equal(t, "7412", do.SourceTargetID)
	require.NotNil(t, do.EvalTargetVersion.CozeWorkflow)
	assert.Equal(t, "7412", do.EvalTargetVersion.CozeWorkflow.ID)
	assert.Equal(t, "v2", do.EvalTargetVersion.CozeWorkflow.Version)
	assert.NotEmpty(t, do.EvalTargetVersion.InputSchema)
	assert.NotEmpty(t, do.EvalTargetVersion.OutputSchema)
}

func TestCozeWorkflow_BuildBySource_EmptyID(t *testing.T) {
	op := NewCozeWorkflowSourceEvalTargetServiceImpl(&fakeCozeAdapter{})
	_, err := op.BuildBySource(context.Background(), 1, "", "")
	require.Error(t, err)
}
