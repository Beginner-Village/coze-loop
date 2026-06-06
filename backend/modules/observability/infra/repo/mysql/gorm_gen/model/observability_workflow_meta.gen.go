// Copyright (c) 2026 coze-dev Authors
// SPDX-License-Identifier: Apache-2.0

package model

import (
	"time"
)

const TableNameObservabilityWorkflowMeta = "observability_workflow_meta"

// ObservabilityWorkflowMeta 工作流元数据快照, 来源于生产 workflow_meta 只读导出.
// 用于把 span 里仅有的 workflow_id 解析为可读的工作流名称/图标.
type ObservabilityWorkflowMeta struct {
	ID            int64     `gorm:"column:id;type:bigint(20) unsigned;primaryKey;comment:工作流ID" json:"id"`                                 // 工作流ID(与生产 workflow.id 一致)
	Name          string    `gorm:"column:name;type:varchar(512);not null;comment:工作流名称" json:"name"`                                      // 工作流名称
	IconURI       string    `gorm:"column:icon_uri;type:varchar(512);not null;comment:图标路径" json:"icon_uri"`                               // 图标路径
	SpaceID       int64     `gorm:"column:space_id;type:bigint(20) unsigned;not null;index:idx_space;comment:空间ID" json:"space_id"`        // 空间ID
	AppID         int64     `gorm:"column:app_id;type:bigint(20) unsigned;not null;comment:应用ID" json:"app_id"`                            // 应用ID
	Status        int32     `gorm:"column:status;type:int;not null;comment:状态" json:"status"`                                              // 状态
	ContentType   int32     `gorm:"column:content_type;type:int;not null;comment:内容类型" json:"content_type"`                                // 内容类型
	Mode          int32     `gorm:"column:mode;type:int;not null;comment:模式" json:"mode"`                                                  // 模式
	LatestVersion string    `gorm:"column:latest_version;type:varchar(64);not null;comment:最新版本" json:"latest_version"`                    // 最新版本
	CreatedAt     time.Time `gorm:"column:created_at;type:datetime;not null;default:CURRENT_TIMESTAMP;comment:工作流创建时间" json:"created_at"`  // 工作流创建时间
	UpdatedAt     time.Time `gorm:"column:updated_at;type:datetime;not null;default:CURRENT_TIMESTAMP;comment:工作流更新时间" json:"updated_at"`  // 工作流更新时间
	SnapshotAt    time.Time `gorm:"column:snapshot_at;type:datetime;not null;default:CURRENT_TIMESTAMP;comment:快照导入时间" json:"snapshot_at"` // 快照导入时间
}

// TableName ObservabilityWorkflowMeta's table name
func (*ObservabilityWorkflowMeta) TableName() string {
	return TableNameObservabilityWorkflowMeta
}
