// Copyright (c) 2025 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"time"

	"github.com/bytedance/gg/gptr"

	"github.com/coze-dev/coze-loop/backend/infra/middleware/session"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/consts"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/component/rpc"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/entity"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/pkg/errno"
	"github.com/coze-dev/coze-loop/backend/pkg/errorx"
)

// cozeWorkflowInputFieldKey 工作流评测对象声明的通用输入字段 key。
const cozeWorkflowInputFieldKey = "input"

func NewCozeWorkflowSourceEvalTargetServiceImpl(cozeAdapter rpc.ICozeTargetRPCAdapter) ISourceEvalTargetOperateService {
	return &CozeWorkflowSourceEvalTargetServiceImpl{cozeAdapter: cozeAdapter}
}

type CozeWorkflowSourceEvalTargetServiceImpl struct {
	cozeAdapter rpc.ICozeTargetRPCAdapter
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) EvalType() entity.EvalTargetType {
	return entity.EvalTargetTypeCozeWorkflow
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) RuntimeParam() entity.IRuntimeParam {
	return entity.NewDummyRuntimeParam()
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) AsyncExecute(ctx context.Context, spaceID int64, param *entity.ExecuteEvalTargetParam) (int64, string, map[string]string, error) {
	return 0, "", nil, errorx.New("async execute not supported")
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) ValidateInput(ctx context.Context, spaceID int64, inputSchema []*entity.ArgsSchema, input *entity.EvalTargetInputData) error {
	if input == nil {
		return nil
	}
	return input.ValidateInputSchema(inputSchema)
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) Execute(ctx context.Context, spaceID int64, param *entity.ExecuteEvalTargetParam) (outputData *entity.EvalTargetOutputData, status entity.EvalTargetRunStatus, err error) {
	start := time.Now()
	outputData = &entity.EvalTargetOutputData{}
	defer func() {
		outputData.TimeConsumingMS = gptr.Of(time.Since(start).Milliseconds())
		if err != nil {
			outputData.EvalTargetRunError = &entity.EvalTargetRunError{}
			if statusErr, ok := errorx.FromStatusError(err); ok {
				outputData.EvalTargetRunError.Code = statusErr.Code()
				outputData.EvalTargetRunError.Message = statusErr.Error()
			} else {
				outputData.EvalTargetRunError.Code = errno.CommonInternalErrorCode
				outputData.EvalTargetRunError.Message = err.Error()
			}
		}
	}()

	if param == nil {
		return outputData, entity.EvalTargetRunStatusFail, errorx.NewByCode(errno.CommonInvalidParamCode, errorx.WithExtraMsg("execute param is nil"))
	}

	// 评测集当前行的字段按名透传为工作流入参。
	parameters := make(map[string]string)
	if param.Input != nil {
		for key, content := range param.Input.InputFields {
			if content == nil {
				continue
			}
			parameters[key] = content.GetText()
		}
	}

	res, err := t.cozeAdapter.RunWorkflow(ctx, spaceID, &rpc.RunWorkflowParam{
		WorkflowID: param.SourceTargetID,
		Version:    param.SourceTargetVersion,
		Parameters: parameters,
	})
	if err != nil {
		return outputData, entity.EvalTargetRunStatusFail, err
	}

	outputData.OutputFields = map[string]*entity.Content{
		consts.OutputSchemaKey: newTextContent(res.Output),
	}
	if res.TotalTokens > 0 || res.InputTokens > 0 || res.OutputTokens > 0 {
		outputData.EvalTargetUsage = &entity.EvalTargetUsage{
			InputTokens:  res.InputTokens,
			OutputTokens: res.OutputTokens,
			TotalTokens:  res.TotalTokens,
		}
	}
	return outputData, entity.EvalTargetRunStatusSuccess, nil
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) BuildBySource(ctx context.Context, spaceID int64, sourceTargetID, sourceTargetVersion string, opts ...entity.Option) (*entity.EvalTarget, error) {
	if sourceTargetID == "" {
		return nil, errorx.NewByCode(errno.CommonInvalidParamCode, errorx.WithExtraMsg("workflow id is empty"))
	}
	userID := session.UserIDInCtxOrEmpty(ctx)
	baseInfo := &entity.BaseInfo{
		CreatedBy: &entity.UserInfo{UserID: gptr.Of(userID)},
		UpdatedBy: &entity.UserInfo{UserID: gptr.Of(userID)},
	}
	do := &entity.EvalTarget{
		SpaceID:        spaceID,
		SourceTargetID: sourceTargetID,
		EvalTargetType: entity.EvalTargetTypeCozeWorkflow,
		EvalTargetVersion: &entity.EvalTargetVersion{
			SpaceID:             spaceID,
			SourceTargetVersion: sourceTargetVersion,
			EvalTargetType:      entity.EvalTargetTypeCozeWorkflow,
			CozeWorkflow: &entity.CozeWorkflow{
				ID:      sourceTargetID,
				Version: sourceTargetVersion,
			},
			// 声明单个通用输入字段 "input"：实验按「目标输入字段 × 评测集行」展开 turn，
			// 需要至少一个输入字段才能生成 turn。前端提交时把该字段映射到评测集同名列
			// (target_field_mapping)，Execute 再把收到的字段按名透传为工作流入参。
			InputSchema: []*entity.ArgsSchema{
				{
					Key:                 gptr.Of(cozeWorkflowInputFieldKey),
					SupportContentTypes: []entity.ContentType{entity.ContentTypeText},
					JsonSchema:          gptr.Of(consts.StringJsonSchema),
				},
			},
			OutputSchema: []*entity.ArgsSchema{
				{
					Key:                 gptr.Of(consts.OutputSchemaKey),
					SupportContentTypes: []entity.ContentType{entity.ContentTypeText},
					JsonSchema:          gptr.Of(consts.StringJsonSchema),
				},
			},
			RuntimeParamDemo: gptr.Of(entity.NewDummyRuntimeParam().GetJSONDemo()),
			BaseInfo:         baseInfo,
		},
		BaseInfo: baseInfo,
	}
	return do, nil
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) BatchGetSource(ctx context.Context, spaceID int64, ids []string) (targets []*entity.EvalTarget, err error) {
	targets = make([]*entity.EvalTarget, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		targets = append(targets, &entity.EvalTarget{
			SpaceID:        spaceID,
			SourceTargetID: id,
			EvalTargetType: entity.EvalTargetTypeCozeWorkflow,
			EvalTargetVersion: &entity.EvalTargetVersion{
				SpaceID:        spaceID,
				EvalTargetType: entity.EvalTargetTypeCozeWorkflow,
				CozeWorkflow:   &entity.CozeWorkflow{ID: id},
			},
		})
	}
	return targets, nil
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) ListSource(ctx context.Context, param *entity.ListSourceParam) (targets []*entity.EvalTarget, nextCursor string, hasMore bool, err error) {
	// 工作流列表由 studio 前端自身接口(GetWorkFlowList)驱动选择，loop 侧不再重复分页。
	return nil, "", false, nil
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) ListSourceVersion(ctx context.Context, param *entity.ListSourceVersionParam) (versions []*entity.EvalTargetVersion, nextCursor string, hasMore bool, err error) {
	return nil, "", false, nil
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) PackSourceInfo(ctx context.Context, spaceID int64, dos []*entity.EvalTarget) (err error) {
	return nil
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) PackSourceVersionInfo(ctx context.Context, spaceID int64, dos []*entity.EvalTarget) (err error) {
	return nil
}

func (t *CozeWorkflowSourceEvalTargetServiceImpl) SearchCustomEvalTarget(ctx context.Context, param *entity.SearchCustomEvalTargetParam) (targets []*entity.CustomEvalTarget, nextCursor string, hasMore bool, err error) {
	return nil, "", false, nil
}
