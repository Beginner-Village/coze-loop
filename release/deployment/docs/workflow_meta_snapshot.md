# 工作流元数据快照 (observability_workflow_meta)

observability 的 trace 详情需要把 span 上的工作流 id 解析成可读的工作流名称 / 图标。
生产里 span 只带工作流 id（没有 name/icon），因此需要从生产 `workflow_meta` 表只读
导出一份快照，灌进 coze-loop 自己的 `observability_workflow_meta` 表，由
`WorkflowProvider` 查表补全。

## span 上的工作流 id tag（已用生产真实数据核对）

经核对生产 `observability_spans` 真实数据，结论:

- 工作流 id 的真实 tag key 是 **`sub_workflow_id`**，类型 **Int64**，存在 span 的
  **`tags_long`** map 里(**不是** `tags_string`，**也不是** `workflow_id`)。
- 同一工作流的多个 node span(span_name = 开始 / 选择器 / 代码 / 输出 / 结束 等)都
  带同一个 `sub_workflow_id`。
- span 上**没有**顶层 `workflow_id` tag。
- 次要来源(评测场景):span 带 `eval_target_id`(tags_long) + `eval_target_type`
  (tags_string，值如 `CozeWorkflow` / `LoopPrompt`)。当 `eval_target_type` 表示工作流
  (`CozeWorkflow`)时，`eval_target_id` 也是工作流 id，作为 `sub_workflow_id` 缺失时的兜底。
- `sub_workflow_id` / `eval_target_id` 的值都与 `workflow_meta.id` 对得上。

`WorkflowProvider.extractWorkflowID` 优先从 `tags_long` 取 `sub_workflow_id`(Int64，
兼容 string 数字)，缺失时再走 workflow 类型的 `eval_target_id`，最后回退到已解析的
legacy `workflow` tag。0 / 缺失 / 不可解析的值跳过。

数据链路:

```
生产 workflow 库 (只读)
      │  export-workflow-meta.sh   (只读 SELECT, 永不写生产)
      ▼
_workflow_export/workflow_meta_all.tsv   (本地快照, 含表头)
      │  cmd/workflow_meta_import         (写 dev/onsite 库, 幂等 upsert)
      ▼
observability_workflow_meta 表
      │  WorkflowProvider.BatchGetWorkflows
      │  (从 span tags_long.sub_workflow_id 取 id, 查表补 name/icon)
      ▼
trace 详情里的 workflow {id,name,icon_uri}
```

## 1. 从生产只读刷新快照

导出脚本在仓库外的导出目录里(只读、可重复执行):

```
/Users/luzhipeng/projects/ynet/_workflow_export/export-workflow-meta.sh
```

它对生产库执行只读 `SELECT`，产出:

- `workflow_meta_all.tsv`     — 全量存活工作流
- `workflow_meta_recent90d.tsv` — 近 90 天有改动的工作流

TSV 列(制表符分隔，首行表头):

```
id  name  icon_uri  space_id  app_id  status  content_type  mode  created_at  updated_at  latest_version
```

`created_at` / `updated_at` 为 unix 毫秒。该脚本只读，不会写任何生产数据。

## 2. 建表

表结构随部署初始化 SQL 自动创建:

- `release/deployment/docker-compose/bootstrap/mysql-init/init-sql/observability_workflow_meta.sql`
- `release/deployment/helm-chart/charts/app/bootstrap/init/mysql/init-sql/observability_workflow_meta.sql`

导入命令也会 `AutoMigrate` 兜底建表，全新环境无需手工执行。

## 3. 导入快照到目标库

导入工具**只写目标(dev/onsite)库，绝不连生产**:

```bash
cd backend
go run ./cmd/workflow_meta_import \
  -dsn 'user:pass@tcp(127.0.0.1:3306)/cozeloop?charset=utf8mb4&parseTime=true&loc=Local' \
  -file /Users/luzhipeng/projects/ynet/_workflow_export/workflow_meta_all.tsv
```

- 按主键 `id` upsert，幂等，可用新快照反复重灌覆盖旧元数据。
- `-batch` 控制每批 upsert 行数(默认 500)。
- 想只刷新近期改动，把 `-file` 换成 `workflow_meta_recent90d.tsv`。

## 4. 刷新策略

工作流元数据变化不频繁。建议在以下时机重新执行第 1、3 步:

- 上线 / 部署新环境时全量灌一次 `workflow_meta_all.tsv`。
- 周期性(如每周)用 `workflow_meta_recent90d.tsv` 增量刷新近期改动。

未在快照里的 `sub_workflow_id` 仍会返回 `{"id":<id>,"name":"","icon_uri":""}`，不会报错，
也不阻塞 trace 读取——只是 name/icon 为空，刷新快照后即可补齐。
