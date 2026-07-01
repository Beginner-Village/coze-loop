// Copyright (c) 2025 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"strconv"
	"time"

	"github.com/bytedance/gg/gptr"

	"github.com/coze-dev/coze-loop/backend/infra/middleware/session"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/consts"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/component/rpc"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/domain/entity"
	"github.com/coze-dev/coze-loop/backend/modules/evaluation/pkg/errno"
	"github.com/coze-dev/coze-loop/backend/pkg/errorx"
)

// cozeBotInputFieldKey 是 CozeBot 评测对象的统一输入字段 key，与 CozeWorkflow 保持一致（"input"），
// 便于「批量测试」按同一约定做评测集列 → target 输入映射。
const cozeBotInputFieldKey = "input"

// newTextContent 构造纯文本 Content（CozeBot / CozeWorkflow operator 共用）。
func newTextContent(text string) *entity.Content {
	t := text
	return &entity.Content{
		ContentType: gptr.Of(entity.ContentTypeText),
		Format:      gptr.Of(entity.Markdown),
		Text:        &t,
	}
}

func NewCozeBotSourceEvalTargetServiceImpl(cozeAdapter rpc.ICozeTargetRPCAdapter) ISourceEvalTargetOperateService {
	return &CozeBotSourceEvalTargetServiceImpl{cozeAdapter: cozeAdapter}
}

type CozeBotSourceEvalTargetServiceImpl struct {
	cozeAdapter rpc.ICozeTargetRPCAdapter
}

func (t *CozeBotSourceEvalTargetServiceImpl) EvalType() entity.EvalTargetType {
	return entity.EvalTargetTypeCozeBot
}

func (t *CozeBotSourceEvalTargetServiceImpl) RuntimeParam() entity.IRuntimeParam {
	return entity.NewDummyRuntimeParam()
}

func (t *CozeBotSourceEvalTargetServiceImpl) AsyncExecute(ctx context.Context, spaceID int64, param *entity.ExecuteEvalTargetParam) (int64, string, map[string]string, error) {
	return 0, "", nil, errorx.New("async execute not supported")
}

func (t *CozeBotSourceEvalTargetServiceImpl) ValidateInput(ctx context.Context, spaceID int64, inputSchema []*entity.ArgsSchema, input *entity.EvalTargetInputData) error {
	if input == nil {
		return nil
	}
	return input.ValidateInputSchema(inputSchema)
}

func (t *CozeBotSourceEvalTargetServiceImpl) Execute(ctx context.Context, spaceID int64, param *entity.ExecuteEvalTargetParam) (outputData *entity.EvalTargetOutputData, status entity.EvalTargetRunStatus, err error) {
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

	query := extractUserQuery(param.Input)
	userID := session.UserIDInCtxOrEmpty(ctx)

	res, err := t.cozeAdapter.RunAgent(ctx, spaceID, &rpc.RunAgentParam{
		BotID:   param.SourceTargetID,
		Version: param.SourceTargetVersion,
		UserID:  userID,
		Query:   query,
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

// extractUserQuery 从评测集当前行取用户提问：优先 prompt user_query 约定 key，否则取首个非空文本字段。
func extractUserQuery(input *entity.EvalTargetInputData) string {
	if input == nil {
		return ""
	}
	if c, ok := input.InputFields[cozeBotInputFieldKey]; ok && c != nil && c.GetText() != "" {
		return c.GetText()
	}
	if c, ok := input.InputFields[consts.EvalTargetInputFieldKeyPromptUserQuery]; ok && c != nil {
		return c.GetText()
	}
	for _, c := range input.InputFields {
		if c != nil && c.GetText() != "" {
			return c.GetText()
		}
	}
	return ""
}

func (t *CozeBotSourceEvalTargetServiceImpl) BuildBySource(ctx context.Context, spaceID int64, sourceTargetID, sourceTargetVersion string, opts ...entity.Option) (*entity.EvalTarget, error) {
	botID, err := strconv.ParseInt(sourceTargetID, 10, 64)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.CommonInvalidParamCode, errorx.WithExtraMsg("bot id must be int64"))
	}
	userID := session.UserIDInCtxOrEmpty(ctx)
	baseInfo := &entity.BaseInfo{
		CreatedBy: &entity.UserInfo{UserID: gptr.Of(userID)},
		UpdatedBy: &entity.UserInfo{UserID: gptr.Of(userID)},
	}
	do := &entity.EvalTarget{
		SpaceID:        spaceID,
		SourceTargetID: sourceTargetID,
		EvalTargetType: entity.EvalTargetTypeCozeBot,
		EvalTargetVersion: &entity.EvalTargetVersion{
			SpaceID:             spaceID,
			SourceTargetVersion: sourceTargetVersion,
			EvalTargetType:      entity.EvalTargetTypeCozeBot,
			CozeBot: &entity.CozeBot{
				BotID:       botID,
				BotVersion:  sourceTargetVersion,
				BotInfoType: entity.CozeBotInfoTypeDraftBot,
			},
			InputSchema: []*entity.ArgsSchema{
				{
					Key:                 gptr.Of(cozeBotInputFieldKey),
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

func (t *CozeBotSourceEvalTargetServiceImpl) BatchGetSource(ctx context.Context, spaceID int64, ids []string) (targets []*entity.EvalTarget, err error) {
	targets = make([]*entity.EvalTarget, 0, len(ids))
	for _, id := range ids {
		botID, parseErr := strconv.ParseInt(id, 10, 64)
		if parseErr != nil {
			continue
		}
		targets = append(targets, &entity.EvalTarget{
			SpaceID:        spaceID,
			SourceTargetID: id,
			EvalTargetType: entity.EvalTargetTypeCozeBot,
			EvalTargetVersion: &entity.EvalTargetVersion{
				SpaceID:        spaceID,
				EvalTargetType: entity.EvalTargetTypeCozeBot,
				CozeBot:        &entity.CozeBot{BotID: botID},
			},
		})
	}
	return targets, nil
}

func (t *CozeBotSourceEvalTargetServiceImpl) ListSource(ctx context.Context, param *entity.ListSourceParam) (targets []*entity.EvalTarget, nextCursor string, hasMore bool, err error) {
	// 智能体列表由 studio 前端自身接口(GetDraftIntelligenceList)驱动选择，loop 侧不再重复分页。
	return nil, "", false, nil
}

func (t *CozeBotSourceEvalTargetServiceImpl) ListSourceVersion(ctx context.Context, param *entity.ListSourceVersionParam) (versions []*entity.EvalTargetVersion, nextCursor string, hasMore bool, err error) {
	return nil, "", false, nil
}

func (t *CozeBotSourceEvalTargetServiceImpl) PackSourceInfo(ctx context.Context, spaceID int64, dos []*entity.EvalTarget) (err error) {
	return nil
}

func (t *CozeBotSourceEvalTargetServiceImpl) PackSourceVersionInfo(ctx context.Context, spaceID int64, dos []*entity.EvalTarget) (err error) {
	return nil
}

func (t *CozeBotSourceEvalTargetServiceImpl) SearchCustomEvalTarget(ctx context.Context, param *entity.SearchCustomEvalTargetParam) (targets []*entity.CustomEvalTarget, nextCursor string, hasMore bool, err error) {
	return nil, "", false, nil
}
