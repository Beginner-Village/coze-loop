// Copyright (c) 2025 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

// Package cozestudio 通过 coze-studio 的 OpenAPI（Bearer PAT 鉴权）执行智能体/工作流，
// 为 loop 的 CozeBot / CozeWorkflow 评测对象提供真实运行能力。
package cozestudio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/component/rpc"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/pkg/errno"
	"github.com/coze-dev/coze-loop/backend/pkg/errorx"
)

const (
	envBaseURL = "YNET_STUDIO_OPENAPI_BASE_URL"
	envToken   = "YNET_STUDIO_OPENAPI_TOKEN"
	envUserID  = "YNET_STUDIO_OPENAPI_USER_ID"

	defaultUserID  = "coze_loop_eval"
	defaultTimeout = 180 * time.Second

	pathWorkflowRun = "/v1/workflow/run"
	pathAgentChat   = "/v3/chat"
)

type CozeTargetRPCAdapter struct {
	baseURL string
	token   string
	userID  string
	client  *http.Client
}

// NewCozeTargetRPCAdapterFromEnv 从环境变量装配 studio OpenAPI 客户端。
func NewCozeTargetRPCAdapterFromEnv() rpc.ICozeTargetRPCAdapter {
	return NewCozeTargetRPCAdapter(
		os.Getenv(envBaseURL),
		os.Getenv(envToken),
		os.Getenv(envUserID),
		&http.Client{Timeout: defaultTimeout},
	)
}

// NewCozeTargetRPCAdapter 显式构造（便于测试注入 httptest server 与 client）。
func NewCozeTargetRPCAdapter(baseURL, token, userID string, client *http.Client) rpc.ICozeTargetRPCAdapter {
	if client == nil {
		client = &http.Client{Timeout: defaultTimeout}
	}
	return &CozeTargetRPCAdapter{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		token:   strings.TrimSpace(token),
		userID:  strings.TrimSpace(userID),
		client:  client,
	}
}

type workflowRunResp struct {
	Code int64   `json:"code"`
	Msg  string  `json:"msg"`
	Data *string `json:"data"`
	Token *int64 `json:"token"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

type chatUsage struct {
	TokenCount  int64 `json:"token_count"`
	InputCount  int64 `json:"input_count"`
	OutputCount int64 `json:"output_count"`
}

type chatNoStreamResp struct {
	Code     int64          `json:"code"`
	Msg      string         `json:"msg"`
	Messages []*chatMessage `json:"messages"`
	Usage    *chatUsage     `json:"usage"`
}

func (a *CozeTargetRPCAdapter) RunWorkflow(ctx context.Context, spaceID int64, param *rpc.RunWorkflowParam) (*rpc.RunTargetResult, error) {
	if param == nil || param.WorkflowID == "" {
		return nil, errorx.NewByCode(errno.CommonInvalidParamCode, errorx.WithExtraMsg("workflow id is empty"))
	}
	paramsJSON := "{}"
	if len(param.Parameters) > 0 {
		pb, err := json.Marshal(param.Parameters)
		if err != nil {
			return nil, errorx.WrapByCode(err, errno.CommonInternalErrorCode)
		}
		paramsJSON = string(pb)
	}
	reqBody := map[string]any{
		"workflow_id": param.WorkflowID,
		"parameters":  paramsJSON,
		"is_async":    false,
	}
	if param.Version != "" {
		reqBody["version"] = param.Version
	}
	body, err := a.doPost(ctx, pathWorkflowRun, reqBody)
	if err != nil {
		return nil, err
	}
	var r workflowRunResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, errorx.WrapByCode(err, errno.CommonRPCErrorCode, errorx.WithExtraMsg("decode workflow run resp"))
	}
	if r.Code != 0 {
		return nil, errorx.NewByCode(errno.CommonRPCErrorCode, errorx.WithExtraMsg(fmt.Sprintf("studio workflow run failed: code=%d msg=%s", r.Code, r.Msg)))
	}
	res := &rpc.RunTargetResult{}
	if r.Data != nil {
		res.Output = *r.Data
	}
	if r.Token != nil {
		res.TotalTokens = *r.Token
	}
	return res, nil
}

func (a *CozeTargetRPCAdapter) RunAgent(ctx context.Context, spaceID int64, param *rpc.RunAgentParam) (*rpc.RunTargetResult, error) {
	if param == nil || param.BotID == "" {
		return nil, errorx.NewByCode(errno.CommonInvalidParamCode, errorx.WithExtraMsg("bot id is empty"))
	}
	userID := param.UserID
	if userID == "" {
		userID = a.userID
	}
	if userID == "" {
		userID = defaultUserID
	}
	reqBody := map[string]any{
		"bot_id":  param.BotID,
		"user_id": userID,
		"stream":  false,
		"additional_messages": []map[string]string{
			{"role": "user", "content": param.Query, "content_type": "text"},
		},
	}
	body, err := a.doPost(ctx, pathAgentChat, reqBody)
	if err != nil {
		return nil, err
	}
	var r chatNoStreamResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, errorx.WrapByCode(err, errno.CommonRPCErrorCode, errorx.WithExtraMsg("decode agent chat resp"))
	}
	if r.Code != 0 {
		return nil, errorx.NewByCode(errno.CommonRPCErrorCode, errorx.WithExtraMsg(fmt.Sprintf("studio agent chat failed: code=%d msg=%s", r.Code, r.Msg)))
	}
	var sb strings.Builder
	for _, m := range r.Messages {
		if m == nil {
			continue
		}
		if m.Role == "assistant" && m.Type == "answer" {
			sb.WriteString(m.Content)
		}
	}
	res := &rpc.RunTargetResult{Output: sb.String()}
	if r.Usage != nil {
		res.InputTokens = r.Usage.InputCount
		res.OutputTokens = r.Usage.OutputCount
		res.TotalTokens = r.Usage.TokenCount
	}
	return res, nil
}

func (a *CozeTargetRPCAdapter) doPost(ctx context.Context, path string, reqBody any) ([]byte, error) {
	if a.baseURL == "" || a.token == "" {
		return nil, errorx.NewByCode(errno.CommonInvalidParamCode,
			errorx.WithExtraMsg("studio openapi not configured (set "+envBaseURL+" and "+envToken+")"))
	}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.CommonInternalErrorCode)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.CommonInternalErrorCode)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.token)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.CommonRPCErrorCode, errorx.WithExtraMsg("call studio openapi"))
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.CommonRPCErrorCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errorx.NewByCode(errno.CommonRPCErrorCode,
			errorx.WithExtraMsg(fmt.Sprintf("studio openapi http %d: %s", resp.StatusCode, truncate(respBody, 512))))
	}
	return respBody, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
