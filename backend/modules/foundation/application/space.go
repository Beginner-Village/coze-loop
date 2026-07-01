// Copyright (c) 2025 ynet Authors
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"context"
	"strconv"
	"strings"

	"github.com/coze-dev/coze-loop/backend/infra/middleware/session"
	domainspace "github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/foundation/domain/space"
	"github.com/coze-dev/coze-loop/backend/kitex_gen/coze/loop/foundation/space"
	"github.com/coze-dev/coze-loop/backend/modules/foundation/application/convertor"
	"github.com/coze-dev/coze-loop/backend/modules/foundation/domain/user/repo"
	"github.com/coze-dev/coze-loop/backend/modules/foundation/pkg/errno"
	"github.com/coze-dev/coze-loop/backend/pkg/errorx"
	"github.com/coze-dev/coze-loop/backend/pkg/lang/ptr"
	"github.com/coze-dev/coze-loop/backend/pkg/lang/slices"
)

type SpaceApplicationImpl struct {
	userRepo repo.IUserRepo
}

func NewSpaceApplication(userRepo repo.IUserRepo) (space.SpaceService, error) {
	return &SpaceApplicationImpl{
		userRepo: userRepo,
	}, nil
}

func (s SpaceApplicationImpl) GetSpace(ctx context.Context, request *space.GetSpaceRequest) (r *space.GetSpaceResponse, err error) {
	r = space.NewGetSpaceResponse()

	if request.GetSpaceID() <= 0 {
		return nil, errorx.NewByCode(errno.CommonInvalidParamCode, errorx.WithExtraMsg("SpaceApplicationImpl.GetSpace invalid param"))
	}

	spaceDO, err := s.userRepo.GetSpaceByID(ctx, request.GetSpaceID())
	if err != nil {
		return nil, err
	}
	r.Space = convertor.SpaceDO2DTO(spaceDO)
	return r, nil
}

func (s SpaceApplicationImpl) ListUserSpaces(ctx context.Context, request *space.ListUserSpaceRequest) (r *space.ListUserSpaceResponse, err error) {
	userIDInCtx := session.UserIDInCtxOrEmpty(ctx)
	if userIDInCtx == "" {
		// 无session时，从请求参数中获取userID
		userIDInCtx = request.GetUserID()
	}

	userID, err := strconv.ParseInt(userIDInCtx, 10, 64)
	if err != nil {
		return nil, errorx.WrapByCode(err, errno.CommonInvalidParamCode, errorx.WithExtraMsg("SpaceApplicationImpl.ListUserSpaces invalid param"))
	}

	spaceDOs, total, err := s.userRepo.ListUserSpace(ctx, userID, request.GetPageSize(), request.GetPageNumber())
	if err != nil {
		return nil, err
	}

	spaces := slices.Map(spaceDOs, convertor.SpaceDO2DTO)
	if len(spaces) == 0 {
		if ctxUser, ok := session.UserInCtx(ctx); ok && isExternalSpaceUser(ctxUser) {
			spaces = []*domainspace.Space{syntheticExternalSpace(ctx, userIDInCtx)}
			total = 1
		} else if spaces == nil {
			spaces = []*domainspace.Space{}
		}
	}

	r = &space.ListUserSpaceResponse{
		Spaces: spaces,
		Total:  ptr.Of(total),
	}
	return r, nil
}

func isExternalSpaceUser(ctxUser *session.User) bool {
	if ctxUser == nil || ctxUser.ID == "" {
		return false
	}
	return ctxUser.IsExternal || strings.HasPrefix(ctxUser.Name, "external-")
}

func syntheticExternalSpace(ctx context.Context, ownerUserID string) *domainspace.Space {
	spaceIDStr := ownerUserID
	if workspaceID, ok := session.ExternalWorkspaceIDInCtx(ctx); ok {
		spaceIDStr = workspaceID
	}
	spaceID, err := strconv.ParseInt(spaceIDStr, 10, 64)
	if err != nil || spaceID <= 0 {
		spaceID, _ = strconv.ParseInt(ownerUserID, 10, 64)
	}
	return &domainspace.Space{
		ID:          spaceID,
		Name:        "Studio Workspace",
		Description: "External Studio workspace",
		SpaceType:   domainspace.SpaceType_Personal,
		OwnerUserID: ownerUserID,
	}
}
