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

func TestCozeBot_EvalType(t *testing.T) {
	op := NewCozeBotSourceEvalTargetServiceImpl(&fakeCozeAdapter{})
	assert.Equal(t, entity.EvalTargetTypeCozeBot, op.EvalType())
}

func TestCozeBot_Execute_Success(t *testing.T) {
	var gotParam *rpc.RunAgentParam
	op := NewCozeBotSourceEvalTargetServiceImpl(&fakeCozeAdapter{
		runAgentFn: func(ctx context.Context, spaceID int64, param *rpc.RunAgentParam) (*rpc.RunTargetResult, error) {
			gotParam = param
			return &rpc.RunTargetResult{Output: "bot-answer", TotalTokens: 8}, nil
		},
	})

	out, status, err := op.Execute(context.Background(), 100, &entity.ExecuteEvalTargetParam{
		SourceTargetID: "7412",
		Input: &entity.EvalTargetInputData{
			InputFields: map[string]*entity.Content{
				consts.EvalTargetInputFieldKeyPromptUserQuery: newTextContent("你好"),
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, entity.EvalTargetRunStatusSuccess, status)
	assert.Equal(t, "7412", gotParam.BotID)
	assert.Equal(t, "你好", gotParam.Query)
	assert.Equal(t, "bot-answer", out.OutputFields[consts.OutputSchemaKey].GetText())
	require.NotNil(t, out.EvalTargetUsage)
	assert.Equal(t, int64(8), out.EvalTargetUsage.TotalTokens)
}

func TestCozeBot_Execute_FallbackQuery(t *testing.T) {
	var gotParam *rpc.RunAgentParam
	op := NewCozeBotSourceEvalTargetServiceImpl(&fakeCozeAdapter{
		runAgentFn: func(ctx context.Context, spaceID int64, param *rpc.RunAgentParam) (*rpc.RunTargetResult, error) {
			gotParam = param
			return &rpc.RunTargetResult{Output: "ok"}, nil
		},
	})
	// 无 user_query 约定 key 时，取首个非空文本字段
	_, status, err := op.Execute(context.Background(), 1, &entity.ExecuteEvalTargetParam{
		SourceTargetID: "7",
		Input:          &entity.EvalTargetInputData{InputFields: map[string]*entity.Content{"col": newTextContent("单列问题")}},
	})
	require.NoError(t, err)
	assert.Equal(t, entity.EvalTargetRunStatusSuccess, status)
	assert.Equal(t, "单列问题", gotParam.Query)
}

func TestCozeBot_Execute_InputKey(t *testing.T) {
	var gotParam *rpc.RunAgentParam
	op := NewCozeBotSourceEvalTargetServiceImpl(&fakeCozeAdapter{
		runAgentFn: func(ctx context.Context, spaceID int64, param *rpc.RunAgentParam) (*rpc.RunTargetResult, error) {
			gotParam = param
			return &rpc.RunTargetResult{Output: "ok"}, nil
		},
	})
	// 统一 "input" key（与 CozeWorkflow 一致，批量测试列映射用）
	_, status, err := op.Execute(context.Background(), 1, &entity.ExecuteEvalTargetParam{
		SourceTargetID: "7",
		Input:          &entity.EvalTargetInputData{InputFields: map[string]*entity.Content{cozeBotInputFieldKey: newTextContent("统一入参")}},
	})
	require.NoError(t, err)
	assert.Equal(t, entity.EvalTargetRunStatusSuccess, status)
	assert.Equal(t, "统一入参", gotParam.Query)
}

func TestCozeBot_Execute_Error(t *testing.T) {
	op := NewCozeBotSourceEvalTargetServiceImpl(&fakeCozeAdapter{
		runAgentFn: func(ctx context.Context, spaceID int64, param *rpc.RunAgentParam) (*rpc.RunTargetResult, error) {
			return nil, errors.New("agent down")
		},
	})
	out, status, err := op.Execute(context.Background(), 1, &entity.ExecuteEvalTargetParam{
		SourceTargetID: "7",
		Input:          &entity.EvalTargetInputData{InputFields: map[string]*entity.Content{}},
	})
	require.Error(t, err)
	assert.Equal(t, entity.EvalTargetRunStatusFail, status)
	require.NotNil(t, out.EvalTargetRunError)
	assert.Contains(t, out.EvalTargetRunError.Message, "agent down")
}

func TestCozeBot_BuildBySource(t *testing.T) {
	op := NewCozeBotSourceEvalTargetServiceImpl(&fakeCozeAdapter{})
	do, err := op.BuildBySource(context.Background(), 100, "7412", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, entity.EvalTargetTypeCozeBot, do.EvalTargetType)
	require.NotNil(t, do.EvalTargetVersion.CozeBot)
	assert.Equal(t, int64(7412), do.EvalTargetVersion.CozeBot.BotID)
	assert.Equal(t, "1.0.0", do.EvalTargetVersion.CozeBot.BotVersion)
	require.Len(t, do.EvalTargetVersion.InputSchema, 1)
	assert.Equal(t, cozeBotInputFieldKey, *do.EvalTargetVersion.InputSchema[0].Key)
}

func TestCozeBot_BuildBySource_InvalidID(t *testing.T) {
	op := NewCozeBotSourceEvalTargetServiceImpl(&fakeCozeAdapter{})
	_, err := op.BuildBySource(context.Background(), 1, "not-a-number", "")
	require.Error(t, err)
}
