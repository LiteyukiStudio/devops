# Luna CLI

Luna CLI 是 Luna DevOps 的命令行客户端，既服务于日常终端操作，也为自动化 Agent 提供稳定的 JSON 输入输出契约。

命令使用固定的两级结构：

```text
luna <工具分类> <具体工具> key=value
```

例如，机器可读帮助目录采用：

```bash
luna help catalog query=project limit=5 output=json interactive=false
```

## 当前开发状态

CLI 目前处于 `0.1.0` 开发阶段。源码已经包含：

- 多实例、上下文和默认项目空间的配置模型；
- Access Token 登录、校验和本地凭据存储基础能力；
- `key=value`、JSON、文件和标准输入参数解析；
- 人类可读输出与版本化 JSON Envelope；
- 本地帮助、上下文、项目空间和 Completion 命令注册；
- 根据 OpenAPI 契约注册全部 109 个已登记操作；
- npm 包与 Bun 独立二进制的统一入口、CI、打包、全局安装 smoke 和发布门禁。

共享契约和 API Client 会被打包进 npm 与 Bun 制品，不要求用户安装 monorepo 工作区。项目尚未完成首次公开发布，因此文档中的安装命令要等 `cli-v*` 版本发布后才可使用。

## 设计边界

- CLI 只调用 Luna DevOps 后端 API，不直接编排 Kubernetes、GitHub、Gitea 或镜像站接口。
- 自动化应设置 `output=json interactive=false`，只解析 `stdout` 的 JSON；诊断信息写入 `stderr`。
- 本地配置默认放在 `~/.luna/`。测试和 CI 必须使用临时 `LUNA_HOME`，不能读取真实用户凭据。
- 中风险操作在交互终端中使用统一确认；非交互模式必须显式传入 `yes=true`。
- 高风险 API 在服务端执行计划协议完成前直接拒绝执行，`yes=true` 也不能绕过。
- CLI 和平台使用独立版本。兼容性由服务端能力协商决定，不只比较版本号。

## 下一步

首次公开发布前还需要完成：

1. 补齐尚未进入 OpenAPI 的公开后端路由，并完成完整命令覆盖率测试。
2. 接入服务端能力协商、Authorization Code + PKCE、Device Code 和 Bearer Step-up MFA。
3. 完成 SSE、WebSocket、下载和服务端执行计划协议。
4. 在 npm 配置 Trusted Publisher，并保护 GitHub `npm` Environment。
5. 接入 Apple Developer ID、公证和 Windows Authenticode 后，再把桌面二进制加入稳定版本。

具体安装方式见[安装与使用](./installation)，制品校验见[发布与制品验证](./release-security)。
