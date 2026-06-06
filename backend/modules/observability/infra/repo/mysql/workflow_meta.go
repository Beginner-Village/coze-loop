// Copyright (c) 2026 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package mysql

import (
	"context"

	"github.com/coze-dev/coze-loop/backend/infra/db"
	"github.com/coze-dev/coze-loop/backend/modules/observability/infra/repo/mysql/gorm_gen/model"
	obErrorx "github.com/coze-dev/coze-loop/backend/modules/observability/pkg/errno"
	"github.com/coze-dev/coze-loop/backend/pkg/errorx"
)

//go:generate mockgen -destination=mocks/workflow_meta.go -package=mocks . IWorkflowMetaDao
type IWorkflowMetaDao interface {
	// BatchGet 按工作流ID批量查询元数据快照, 返回 id -> meta 映射, 缺失的 id 不会出现在结果里.
	BatchGet(ctx context.Context, ids []int64) (map[int64]*model.ObservabilityWorkflowMeta, error)
}

func NewWorkflowMetaDaoImpl(db db.Provider) IWorkflowMetaDao {
	return &WorkflowMetaDaoImpl{dbMgr: db}
}

type WorkflowMetaDaoImpl struct {
	dbMgr db.Provider
}

func (w *WorkflowMetaDaoImpl) BatchGet(ctx context.Context, ids []int64) (map[int64]*model.ObservabilityWorkflowMeta, error) {
	result := make(map[int64]*model.ObservabilityWorkflowMeta, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	var rows []*model.ObservabilityWorkflowMeta
	session := w.dbMgr.NewSession(ctx, db.WithMaster())
	if err := session.WithContext(ctx).
		Model(&model.ObservabilityWorkflowMeta{}).
		Where("id IN ?", ids).
		Find(&rows).Error; err != nil {
		return nil, errorx.WrapByCode(err, obErrorx.CommonMySqlErrorCode)
	}
	for _, r := range rows {
		result[r.ID] = r
	}
	return result, nil
}
