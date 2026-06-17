<div align="center">
<p>
  <a href="#什么是-ynet-loop">YNET Loop</a> •
  <a href="#功能清单">功能清单</a> •
  <a href="#快速开始">快速开始</a> •
  <a href="#开发指南">开发指南</a>
</p>
<p>
  <img alt="License" src="https://img.shields.io/badge/license-apache2.0-blue.svg">
  <img alt="Go Version" src="https://img.shields.io/badge/go-%3E%3D%201.24.0-blue">
</p>

[English](README.md) | 中文

</div>

## 什么是 YNET Loop

YNET Loop 是一个面向开发者，专注于 AI Agent 开发与运维的平台级解决方案。 它可以解决 AI Agent 开发过程中面临的各种挑战，提供从开发、调试、评估、到监控的全生命周期管理能力。
YNET Loop 在商业化版本的基础上，推出开源版免费对开发者开放核心基础功能模块，以开源模式共享核心技术框架，开发者可根据业务需求定制与扩展，便于社区共建、分享交流，助力开发者零门槛参与 AI Agent 的探索与实践。

## YNET Loop 能做什么？

YNET Loop 通过提供全生命周期的管理能力，帮助开发者更高效地开发和运维 AI Agent。无论是提示词工程、AI Agent 评测，还是上线后的监控与调优，YNET Loop 都提供了强大的工具和智能化的支持，极大地简化了 AI Agent 的开发流程，提升了 AI Agent 的运行效果和稳定性。

* **Prompt 开发**：YNET Loop 的 Prompt 开发模块为开发者提供了从编写、调试、优化到版本管理的全流程支持，通过可视化 Playground 实现 Prompt 的实时交互测试，让开发者能够直观比较不同大语言模型的输出效果。
* **评测**：YNET Loop 评测模块为开发者提供系统化的评测能力，能够对 Prompt 和智能体的输出效果进行多维度自动化检测，例如准确性、简洁性和合规性等。
* **观测**：YNET Loop 为开发者提供了全链路执行过程的可视化观测能力，完整记录从用户输入到 AI 输出的每个处理环节，包括 Prompt 解析、模型调用和工具执行等关键节点，并自动捕获中间结果和异常状态。

## 功能清单

| **功能** | **功能点** |
| --- | --- |
| Prompt 调试 | *Playground 调试、对比 <br>* Prompt 版本管理 |
| 评测 | *管理评测集 <br>* 管理评估器 <br> * 管理实验 |
| 观测 | *SDK 上报 Trace <br>* Trace 数据观测 |
| 模型 | 支持接入 OpenAI、火山方舟等模型 |

## 快速开始

### 部署方式1：Docker 部署 (Docker Compose)
>
> 请提前安装并启动 Docker Engine。

操作步骤：

1. 获取源码。
   执行以下命令，获取 YNET Loop 最新版本的源码。

   ```Bash
   # 克隆代码
   git clone http://git.ynet.io/Cheetah/Develop/ai/agent/ynet/ynet-loop.git

   # 进入 ynet-loop 目录下
   cd ynet-loop
   ```

2. 配置模型。
   1. 进入 `ynet-loop` 目录。
   2. 编辑文件 `release/deployment/docker-compose/conf/model_config.yaml`。
   3. 修改 api_key 和 model 字段。以火山方舟为例：
      * api_key：火山方舟 API Key。中国境内用户参考[火山方舟文档](https://www.volcengine.com/docs/82379/1541594)；非中国境内的用户可参考[BytePlus ModelArk 文档](https://docs.byteplus.com/en/docs/ModelArk/1361424)。
      * model：火山方舟模型接入点的 Endpoint ID。中国境内用户参考[火山方舟文档](https://www.volcengine.com/docs/82379/1099522)；非中国境内的用户可参考[BytePlus ModelArk 文档](https://docs.byteplus.com/en/docs/ModelArk/1099522)。
3. 启动服务。
   执行以下命令，使用 Docker Compose 快速部署 YNET Loop 开源版。

   ```Bash
   # 启动服务，默认为开发模式
   # 在 ynet-loop/ 目录下执行
   make compose-up
   ```

4. 通过浏览器访问 YNET Loop 开源版 `http://localhost:8082`。

### 部署方式2：Kubernetes 部署（Helm Chart）

> * 已准备 Kubernetes 集群、启用 Nginx Ingress Addons，并安装 Kubectl 和 Helm 工具。
> * 如需在本地快速体验，可通过 Minikube 部署 Kubernetes 集群。

操作步骤：

1. 执行以下命令获取 Helm Chart 包。

   ```Bash
   helm pull oci://docker.io/ynetdev/ynet-loop --version 1.0.0-helm
   tar -zxvf ynet-loop-1.0.0-helm.tgz && cd ynet-loop && rm -f ../ynet-loop-1.0.0-helm.tgz
   ```

2. 配置模型。
   进入 `ynet-loop` 目录，编辑文件 `release/deployment/helm-chart/umbrella/conf/model_config.yaml`。配置以下字段，此处以火山方舟为例：
   * api_key：火山方舟 API Key。中国境内用户参考[火山方舟文档](https://www.volcengine.com/docs/82379/1541594)；非中国境内的用户可参考[BytePlus ModelArk 文档](https://docs.byteplus.com/en/docs/ModelArk/1361424)。
   * model：火山方舟模型接入点的 Endpoint ID。中国境内用户参考[火山方舟文档](https://www.volcengine.com/docs/82379/1099522)；非中国境内的用户可参考[BytePlus ModelArk 文档](https://docs.byteplus.com/en/docs/ModelArk/1099522)。
3. 配置 Ingress 规则。
   Ingress 用于暴露服务到外部，需根据集群实际情况配置项目目录下的`templates/ingress.yaml` 文件，自行修改 ingressClassName 等参数，配置 class、instance、host、ip 分配等要素。
4. 部署并启动服务。
   执行以下命令，使用 Helm 快速部署 YNET Loop 开源版。

   ```Bash
   # 在 ynet-loop/ 目录下执行
   make helm-up
   # 等待服务部署完成后，查看集群pod状态
   make helm-pod
   # 查看服务启动日志，如果 app 和 nginx 均正常运行，表示部署成功
   make helm-logf-app
   make helm-logf-nginx
   ```

5. 通过浏览器访问 YNET Loop 开源版。
   访问域名及 URL 取决于你的集群分配的域名以及 URL。
6. 开始定制你的 YNET Loop 项目。
   参考 `examples/` 目录示例，修改 `values.yaml` 即可覆盖默认设置，修改后重新执行 `make helm-up` 生效。

> [!WARNING]
> 如果要将 YNET Loop 部署到公网环境，建议在部署前评估整体评估安全风险，例如账号注册功能、YNET Server 监听地址配置、SSRF 和部分 API 水平越权的风险，并采取相应防护措施。

## 使用 YNET Loop 开源版

* Prompt 开发与调试：YNET Loop 提供了完整的提示词开发流程。
* 评测：YNET Loop 的评测功能提供标准评测数据管理、自动化评估引擎和综合的实验结果统计。
* Trace 上报与查询：YNET Loop 支持对平台上创建的 Prompt 调试的 Trace 自动上报，实时追踪每一条 Trace 数据。
* 开源版使用 YNET Loop SDK：YNET Loop 三个语言的 SDK 均适用于商业版和开源版。对于开源版，开发者只需要初始化时修改部分参数配置。

## 开发指南

* 系统架构：了解 YNET Loop 开源版的技术架构与核心组件。
* 启动模式：安装部署 YNET Loop 开源版时，默认使用稳定模式，直接通过镜像启动，无需额外编译构建步骤。
* 模型配置：YNET Loop 开源版通过 Eino 框架支持多种 LLM 模型，参考此文档查看支持的模型列表，了解如何配置模型。
* 代码开发与测试：了解如何基于 YNET Loop 开源版进行二次开发与测试。
* 故障排查：了解如何查看容器状态、系统日志。

## License

本项目采用 Apache 2.0 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 社区贡献

我们欢迎社区贡献，贡献指南参见 [CONTRIBUTING](CONTRIBUTING.md) 和 [Code of conduct](CODE_OF_CONDUCT.md)，期待您的贡献！

## 安全与隐私

如果你在该项目中发现潜在的安全问题，或你认为可能发现了安全问题，请及时通知安全团队。

## 加入社区

我们致力于构建一个开放、友好的开发者社区，欢迎所有对 AI Agent 开发感兴趣的开发者加入我们！

### 问题反馈与功能建议

为了更高效地跟踪和解决问题，保证信息透明和便于协同，我们推荐通过项目的 issue 与合并请求渠道参与。

### 技术交流与讨论

加入我们的技术交流群，与其他开发者分享经验、获取项目最新动态：

* Discord 服务器：[YNET Community](https://discord.com/invite/sTVN9EVS4B)
* Telegram 群组：[YNET](https://t.me/+pP9CkPnomDA0Mjgx)

## 致谢

感谢所有为 YNET Loop 项目做出贡献的开发者和社区成员。特别感谢：

* [Eino](https://github.com/cloudwego/eino) 框架团队提供的 LLM 集成支持
* [CloudWeGo](https://www.cloudwego.io) 团队开发的高性能框架
* 所有参与测试和反馈的用户
