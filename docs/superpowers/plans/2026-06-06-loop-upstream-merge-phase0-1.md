# Loop 上游融合 — 验证现有融合 实施 Plan（修订版）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

## ⚠️ 重大修订说明（2026-06-06）
原计划假设"分批 cherry-pick 做融合"。执行 Task 0.1 时发现：**融合早已在 `ynet-upstream-merge-2026-06` 分支用 `git merge upstream/main` 完成**：
- `8af71111 Merge upstream/main (#519-#537): restore eval_target + agent eval`
- `a18ed619 fix(eval): restore loop_task/evaluator metrics + exptStatusLabel after upstream merge`
- `upstream/main` 已是 HEAD 完整祖先（领先数 0）；相对 origin/main 改动 800 文件 +78520/-44786。
- 本地定制基本健在：sidecar `:8890`(api.go)、`loop_task_total`/`loop_evaluator_invocation_total`(pkg/observability/metrics.go)、Studio bridge(pat_verify.go/openapi.go)。
- **本地 `go build ./...` 已通过（exit 0，0 错误）。**

**因此 Phase 1 改为「验证现有融合」**，不再重做。

**Goal:** 验证 `ynet-upstream-merge-2026-06` 分支上的融合可编译、可测、本地定制无回退、新能力（agent eval / prompt 批量调试 / trace 预览）可用，并在 226/220 部署冒烟。

**Tech Stack:** Go 1.24.6 / kitex / thrift(v0.13.0) / MySQL / ClickHouse / Docker compose / 私有 registry 10.10.10.206:8090。

**Spec:** `coze-loop/docs/superpowers/specs/2026-06-06-ynet-4projects-backlog-design.md`

**约定：**
- 工作目录：`/Users/luzhipeng/projects/ynet/coze-loop`（当前分支 `ynet-upstream-merge-2026-06`）。
- 服务器：`sshpass -p 'root1234' ssh -o StrictHostKeyChecking=no dev@10.10.10.226`（或 .220）；sudo 用 `echo root1234 | sudo -S`。
- commit 用 Conventional Commits，结尾加 `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`；**仅用户确认后 push**。

---

## V1: 本地编译 + 单测

**Files:** 无改动（验证）

- [x] **Step 1: go build ./...** — 已通过（exit 0，日志 /tmp/loop_build.log 无 error）。
- [ ] **Step 2: 跑 evaluation 单测**

Run: `cd backend && go test ./modules/evaluation/... 2>&1 | tail -40`
Expected: PASS（环境依赖型用例可能 skip/fail，逐项记录，区分"真失败"与"缺中间件"）。

- [ ] **Step 3: 跑 observability 单测**

Run: `cd backend && go test ./modules/observability/... ./pkg/observability/... 2>&1 | tail -40`
Expected: PASS；重点看 metrics / trace 相关用例。

- [ ] **Step 4: 记录结果**

把通过/失败/跳过的包与原因写入 `docs/superpowers/plans/verify-V1-tests.md`。区分代码缺陷 vs 测试环境缺中间件。

---

## V2: 本地定制完整性审计 + 新能力存在性核对

**Files:** 无改动（验证），产出 `docs/superpowers/plans/verify-V2-audit.md`

- [ ] **Step 1: 本地定制审计（逐项确认仍在）**

```bash
# 监听端口 8889（应用主端口，避开 Studio 8888）
git grep -n "8889\|LISTEN_ADDR" -- backend/api backend/cmd 2>/dev/null
# sidecar metrics :8890
git grep -n "8890\|METRICS_PORT" -- backend/api/api.go
# loop 业务指标
git grep -n "loop_task_total\|loop_evaluator_invocation_total" -- backend/pkg/observability
# RED 中间件 + hertz metrics
sed -n '1,40p' backend/pkg/observability/hertz_metrics.go
# Studio session bridge / pat verify
git grep -ln "pat_verify\|server-to-server\|StudioSession" -- backend
# trace_consume psm/tenant tags + SpanStats
git grep -n "psm\|tenant" -- backend/modules/observability/infra/metrics/metrics.go | head
```
Expected: 每项都有命中。任一缺失 → 标红记录，作为回归修复项。

- [ ] **Step 2: 新能力存在性核对（融合目标）**

```bash
# #520 agent eval：a2a_agent / custom_agent EvalTarget 类型
git grep -rin "a2a_agent\|custom_agent\|CustomAgent\|A2AAgent" -- backend/modules/evaluation idl 2>/dev/null | head
# #524 prompt 批量调试
git grep -rin "batch_debug\|BatchDebug\|prompt_batch" -- backend/modules idl 2>/dev/null | head
# #492 trace 列抽取预览
git grep -rin "column_extract\|ColumnExtract\|metadata.discovery\|agent_metadata" -- backend/modules/observability 2>/dev/null | head
# #525 trace size / #516 without_clip
git grep -rin "without_clip\|TraceSize\|trace_size" -- backend/modules/observability idl 2>/dev/null | head
```
Expected: 每个融合目标都有代码命中。缺失 → 说明该 PR 没合进来，记录待补。

- [ ] **Step 3: 写审计报告**

`verify-V2-audit.md`：定制项 ✓/✗ 表 + 新能力 ✓/✗ 表 + 结论（是否需补合某 PR / 修复某定制）。

---

## V3: 226 部署融合镜像 + 评测/trace 冒烟

**Files:** 无源码改动；产出 `docs/superpowers/plans/verify-V3-226.md`

- [ ] **Step 1: 确认 DB 迁移文件并核对 226 现状**

```bash
find release -iname "experiment*.sql" -o -iname "*column_extract*.sql" | head
sshpass -p 'root1234' ssh dev@10.10.10.226 'docker ps --format "{{.Names}}" | grep -Ei "loop.*(mysql|clickhouse)"'
```
Expected: 找到迁移 SQL；确认 226 上 Loop 的 mysql/clickhouse 容器名。

- [ ] **Step 2: build 融合镜像并 push**

```bash
make image IMAGE_REGISTRY=10.10.10.206:8090 IMAGE_NAME=ynet-loop/app IMAGE_TAG=fusion-verify-$(git rev-parse --short HEAD)
```
（具体 Make 变量以根 Makefile 为准，先 `grep -nE "IMAGE_|^image:" Makefile` 确认。）
Expected: 镜像 build 成功并 push 到 206:8090。

- [ ] **Step 3: 226 部署 + 执行迁移**

scp 迁移 SQL → 226，对 Loop mysql 执行；更新 226 compose 镜像 tag 并重启 loop-app。
Expected: loop-app healthy；迁移无错。

- [ ] **Step 4: 评测冒烟（agent eval + prompt 批量调试 + trace 预览）**

建实验 → 选 `custom_agent`/`a2a_agent` target → 跑批 → 看结果；prompt 批量调试；trace 输入输出预览。
Expected: 三项可用；结果记入 verify-V3-226.md。

- [ ] **Step 5: /metrics + sidecar 回归**

```bash
sshpass -p 'root1234' ssh dev@10.10.10.226 \
  'curl -s http://localhost:8890/metrics | grep -E "loop_(task|evaluator)" | head; \
   curl -s -o /dev/null -w "8889:%{http_code}\n" http://localhost:8889/'
```
Expected: 业务指标在；8889 主端口可达。

---

## V4: 220 端到端 + obs 看板 + 验收（合并原 Phase 0.3 修 unhealthy）

**Files:** 产出 `docs/superpowers/plans/verify-V4-acceptance.md`

- [ ] **Step 1: 修复/确认 220 loop-app 状态**

```bash
sshpass -p 'root1234' ssh dev@10.10.10.220 \
  'docker inspect --format "{{json .State.Health}}" ynet-loop-app | python3 -m json.tool | tail -20; \
   docker logs --tail 40 ynet-loop-app 2>&1 | tail -20'
```
依根因修复或确认结论。

- [ ] **Step 2: 220 部署融合镜像 + Studio↔Loop 端到端**

220 更新 loop-app 镜像 → 从 Studio 发起评测 → trace 流向 Loop。
Expected: 端到端通；server-to-server trace ingest 未回退。

- [ ] **Step 3: obs 看板回归**

Grafana 查 Loop 业务指标（loop_task/evaluator + RED）。
Expected: 看板有 Loop 指标。

- [ ] **Step 4: 填验收清单并提交**

逐条勾：
- [ ] go build 通过；evaluation+observability 单测结论明确
- [ ] 本地定制审计全 ✓（或列出需修项）
- [ ] agent eval / prompt 批量调试 / trace 预览 可用
- [ ] /metrics(:8889)+sidecar(:8890) 正常；Studio 集成未回退
- [ ] DB 迁移已执行
```bash
git add docs/superpowers/plans/verify-*.md
git commit -m "docs(merge): Phase 1 verification of existing upstream fusion

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## 后续 Plan（各自单独成文）
- Phase 2 — Studio 自研对接（falcon 批量测试入口 + 工作流分类 + Task lazy init）
- Phase 3 — Intent-hub + Guard-go 加固
- 贯穿 — 部署标准化 + /metrics Hertz 采集修复
- 插队 — 客户现场 P0（401 / OB 切换 / health）
