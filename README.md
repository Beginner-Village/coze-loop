<div align="center">
<p>
<a href="#what-can-ynet-loop-do">YNET Loop</a> •
<a href="#feature-list">Feature list</a> •
<a href="#quick-start">Quick start</a> •
<a href="#developer-guide">Developer guide</a>
</p>
<p>
  <img alt="License" src="https://img.shields.io/badge/license-apache2.0-blue.svg">
  <img alt="Go Version" src="https://img.shields.io/badge/go-%3E%3D%201.24.0-blue">
</p>

English | [中文](README.cn.md)

</div>

## What is YNET Loop

YNET Loop is a developer-oriented, platform-level solution focused on the development and operation of AI agents. It addresses various challenges faced during the AI agent development process, providing full lifecycle management capabilities from development, debugging, evaluation, to monitoring.

Based on the commercial version, YNET Loop introduces an open-source edition that offers developers free access to core foundational feature modules. By sharing its core technology framework in an open-source model, developers can customize and extend according to business needs, facilitating community co-construction, sharing, and exchange, helping developers participate in AI agent exploration and practice with zero barriers.

## What can YNET Loop do?

YNET Loop helps developers develop and operate AI Agent more efficiently by providing full lifecycle management capabilities. Whether it is prompt engineering, AI Agent evaluation, or monitoring and optimization after deployment, YNET Loop provides powerful tools and intelligent support, greatly simplifying the development process of AI Agents and enhancing their operational performance and stability.

* **Prompt development**: The Prompt development module of YNET Loop provides developers with end-to-end support for writing, debugging, optimizing, and version management. Through a visual Playground, it enables real-time interactive testing of prompts, allowing developers to intuitively compare the output of different LLMs.
* **Evaluation**: The YNET Loop evaluation module provides developers with systematic evaluation capabilities, enabling automated multi-dimensional testing of prompts and YNET agents' output, such as accuracy, conciseness, compliance, and more.
* **Observability**: YNET Loop provides developers with observability for the entire execution process, fully recording every stage from user input to AI output, including key stages such as prompt parsing, model invocation, and tool execution, and automatically capturing intermediate results and exceptions.

## Feature list

| **Feature** | **Functional points** |
| --- | --- |
| Prompt debugging | *Playground debugging and comparison <br>* Prompt version management |
| Evaluation | *Manage evaluation sets <br> Management evaluator <br>* Manage experiments |
| Observation | SDK trace reporting <br> * Trace data observation |
| Model | Support integration with OpenAI, Volcengine Ark, and other models |

## Quick Start

### Deployment method 1: Docker deployment (Docker Compose)
>
> Please install and start Docker Engine before you start.

Procedure:

1. Clone the source code.
   Run the following command to obtain the latest version of the YNET Loop source code.

   ```Bash
   # Clone the code
   git clone http://git.ynet.io/Cheetah/Develop/ai/agent/ynet/ynet-loop.git

   # Enter the ynet-loop directory
   cd ynet-loop
   ```

2. Configure a model.
   1. Enter the `ynet-loop` directory.
   2. Edit the file `release/deployment/docker-compose/conf/model_config.yaml`.
   3. Modify the api_key and model fields. Take Volcengine Ark as an example:
      * api_key: Volcengine Ark API Key. Users in China can refer to the [Volcengine Ark documentation](https://www.volcengine.com/docs/82379/1541594), while users outside China can refer to the [BytePlus ModelArk documentation](https://docs.byteplus.com/en/docs/ModelArk/1361424).
      * model: The Endpoint ID of the Volcengine Ark model access point. Users within China can refer to [the Volcengine Ark documentation](https://www.volcengine.com/docs/82379/1099522); users outside China can refer to [the BytePlus ModelArk documentation](https://docs.byteplus.com/en/docs/ModelArk/1099522).
3. Start the service.
   Run the following commands to quickly deploy the open-source version of YNET Loop using Docker Compose.

   ```Bash
   # Start the service (default: development mode)
   # Run in the ynet-loop/ directory
   make compose-up
   ```

4. Access the YNET Loop open-source version through your browser `http://localhost:8082`.

### Deployment method 2: Kubernetes deployment using Helm Chart

> * The Kubernetes cluster has been prepared, the Nginx Ingress add-ons have been enabled, and the Kubectl and Helm tools have been installed.
> * To quickly try it out locally, you can deploy a Kubernetes cluster using Minikube.

Procedure:

1. Run the following command to obtain the Helm Chart package.

   ```Bash
   helm pull oci://docker.io/ynetdev/ynet-loop --version 1.0.0-helm
   tar -zxvf ynet-loop-1.0.0-helm.tgz && cd ynet-loop && rm -f ../ynet-loop-1.0.0-helm.tgz
   ```

2. Configure a model.
   Go to the `ynet-loop` directory and edit the `release/deployment/helm-chart/umbrella/conf/model_config.yaml` file. Configure the following fields, using Volcengine Ark as an example:
   * api_key: Volcengine Ark API Key. Users in mainland China can refer to the [Volcengine Ark documentation](https://www.volcengine.com/docs/82379/1541594), while users outside mainland China can refer to the [BytePlus ModelArk documentation](https://docs.byteplus.com/en/docs/ModelArk/1361424).
   * model: The Endpoint ID of the Volcengine Ark model access point. Users in China can refer to the [Volcengine Ark documentation](https://www.volcengine.com/docs/82379/1099522), while users outside China can refer to the [BytePlus ModelArk documentation](https://docs.byteplus.com/en/docs/ModelArk/1099522).
3. Configure Ingress rules.
   Ingress is used to expose services to external networks. You need to configure the `templates/ingress.yaml` file in the project directory according to the actual cluster situation, manually modify parameters such as ingressClassName, and configure elements such as class, instance, host, and IP allocation.
4. Deploy and start the service.
   Execute the following commands to quickly deploy the open-source version of YNET Loop using Helm.

   ```Bash
   # Run in the ynet-loop/ directory
   make helm-up
   # After the service deployment is complete, check the status of the cluster pods
   make helm-pod
   # Check the service startup logs. If both the app and nginx are running normally, the deployment is successful
   make helm-logf-app
   make helm-logf-nginx
   ```

5. Access the YNET Loop open source edition via a browser.
   The access domain name and URL depend on the domain name and URL assigned to your cluster.
6. Start customizing your YNET Loop project.
   Refer to the examples in the `examples/` directory. Modify `values.yaml` to override the default settings. After making changes, rerun `make helm-up` for the changes to take effect.

> [!WARNING]
> If you want to deploy YNET Loop in a public network environment, it is recommended to assess security risks before you begin, and take corresponding protection measures. Possible security risks include account registration functions, YNET Server listening address configurations, SSRF (Server - Side Request Forgery), and some horizontal privilege escalations in APIs.

## Use the YNET Loop open source version

* Prompt development and debugging: YNET Loop provides a complete prompt development workflow.
* Evaluation: The evaluation functionality of YNET Loop provides standard evaluation data management, an automated evaluation engine, and comprehensive statistics on experimental results.
* Trace reporting and query: YNET Loop supports automatic reporting of traces from prompt debugging sessions created on the platform, enabling real-time tracking of each trace.
* Open-source Edition usage of the YNET Loop SDK: The YNET Loop SDK in three languages is suitable for both commercial and open-source editions. For the Open-source Edition, developers only need to modify some parameter configurations during initialization.

## Developer guide

* System architecture: Learn about the technical architecture and core components of YNET Loop Open-source Edition.
* Startup mode: When installing and deploying YNET Loop Open-source Edition, the default development mode allows backend file modifications without requiring service redeployment.
* Model configuration: YNET Loop Open-source Edition supports various LLM models through the Eino framework. Refer to this document to view the supported model list and learn how to configure models.
* Code development and testing: Learn how to perform secondary development and testing based on YNET Loop Open-source Edition.
* Fault troubleshooting: Learn how to check container status and system logs.

## License

This project uses the Apache 2.0 license. For more details, please refer to the [LICENSE](LICENSE) file.

## Community Contributions

We welcome community contributions. For contribution guidelines, please refer to [CONTRIBUTING](CONTRIBUTING.md) and [Code of conduct](CODE_OF_CONDUCT.md). We look forward to your contributions!

## Security and Privacy

If you identify potential security issues in this project or believe you may have found one, please notify the security team in time.

## Join the Community

We are committed to building an open and friendly developer community. All developers interested in AI Agent development are welcome to join us!

### Issue Reports & Feature Requests

To efficiently track and resolve issues while ensuring transparency and collaboration, we recommend participating through the project's issue and merge request channels.

### Technical Discussion & Communication

Join our technical discussion groups to share experiences with other developers and stay updated with the latest project developments:

* Discord Server: [YNET Community](https://discord.com/invite/sTVN9EVS4B)

* Telegram Group: [YNET](https://t.me/+pP9CkPnomDA0Mjgx)

## Acknowledgments

Thanks to all developers and community members who contributed to the YNET Loop project Special thanks:

* LLM integration support provided by the [Eino](https://github.com/cloudwego/eino) framework team
* High-performance frameworks developed by the [CloudWeGo](https://www.cloudwego.io) team
* All users who participated in testing and feedback
