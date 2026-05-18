# Loop Deploy — obs-merge-20260518 (upstream 5-13 + obs sidecar + Studio bridge)

> Status: **deployed + verified on 220** (docker compose, 2026-05-18 23:48).
> CDRCB / production NOT touched.

## What this image contains

`10.10.10.206:8090/ynet-loop/app:obs-merge-20260518` =
- upstream/main @ `56ba0ef2` (#518 expt_template item_retry_num, 212 commits behind ynet-main-2026-05 originally)
- + Studio session bridge (`backend/api/router/.../session.go`, `backend/infra/middleware/session/session.go`, `pat_verify.go`, `observability/openapi.go`, `foundation/user_repo_impl.go`)
- + RED middleware + `/metrics` sidecar on `:8890` (`backend/api/api.go` Start)
- + lumberjack rolling log
- + nil-safe init closures (commit `14c32ca6`)
- + trace_producer skip-empty-topic tenant (commit `14c32ca6`)

`10.10.10.206:8090/ynet-loop/nginx:obs-merge-20260518` =
- nginx:1.27-alpine + frontend resources from the same build
- **does NOT proxy /api/* by default** — see `nginx-default.conf` mount below

## 220 deploy delta (what we changed on disk)

Working tree: `/home/dev/ynet-deploy/loop/`

```
docker-compose.yml                          ← updated (image tag, MySQL, COZE_LOOP_* envs, ports, nginx mount, entrypoint, PWD)
docker-compose.yml.obs-merge-final-20260518 ← backup
nginx-default.conf                          ← NEW (proxy /api/, /v1/, /metrics to ynet-loop-app:8888)
observability.yaml                          ← UNMOUNTED (image-bundled patched conf wins)
observability.yaml.bak-1779118600           ← old mounted version with cozeloop-namesrv hostnames
```

### docker-compose.yml summary
- `image:` → `:obs-merge-20260518` for both app and nginx
- `ports:` → `["8888:8888", "8892:8890"]` (8892 is the metrics sidecar host port)
- `entrypoint:` → `["/coze-loop/bin/main"]` (upstream image uses /coze-loop, not /ynet-loop)
- `environment:` add BOTH `YNET_LOOP_*` AND `COZE_LOOP_*` envs (the upstream binary reads `COZE_LOOP_*`; we keep `YNET_LOOP_*` for back-compat). Specifically MySQL → `10.10.10.220:3308`, user `root`, password `root`, db `ynet-loop`.
- Add `PWD=/coze-loop` so `file.FindSubDir(os.Getenv("PWD"), "conf/locales")` works.
- Drop the `./observability.yaml:/ynet-loop/conf/observability.yaml:ro` line (image now has the patched obs.yaml).
- Nginx service: add `./nginx-default.conf:/etc/nginx/conf.d/default.conf:ro` mount.

### MySQL
- External OceanBase at `111.204.125.244:8100` was unreachable from 220 — substituted with the local `ynet-prod-mysql` container at `10.10.10.220:3308` (root/root).
- Database `ynet-loop` created fresh; **66 init/alter SQL files** from
  `release/deployment/helm-chart/charts/app/bootstrap/init/mysql/init-sql/`
  applied via `/tmp/loop-db-setup.sh` (idempotent, uses the upstream
  `CozeLoopExecuteAlterFile` stored procedure for ALTERs).
- Resulting tables: 55 (excluding alter-only files).

### RMQ topics
RMQ broker at `10.10.10.226:9876` had **zero** Loop topics. We pre-created 18 via `mqadmin updateTopic`:
- `trace_ingestion_event`, `trace_annotation_event`, `trace_span_with_annotation_event`, `trace_backfill_event`, `trace_to_task`, `observability_span_queue`
- `cozeloop_async_tasks`, `cozeloop_evaluation_correction_evaluator_result`, `cozeloop_evaluation_expt_turn_result_filter`, `cozeloop_evaluation_online_expt_eval_result`
- `data_async_tasks`
- `evaluation_expt_aggr_calculate_event`, `evaluation_expt_record_eval_event`, `evaluation_expt_scheduler_event`
- `evaluator_record_correction_event`
- `expt_export_csv_event`, `expt_lifecycle_event`, `expt_online_eval_result_event`

Script: `/tmp/create-rmq-topics.sh` on 220.

### Image conf patches (baked-in via `/tmp/loop-ctx/Dockerfile`)
We baked a patched `/coze-loop/conf` into the image with sed
replacements applied to all `*.yaml`:
- `cozeloop-namesrv:9876` → `10.10.10.226:9876`
- `cozeloop-mysql` → `10.10.10.220` (and port 3306 → 3308)
- `cozeloop-redis` host → `10.10.10.226`
- `cozeloop-clickhouse:9008` → `10.10.10.226:19000`
- `cozeloop-minio:19000` → `10.10.10.226:9000`
- Passwords/databases/buckets kept as `ynet-loop-*` to match middleware containers

## Smoke test (2026-05-18 23:48)
9/9 pass:
- `/ping` 200
- `/metrics` (sidecar :8892) 200, 43 metric families
- `/api/observability/v1/spans/list` (old endpoint)
- `/api/evaluation/v1/experiment_templates/list` (NEW)
- `/api/evaluation/v1/evaluators/list` (NEW)
- `/api/evaluation/v1/evaluation_sets/list` (NEW)
- `/api/evaluation/v1/experiments/list`
- `/api/observability/v1/tasks/list` (the path that exercises the nil-safe task client)
- `/v1/loop/opentelemetry/v1/traces` (Studio server-to-server bridge, returns `code:0`)

UI verified (http://10.10.10.220:8082):
- Login + register flow
- Side nav: Prompt 工程 (Prompt 开发 / Playground), 评测 (评测集 / 评估器 / 实验), 观测 (Trace), 标签 (标签管理)
- 评估器 new-button dropdown: LLM 评估器, Code 评估器
- 实验 list with 批量选择 + 过滤器
- Trace list with platform / preset time range / advanced filter

## Open follow-ups (NOT done today)

1. **Schema-loop is on a substitute MySQL** (220:3308 / db `ynet-loop`). To go back to OceanBase, restore the OB host route then point compose `(COZE_|YNET_)LOOP_MYSQL_*` back to it. The 66 SQL files have already been applied to OB once long ago — verify drift before flipping.
2. **stash@{0} (104-file infra rebrand) deliberately kept on the stash**, not merged, so future upstream merges stay easy. If we ever want to ship hardcoded `ynet-loop-*` hostnames into the conf, apply it then.
3. **task-related features will not work end-to-end** until provideTaskClient is made truly lazy. The current nil-safe patch keeps boot clean but stored taskservice.Client has nil impl. If you exercise auto-task / task management features, methods will panic. Real fix = make `provideTaskClient` return a lazy adapter that fetches `observabilityHandler.ITaskApplication` at call time. Out of scope for the deploy.
4. **CDRCB production**: this image has not been pushed to the bank Harbor; conf hostnames here are 10.10.10.x dev IPs, not 30.3.165.x. Re-bake conf for prod when shipping there.
