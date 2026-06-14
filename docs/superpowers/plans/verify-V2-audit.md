# V2 融合分支审计报告

**分支**: `ynet-upstream-merge-2026-06`  
**审计日期**: 2026-06-06  
**审计方法**: `git grep` 纯读操作，不修改任何文件

---

## 表1：本地定制完整性（必须仍在）

| # | 定制项 | 状态 | 证据（文件:行） |
|---|--------|------|----------------|
| A1 | 应用主端口 8889 / LISTEN_ADDR | ⚠️ 部分 | `LISTEN_ADDR` 存在于 `backend/api/api.go:192`；但默认端口为 **8888**（`api.go:194: listenAddr = ":8888"`），不是 8889。8889 仅出现在测试数据中（eval_openapi_app_test.go:1894 作为实验 ID，非端口）。若 ynet 部署要求默认 8889，需通过 `LISTEN_ADDR=:8889` 环境变量覆盖，代码机制在，默认值已回退上游 8888。 |
| A2 | sidecar metrics 端口 8890 / METRICS_PORT | ✓ | `backend/api/api.go:204` 注释说明 default 8890；`api.go:217: addr = ":8890"` |
| A3 | loop 业务指标 `loop_task_total` / `loop_evaluator_invocation_total` | ✓ | `backend/pkg/observability/metrics.go:38,53` |
| A4 | RED / hertz_metrics 中间件 | ✓ | `backend/pkg/observability/http_middleware.go:16`（RED 注释）；`backend/pkg/observability/metrics.go:5,15`；文件 `backend/pkg/observability/hertz_metrics.go` 存在（`git grep -ln` 命中）；`HertzMetrics` / `hertz_metrics` 全局有引用 |
| A5 | Studio session bridge / pat_verify | ✓ | `backend/api/router/coze/loop/apis/middleware/pat_verify.go`（文件存在）；`backend/modules/observability/application/openapi.go`（StudioSession 相关）；`backend/pkg/observability/hertz_metrics.go` |
| A6 | trace_consume psm/tenant tags + SpanStats | ✓ | `backend/modules/observability/infra/metrics/metrics.go:147-148`（ConsumeTagPSM, ConsumeTagTenant）；`backend/modules/observability/domain/trace/entity/collector/consumer/span_stats.go`（SpanStatsEntry 类型全文件） |

**注意**：A1 LISTEN_ADDR 机制健在，但代码中 hardcode 默认值为 `:8888`（上游值）。若 ynet 部署脚本通过环境变量 `LISTEN_ADDR=:8889` 注入，则无问题；若依赖代码默认值为 8889，则需要修复。

---

## 表2：新能力存在性核对（上游 #519-#537 融合目标）

| PR | 能力 | 状态 | 证据（文件:行） |
|----|------|------|----------------|
| #520 | agent eval — A2AAgent / CustomAgent EvalTarget 类型 | ✓ | `backend/kitex_gen/coze/loop/evaluation/domain/eval_target/eval_target.go:59`（EvalTargetType_A2AAgent=9）；`:61`（EvalTargetType_CustomAgent=10）；`:1501-1503`（结构体字段 A2aAgent/CustomAgent） |
| #524 | prompt 批量调试 BatchDebugEvaluator | ✓ | `backend/api/handler/coze/loop/apis/evaluator_service.go:152-155`（handler 及路由 `/api/evaluation/v1/evaluators/batch_debug`）；`backend/api/router/coze/loop/apis/coze.loop.apis.go:161`；kitex_gen client/server 均存在 |
| #492 | trace 列抽取 ColumnExtract + Agent Metadata Discovery | ✓ | `UpsertColumnExtractConfig` / `GetColumnExtractConfig` handler `backend/api/handler/coze/loop/apis/observability_trace_service.go:182-191`；路由 `coze.loop.apis.go:334-335`；`GetAgentMetadata` handler `:194-197`；路由 `:377`；kitex_gen client 含全部三个方法 |
| #525/#516 | trace size 字段 + without_clip | ✓ | `backend/kitex_gen/coze/loop/observability/domain/trace/trace.go:48`（Trace_Size_DEFAULT）；`backend/kitex_gen/coze/loop/observability/trace/coze.loop.observability.trace.go:34`（ListSpansRequest.WithoutClip 字段）；完整 getter/setter/serialization 均在 |
| #514 | 定时评测 cron / TriggerInterval / Scheduler | ✓ | `backend/modules/evaluation/application/convertor/experiment/expt_template.go:620`（dto.TriggerInterval）；`:607`（exptSchedulerDO2DTO）；`:73`（CronActivate）；`:1061`（SchedulerExperimentNameFromTemplate）；回归测试 `expt_template_test.go:56-79` |

---

## 结论

### 本地定制：5/6 项全 ✓，1 项有条件 ✓（A1 存在风险）

- **5 项完全确认**：metrics 端口 8890（A2）、loop 业务指标（A3）、RED/hertz 中间件（A4）、pat_verify/bridge（A5）、psm/tenant/SpanStats（A6）
- **1 项需关注**：LISTEN_ADDR 机制存在（A1），但代码默认值为上游的 `:8888`，非 ynet 定制的 `8889`。若 ynet 的 `docker-compose.yml` / 部署脚本中有 `LISTEN_ADDR=:8889`，则运行时无问题；若没有，服务将监听 8888 而非 8889。

### 新能力：5/5 项全 ✓

所有目标 PR 的新能力均已合入，证据清晰。实现在 `backend/api` handler + router + `backend/kitex_gen` 生成代码中均存在。

---

## 待修 / 待确认事项

| 优先级 | 项 | 建议 |
|--------|-----|------|
| 中 | A1 默认端口 8888 vs 8889 | 确认 ynet 部署脚本是否设置了 `LISTEN_ADDR=:8889`。若无，在 `backend/api/api.go:194` 将默认值改回 `:8889` 或在 docker-compose/Deployment 中补充该环境变量。 |
| 低 | SpanStats 位置 | SpanStats 在 `backend/modules/observability/domain/trace/entity/collector/consumer/` 下，A6 审计仅对 `backend/modules/observability/infra/metrics` 做 grep，SpanStats 不在该路径内，但功能确实存在。审计命令范围可适当放宽。 |
