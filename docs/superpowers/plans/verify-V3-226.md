# V3 验证报告 — 226 服务器融合版冒烟测试

**日期**: 2026-06-06  
**分支**: ynet-upstream-merge-2026-06  
**目标容器**: ynet-loop-app @ 10.10.10.226  
**镜像**: 10.10.10.206:8090/ynet-loop/app:latest（06-05 构建）

---

## 1. 容器健康与监听端口

```
$ docker ps --format "{{.Names}}\t{{.Status}}" | grep ynet-loop-app
ynet-loop-app    Up 12 hours (healthy)
```

```
$ netstat -tlnp | grep -E "8888|8890"
tcp  0  0 :::8890  :::*  LISTEN  12/main
tcp  0  0 :::8888  :::*  LISTEN  12/main
```

- **结论**: 容器健康（Docker healthcheck = healthy），应用主端口 8888 + 指标端口 8890 均正常监听。

---

## 2. 指标回归（本地定制不能丢）

```
$ wget -qO- http://localhost:8890/metrics | grep loop
# HELP loop_evaluator_invocation_total Evaluator calls by name
# TYPE loop_evaluator_invocation_total counter
loop_evaluator_invocation_total{evaluator_name=""} 1
```

- `loop_evaluator_invocation_total` 出现，说明 `backend/pkg/observability/metrics.go` 已编译进二进制并注册到 Prometheus。
- `loop_task_total` / `loop_task_duration_seconds` 未出现在输出中，原因是 Prometheus counter/histogram 仅在首次 Observe/Inc 后才出现 series，容器运行期间暂无任务触发，属正常现象（定义在代码中，非丢失）。
- 指标端口 8890 独立服务，与主业务端口 8888 隔离——本地定制保留。

**结论**: 指标定制在，无回退。

---

## 3. 融合新路由探测

路由来源文件：`backend/api/router/coze/loop/apis/coze.loop.apis.go`

| 路由 Path | 方法 | HTTP 状态码 | 非 404？ | 说明 |
|-----------|------|------------|---------|------|
| `/api/evaluation/v1/evaluators/batch_debug` | POST | **200** | ✅ | 评测批量 debug（上游 PR #520+ 融合新增） |
| `/api/observability/v1/column_extract_config` | GET | **200** | ✅ | trace 列提取配置读取（上游 PR #530+ 融合新增） |
| `/api/observability/v1/trace/agent/metadata` | GET | **200** | ✅ | agent metadata（上游 PR #525+ 融合新增） |

所有探测均返回 200（框架层鉴权未拦截 OPTIONS/路由匹配，handler 已注册）。无任何 404。

**结论**: 3 条融合新路由全部存在，证明运行的二进制是融合版本。

---

## 4. 综合结论

| 检查项 | 结论 |
|--------|------|
| 容器健康 | healthy，运行 12+ 小时 |
| 主端口 8888 监听 | 正常 |
| 指标端口 8890 监听 | 正常 |
| 本地定制指标注册 | 在（loop_evaluator_invocation_total 可见） |
| 融合新路由 batch_debug | 非 404（200） |
| 融合新路由 column_extract_config | 非 404（200） |
| 融合新路由 agent/metadata | 非 404（200） |
| **226 是否为融合版** | **是** |

---

## 5. 人工 UI 冒烟清单（需在前端手动验证）

以下项目需要操作人员登录前端界面逐一点击验证，自动化探测无法覆盖：

- [ ] **评测实验 - 新增 custom_agent target 类型**
  - 进入评测模块 → 新建实验 → Target 类型选择 `custom_agent`
  - 期望：类型可选，表单正常展示，提交后实验创建成功

- [ ] **评测实验 - 新增 a2a_agent target 类型**
  - 进入评测模块 → 新建实验 → Target 类型选择 `a2a_agent`
  - 期望：类型可选，表单正常展示，提交后实验创建成功

- [ ] **评测批量 debug（batch_debug）**
  - 进入评测模块 → 选中一个评测器 → 触发"批量调试" / Batch Debug 功能
  - 期望：请求 `/api/evaluation/v1/evaluators/batch_debug` 正常返回，结果展示在前端

- [ ] **跑批评测任务**
  - 对上述 custom_agent 或 a2a_agent 实验创建并触发一次跑批
  - 期望：任务状态从 pending → running → completed，结果可查看

- [ ] **Trace - 列提取配置（column_extract_config）**
  - 进入 Trace / Observability 模块 → 打开 trace 列管理 → 保存一次列提取配置
  - 期望：调用 `/api/observability/v1/column_extract_config` (POST) 成功，配置持久化

- [ ] **Trace - Agent Metadata**
  - 在 Trace 详情页选择一个 agent trace → 查看 agent metadata 信息
  - 期望：调用 `/api/observability/v1/trace/agent/metadata` 成功，metadata 正常展示

- [ ] **Trace 输入输出预览**
  - 在 Trace 列表选择一条 trace → 展开查看输入 / 输出内容
  - 期望：span 输入/输出文本正常渲染，无乱码/空白

- [ ] **指标大盘确认**
  - 访问 Grafana / 监控系统 → 查找 `loop_evaluator_invocation_total` 指标
  - 期望：跑批后指标数值有增量，证明埋点链路完整
