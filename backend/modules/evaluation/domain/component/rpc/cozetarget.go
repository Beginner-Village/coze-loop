// Copyright (c) 2025 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"context"

	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/entity"
)

// ICozeTargetRPCAdapter 调用 coze-studio OpenAPI 执行「智能体(bot)」「工作流(workflow)」，
// 供 CozeBot / CozeWorkflow 两类评测对象的 operator 在批量评测时真实运行目标对象。
//
//go:generate mockgen -destination=mocks/cozetarget.go -package=mocks . ICozeTargetRPCAdapter
type ICozeTargetRPCAdapter interface {
	// RunWorkflow 同步运行工作流（POST /v1/workflow/run），返回工作流输出。
	RunWorkflow(ctx context.Context, spaceID int64, param *RunWorkflowParam) (*RunTargetResult, error)
	// RunAgent 非流式运行智能体（POST /v3/chat, stream=false），返回智能体最终回答。
	RunAgent(ctx context.Context, spaceID int64, param *RunAgentParam) (*RunTargetResult, error)
}

type RunWorkflowParam struct {
	WorkflowID string
	Version    string
	// Parameters 工作流入参：评测集当前行的字段(key->文本)按名透传给工作流。
	Parameters map[string]string
}

type RunAgentParam struct {
	BotID   string
	Version string
	// UserID 数据隔离用户标识；为空时由 adapter 兜底。
	UserID string
	// Query 用户提问文本。
	Query string
	// History 可选历史消息。
	History []*entity.Message
}

type RunTargetResult struct {
	// Output 目标对象的输出文本（工作流为 data，智能体为 answer 消息拼接）。
	Output       string
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
}
