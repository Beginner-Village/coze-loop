// Copyright (c) 2025 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package cozestudio

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/component/rpc"
)

func TestRunWorkflow_Success(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"code":0,"msg":"Success","data":"hello-output","token":12}`))
	}))
	defer srv.Close()

	a := NewCozeTargetRPCAdapter(srv.URL, "tk", "u1", srv.Client())
	res, err := a.RunWorkflow(context.Background(), 100, &rpc.RunWorkflowParam{
		WorkflowID: "7412",
		Version:    "v1",
		Parameters: map[string]string{"question": "hi"},
	})
	require.NoError(t, err)
	assert.Equal(t, "/v1/workflow/run", gotPath)
	assert.Equal(t, "Bearer tk", gotAuth)
	assert.Equal(t, "7412", gotBody["workflow_id"])
	assert.Equal(t, `{"question":"hi"}`, gotBody["parameters"])
	assert.Equal(t, "hello-output", res.Output)
	assert.Equal(t, int64(12), res.TotalTokens)
}

func TestRunWorkflow_BizError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":700012,"msg":"workflow not published"}`))
	}))
	defer srv.Close()

	a := NewCozeTargetRPCAdapter(srv.URL, "tk", "", srv.Client())
	_, err := a.RunWorkflow(context.Background(), 1, &rpc.RunWorkflowParam{WorkflowID: "7"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not published")
}

func TestRunAgent_Success(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		// 混入非 answer 消息，验证只取 assistant+answer
		_, _ = w.Write([]byte(`{"code":0,"messages":[` +
			`{"role":"assistant","type":"verbose","content":"debug"},` +
			`{"role":"assistant","type":"answer","content":"the answer"}],` +
			`"usage":{"token_count":9,"input_count":4,"output_count":5}}`))
	}))
	defer srv.Close()

	a := NewCozeTargetRPCAdapter(srv.URL, "tk", "", srv.Client())
	res, err := a.RunAgent(context.Background(), 1, &rpc.RunAgentParam{BotID: "7412", Query: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "7412", gotBody["bot_id"])
	assert.Equal(t, false, gotBody["stream"])
	assert.Equal(t, "the answer", res.Output)
	assert.Equal(t, int64(4), res.InputTokens)
	assert.Equal(t, int64(9), res.TotalTokens)
}

func TestRunAgent_DefaultUserID(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		_, _ = w.Write([]byte(`{"code":0,"messages":[{"role":"assistant","type":"answer","content":"x"}]}`))
	}))
	defer srv.Close()

	a := NewCozeTargetRPCAdapter(srv.URL, "tk", "", srv.Client())
	_, err := a.RunAgent(context.Background(), 1, &rpc.RunAgentParam{BotID: "7"})
	require.NoError(t, err)
	assert.Equal(t, defaultUserID, gotBody["user_id"])
}

func TestNotConfigured(t *testing.T) {
	a := NewCozeTargetRPCAdapter("", "", "", nil)
	_, err := a.RunWorkflow(context.Background(), 1, &rpc.RunWorkflowParam{WorkflowID: "7"})
	require.Error(t, err)
	_, err = a.RunAgent(context.Background(), 1, &rpc.RunAgentParam{BotID: "7"})
	require.Error(t, err)
}
