# Luna DevOps AI Skills

这个目录存放与 `luna` CLI 配套的 AI Skills。Agent 通过 CLI 使用 Luna DevOps，不直接调用平台 REST API、Kubernetes API 或第三方 Provider API。

当前状态为**预发布设计**：应先完成 [`notes/cli-spec.md`](../notes/cli-spec.md) 中的 CLI、机器可读 Help、稳定输出和鉴权能力，再进行 Skills 安装与真实场景验证。CLI 尚未可用时，这些 Skills 只用于审阅和规划，不能假装执行平台命令。

## 目录结构

```text
ai-supports/
  skills/
    luna-devops-router/
      SKILL.md          skill 路由器，先判断任务再按需加载模块
    luna-devops-cli/
      SKILL.md          CLI 可用性、命令发现、输出和安全契约
    luna-devops-workspace/
      SKILL.md          工作台、项目空间和成员
    luna-devops-source/
      SKILL.md          代码源、仓库、分支和 Webhook
    luna-devops-registry/
      SKILL.md          镜像站、镜像仓库和凭据
    luna-devops-build/
      SKILL.md          构建、构建模板、变量和日志
    luna-devops-deployment/
      SKILL.md          应用、部署配置、发布和回滚
    luna-devops-topology/
      SKILL.md          服务依赖、自定义拓扑和 ServiceBinding
    luna-devops-runtime/
      SKILL.md          集群、Kubernetes 资源和事件
    luna-devops-gateway/
      SKILL.md          访问入口、域名、证书和 Gateway API
    luna-devops-billing/
      SKILL.md          账单、余额、用量和费率
    luna-devops-notifications/
      SKILL.md          通知渠道、模板、规则和投递
    luna-devops-security/
      SKILL.md          认证、MFA、OIDC、用户和 Access Token
    luna-devops-system/
      SKILL.md          站点设置、应用市场、数据保留和系统组件
    luna-devops-debugging/
      SKILL.md          跨模块诊断和排障
```

## 设计原则

- 先完成 CLI，再启用和验证 Skills。
- CLI 的 `help catalog agent=true` 和 `help command ... agent=true` 是命令、参数、输出和风险元数据的事实来源。
- Skills 只负责意图路由、操作顺序、风险控制和结果解释，不复制完整命令手册。
- Agent 只调用 `luna`，不能绕过 CLI 直接编排平台或第三方 API。
- CLI 和后端继续执行 OAuth、Scope、RBAC、MFA、审计和 Secret 脱敏。
- 删除、计费、secret、runtime exec、terminal、data export 等操作按高风险处理。
- mutation 必须先读取当前状态、说明影响并等待用户明确确认。
- Agent 每条命令都强制使用 `agent=true`，由 CLI 统一启用结构化输出、禁用交互和颜色，并施加分页、轮询、流式读取和响应体大小上限；不能依赖用户的默认输出模式。
- 命令输出必须短、结构化、可审计、脱敏。
- skills 按模块渐进加载：先加载 `luna-devops-router` 判断意图，再只加载当前任务需要的一个或少数模块 skill。
- 所有 `SKILL.md` 的元数据描述、标题、规则和工作流统一使用中文编写；命令名、参数名、JSON key、API 枚举、Scope 和稳定错误码保留英文技术标识。

## Skills 覆盖

当前 Skills 按平台能力拆成 15 个模块，目标是覆盖 Luna DevOps 的主要用户路径和管理员路径。完整接口覆盖由 CLI 与 OpenAPI 保证，Skills 不重复罗列所有 endpoint。

| 模块 | 覆盖能力 |
| --- | --- |
| `luna-devops-router` | 意图识别、模块分流、按需加载 |
| `luna-devops-cli` | CLI 可用性、上下文、命令发现、输出和安全操作 |
| `luna-devops-workspace` | 看板、项目空间、成员、置顶和排序 |
| `luna-devops-source` | Git provider、Git account、仓库、分支、Webhook、代码源绑定 |
| `luna-devops-registry` | 镜像站、凭据、镜像模板、镜像仓库和 tag |
| `luna-devops-build` | build run、build job、构建模板、变量、日志、触发和取消 |
| `luna-devops-deployment` | 应用、部署目标、运行配置、发布、重启、回滚 |
| `luna-devops-topology` | 项目拓扑、ServiceBinding、自定义依赖边 |
| `luna-devops-runtime` | runtime cluster、Kubernetes 资源、YAML、事件、Pod 状态 |
| `luna-devops-gateway` | Gateway route、域名检查、TLS、证书、访问入口 |
| `luna-devops-billing` | 余额、账单、用量、费率、流水、网关流量 |
| `luna-devops-notifications` | 通知渠道、模板、规则、投递和测试 |
| `luna-devops-security` | 登录、MFA、OIDC、OAuth app、用户、Access Token、scope |
| `luna-devops-system` | 站点设置、公开配置、应用市场、系统组件、数据保留 |
| `luna-devops-debugging` | 构建、部署、网关、拓扑、账单、通知、权限排障 |

这些 Skill 不替代 CLI Help、后端 RBAC、Scope、审计、MFA 和脱敏逻辑。

## 调用拓扑

```text
用户或 AI Agent
  -> Luna DevOps Skills
  -> luna CLI
  -> Luna DevOps REST API
  -> 后端权限、MFA、审计和业务服务
```

## 启用门禁

只有满足以下条件后，才能把 Skills 标记为可用：

1. `luna version show agent=true`、机器可读 Help 和多实例 context 已稳定。
2. CLI 公开 API 覆盖门禁达到 100%。
3. JSON 输出、错误结构和退出码完成兼容性测试。
4. OAuth、Device Code、Access Token 和 Step-up MFA 已完成集成测试。
5. 使用真实测试实例完成各领域 Skill 的只读、变更、失败和权限场景评估。
