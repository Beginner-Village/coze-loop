// Copyright (c) 2025 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coze-dev/coze-loop/backend/modules/evaluation/consts"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/entity"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/infra/rpc/cozestudio"
)

// 真实环境集成测试：用与部署到 loop 相同的 operator + cozestudio adapter，
// 打通 live studio OpenAPI 跑一个真实工作流，验证批量测试工作流会真正执行并回填输出。
// 仅当设置 YNET_STUDIO_OPENAPI_BASE_URL / TOKEN / TEST_WORKFLOW_ID 时运行，否则跳过。
//
// 运行示例（对 226）：
//   COZE_LOOP_SESSION_HMAC_KEY=x COZE_LOOP_STUDIO_HMAC_KEY=y \
//   YNET_STUDIO_OPENAPI_BASE_URL=http://10.10.10.226:8896 \
//   YNET_STUDIO_OPENAPI_TOKEN=<pat> TEST_WORKFLOW_ID=7655741277509517312 \
//   go test ./modules/evaluation/domain/service/ -run TestIntegration_CozeWorkflow_Live -v -count=1
func TestIntegration_CozeWorkflow_Live(t *testing.T) {
	base := os.Getenv("YNET_STUDIO_OPENAPI_BASE_URL")
	token := os.Getenv("YNET_STUDIO_OPENAPI_TOKEN")
	workflowID := os.Getenv("TEST_WORKFLOW_ID")
	if base == "" || token == "" || workflowID == "" {
		t.Skip("set YNET_STUDIO_OPENAPI_BASE_URL / YNET_STUDIO_OPENAPI_TOKEN / TEST_WORKFLOW_ID to run live integration test")
	}

	adapter := cozestudio.NewCozeTargetRPCAdapter(base, token, "ynet_eval", nil)
	op := NewCozeWorkflowSourceEvalTargetServiceImpl(adapter)

	out, status, err := op.Execute(context.Background(), 0, &entity.ExecuteEvalTargetParam{
		SourceTargetID: workflowID,
		Input:          &entity.EvalTargetInputData{InputFields: map[string]*entity.Content{}},
	})

	require.NoError(t, err)
	assert.Equal(t, entity.EvalTargetRunStatusSuccess, status)
	require.NotNil(t, out.OutputFields[consts.OutputSchemaKey])
	outText := out.OutputFields[consts.OutputSchemaKey].GetText()
	t.Logf("workflow live output = %s", outText)
	assert.NotEmpty(t, outText, "工作流应返回非空输出")
	// demo2 (queryBalance) 输出含「余额」/balance
	assert.True(t,
		strings.Contains(outText, "余额") || strings.Contains(strings.ToLower(outText), "balance"),
		"输出应包含余额信息，实际=%s", outText)
}
