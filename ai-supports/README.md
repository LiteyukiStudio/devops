# Luna DevOps AI 支持方案

这个目录用于存放 Luna DevOps 面向 AI 能力的设计文档、工具声明和配套 skills。

第一阶段目标是平台内嵌 AI 助手。外部 MCP 仍然保留在设计里，但应在内部助手、共享 Tool Kernel、二次确认、审计和输出脱敏稳定后再开放。

AI 能力应把 Luna DevOps 后端能力包装成安全 tools，同时继续使用现有 REST API、session/access-token 鉴权、RBAC、审计日志、MFA step-up 和 secret 脱敏作为安全边界。

## 目录结构

```text
ai-supports/
  assistant/
    design.md           平台内嵌 AI 助手设计
  mcp/
    design.md           MCP 接入设计
    security.md         风险、确认、审计和安全策略
    tools.yaml          MCP tool 白名单声明
  skills/
    luna-devops-operator/
      SKILL.md          平台运维 skill
    luna-devops-deployment/
      SKILL.md          构建和部署 skill
    luna-devops-debugging/
      SKILL.md          诊断和排障 skill
```

## 设计原则

- 先做内部助手，再开放外部 MCP。
- 内部 ADK tools 和未来外部 MCP tools 共用同一个 Tool Kernel。
- 第一版只暴露小而实用的工具集，不自动发布所有 REST endpoint。
- 复用现有后端权限模型，AI tools 不能绕过 Luna DevOps RBAC。
- 删除、计费、secret、runtime exec、terminal、data export 等操作按高风险处理。
- mutation 优先走 preflight 和 confirmation，不直接执行。
- 内部助手使用平台内嵌确认弹窗；外部 MCP 返回平台 confirmation URL。
- tool 输出必须短、结构化、可审计、脱敏。
- 先写工具声明，再在声明后实现 adapter。

## 当前方向

内部助手：

```text
前端 AI 小窗
  -> /api/v1/assistant/*
  -> 后端 Agent runtime，第一版使用 ADK Go
  -> shared Tool Kernel
  -> 现有 Luna DevOps services/API
```

未来外部接入：

```text
外部 Agent 平台 / MCP client
  -> /api/v1/mcp
  -> 现有 Luna DevOps Access Token
  -> shared Tool Kernel
  -> 现有 Luna DevOps services/API
```

