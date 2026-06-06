CREATE TABLE IF NOT EXISTS `observability_workflow_meta`
(
    `id`             bigint unsigned                          NOT NULL COMMENT '工作流ID(与生产 workflow.id 一致)',
    `name`           varchar(512) COLLATE utf8mb4_general_ci  NOT NULL DEFAULT '' COMMENT '工作流名称',
    `icon_uri`       varchar(512) COLLATE utf8mb4_general_ci  NOT NULL DEFAULT '' COMMENT '图标路径',
    `space_id`       bigint unsigned                          NOT NULL DEFAULT '0' COMMENT '空间ID',
    `app_id`         bigint unsigned                          NOT NULL DEFAULT '0' COMMENT '应用ID',
    `status`         int                                      NOT NULL DEFAULT '0' COMMENT '状态',
    `content_type`   int                                      NOT NULL DEFAULT '0' COMMENT '内容类型',
    `mode`           int                                      NOT NULL DEFAULT '0' COMMENT '模式',
    `latest_version` varchar(64) COLLATE utf8mb4_general_ci   NOT NULL DEFAULT '' COMMENT '最新版本',
    `created_at`     datetime                                 NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '工作流创建时间',
    `updated_at`     datetime                                 NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '工作流更新时间',
    `snapshot_at`    datetime                                 NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '快照导入时间',
    PRIMARY KEY (`id`),
    KEY `idx_space` (`space_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_general_ci COMMENT ='工作流元数据快照(只读导出, 用于把 span workflow_id 解析为名称/图标)';
