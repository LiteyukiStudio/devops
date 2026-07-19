# MCP 接入设计

## 目标

Luna DevOps 后续需要提供安全的 MCP 接口，供外部 AI 助手、Agent 平台、IDE 和其他 MCP client 调用。MCP 层把经过筛选的后端能力包装成 tools，让外部 Agent 可以查看项目状态、解释部署情况、触发构建、准备发布和诊断问题。

第一阶段产品实现应先做 `ai-supports/assistant/design.md` 里的平台内嵌 AI 助手。外部 MCP 应在 shared Tool Kernel、确认流程、审计和输出脱敏稳定后再开放。

MCP 是外部集成协议，不是控制台替代品。高影响确认、MFA step-up、可视化检查和长任务监控仍应回到 Luna DevOps 控制台完成。

## 参考

本设计参考 Model Context Protocol 的基本形态：

- MCP tools 是模型可调用的结构化函数。
- MCP resources 是服务端暴露给 client 读取的上下文数据。
- MCP prompts 是可复用的 prompt 模板。
- 对外托管的 MCP 服务应具备明确授权和用户同意机制。

参考链接：

- MCP specification：`https://modelcontextprotocol.io/specification/2025-06-18`
- MCP tools：`https://modelcontextprotocol.io/specification/2025-06-18/server/tools`
- MCP authorization：`https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization`
- 官方 MCP servers 仓库：`https://github.com/modelcontextprotocol/servers`

## 推荐架构

外部 MCP：

```text
外部 Agent 平台 / MCP client
  -> POST /api/v1/mcp
     -> MCP transport 和 JSON-RPC dispatcher
     -> Luna Access Token authentication
     -> shared Tool Kernel
     -> 现有 handlers/services/providers
     -> response projector 和 redactor
```

内部助手第一版不走 MCP：

```text
前端 AI 小窗
  -> /api/v1/assistant/*
  -> ADK runtime adapter
  -> shared Tool Kernel
  -> 现有 handlers/services/providers
```

两条路径都必须收敛到 shared Tool Kernel：

```text
内部 ADK tool call / 外部 MCP tools.call
  -> adapter layer，ADK tool adapter 或 MCP JSON-RPC dispatcher
  -> tool registry
  -> risk 和 confirmation guard
  -> 现有 authz scope resolver
  -> REST bridge / service adapter
  -> response projector 和 redactor
```

第一版 MCP 可以作为 `cmd/api` 内的 API 模块运行，便于复用：

- trusted proxy 和请求身份处理
- Access Token authentication
- `authz.RequiredAccessTokenScope`
- 项目空间成员关系检查
- MFA step-up assertions
- audit log
- secret redaction 和安全错误响应
- 现有限流 helpers

如果后续 MCP 流量和发布节奏需要独立，可以再考虑 `cmd/mcp` 服务。即使拆服务，也应复用同一个 Tool Kernel 和 authz package。

## 鉴权策略

外部 MCP 使用现有 Luna DevOps Access Token。

```text
MCP client
  -> Authorization: Bearer <Luna Access Token>
  -> /api/v1/mcp
  -> tools/list 按 token scopes 和 risk policy 过滤
  -> tools/call 重新校验 token、scope、project RBAC 和 tool policy
```

MCP 权限与 REST API 权限保持一致：

- `tools.yaml.authz` 映射到现有 Access Token scopes。
- `tools/list` 必须隐藏 token 无权调用的 tools。
- `tools/call` 必须重复 scope 校验，不能信任之前的 `tools/list` 结果。
- 每个 project-scoped tool 都必须继续检查项目空间成员关系和角色。
- 高风险 MCP 行为再由 risk policy 和 confirmation 额外约束。

第一版不要引入第二套 MCP 专用权限模型。

## 内部模块形态

建议 Go package：

```text
internal/mcp/
  server.go             MCP transport、initialize、tools/list、tools/call
  registry.go           tool descriptor 加载和校验
  descriptor.go         declaration structs
  bridge.go             REST bridge
  auth.go               session/JWT/PAT identity extraction
  confirmation.go       preflight token、digest、expiry、replay guard
  projection.go         response shaping、truncation、redaction
  audit.go              MCP audit helpers
  rate_limit.go         per-user/client/tool throttling
```

路由入口：

```go
v1.POST("/mcp", handlers.HandleMCP)
```

第三方 MCP client 使用 `Authorization: Bearer <access-token>` 和 scoped token。平台内嵌助手使用 `/api/v1/assistant/*` 和 browser session cookie，不走 MCP。

## 声明驱动的 Tool Registry

MCP registry 应从第一方声明文件加载 tool descriptor。`ai-supports/mcp/tools.yaml` 是设计种子，生产代码可以把等价声明 embed 到 Go 代码里。

每个 tool descriptor 应包含：

| 字段 | 用途 |
| --- | --- |
| `name` | 稳定 MCP tool name，例如 `luna.projects.list` |
| `title` | 面向用户展示的标题 |
| `description` | 面向模型的简短使用说明 |
| `category` | 分组和权限归类 |
| `risk` | `read`、`low`、`medium`、`high` 或 `critical` |
| `authz` | 现有 Luna DevOps action scope，例如 `project:read` |
| `http` | 现有 REST method 和 path template |
| `source` | 可选 OpenAPI 来源，例如 `operationId` |
| `inputSchema` | 暴露给 MCP client 的 JSON Schema |
| `confirmation` | mutation 的确认和 step-up 策略 |
| `output` | 脱敏、截断和字段投影规则 |

启动时必须校验 registry：

- tool name 唯一
- method 和 path 真实存在
- required authz action 存在
- high/critical tools 必须配置 confirmation policy
- 不返回 secret 字段
- list tools 必须有 pagination 默认值和最大 page size

## 为什么不从 OpenAPI 全量生成 tools

OpenAPI 适合复用 schema，但直接把 OpenAPI 全量导出为 MCP tools 对 Luna DevOps 来说太宽。

API 里包含删除、计费调整、安全设置、terminal 授权、runtime command execution、data export、credential update、cleanup 等高风险接口。这些接口需要产品级 guardrails，通用生成器很难正确推断。

推荐做法：

1. 逐步给 OpenAPI 补 `operationId`。
2. 使用 OpenAPI schemas，避免重复手写请求和响应结构。
3. 保留 `tools.yaml` 作为 allowlist。
4. 当对应 OpenAPI operation 存在时，在 tool 上补 `source.openapi` 和 `source.operationId`。
5. 只为选中的 operations 生成 descriptor stub。
6. 每个新 tool 启用前都必须审查 risk、authz、audit、confirmation 和 output redaction。

`tools.yaml` 不是 OpenAPI 文档。OpenAPI 继续作为面向开发者、SDK 和 HTTP 集成的完整 REST contract；`tools.yaml` 是面向 Agent 的 allowlist，用来描述 tool 语义、风险等级、确认策略和输出规则。

## Tool 分类

初始分类应贴近控制台心智模型：

- `workspace`：dashboard、project spaces、applications、members
- `source`：Git accounts、repositories、branches、repository bindings
- `registry`：artifact registries、credentials summary、image records
- `build`：build jobs、build runs、logs、trigger/cancel/retry
- `deployment`：deployment targets、releases、rollback
- `runtime`：runtime clusters、workload resources、events、logs
- `gateway`：gateway routes、domain checks、access URLs
- `billing`：summary、ledger、usage records
- `events`：platform events、notifications
- `system`：platform settings、users、identity providers

MVP 只启用 read 和低风险构建/部署辅助能力：

- list/read projects 和 applications
- 查看 topology、gateway routes、releases、build runs、events
- 读取截断后的 build logs
- 触发 build 但需要 confirmation
- 输出 release plan dry-run，不直接发布

## MCP Resources

Resources 用于稳定上下文，不用于直接执行动作：

| Resource URI | 内容 |
| --- | --- |
| `luna://workspace/{projectId}` | 当前 actor 可见的 project summary 和 member role |
| `luna://application/{projectId}/{applicationId}` | application summary、bindings、targets、last release |
| `luna://deployment/{projectId}/{targetId}` | deployment target runtime config summary |
| `luna://build-run/{projectId}/{runId}` | build result、image 和紧凑 log tail |
| `luna://platform/capabilities` | enabled providers、feature switches 和 MCP tool availability |

Resources 必须使用与等价 REST read 相同的 authz 检查。

## MCP Prompts

Prompts 用于帮助 client 产出一致、安全的操作流程：

- `luna_deploy_web_project`：引导用户从 repository binding 到 build 和 release
- `luna_diagnose_failed_build`：检查 failed build run、logs、repository settings 和 registry status
- `luna_explain_project_topology`：总结 applications、dependencies、gateway routes 和 releases
- `luna_prepare_release`：生成 release checklist，并在 mutation 前要求 confirmation

Prompts 可以由代码生成，也可以从 `ai-supports/skills` 下的 skills 转换。

## 推进计划

### 阶段 0：声明和审查

- 保持 `ai-supports/mcp/tools.yaml` 作为规范提案。
- 根据后端 authz 和产品风险审查每个 tool。
- 为选中的 OpenAPI operation 补 `operationId`。

### 阶段 1：先做内部助手

- 在 `/api/v1/assistant/*` 后实现 ADK Go 内嵌助手。
- 建立 shared Tool Kernel 和 read-only tools。
- 验证 session handling、project RBAC、tool audit 和 output redaction。

### 阶段 2：只读 MCP

- 实现 `/api/v1/mcp` 的 initialize、tools/list、tools/call。
- 启用 dashboard、projects、applications、build runs、releases、gateway routes、events、billing summary 等 read-only tools。
- 为每次 MCP call 写 audit。
- 增加 response redaction 和 log truncation。

### 阶段 3：带确认的 mutation

- 启用 build trigger、build cancel 等 low/medium mutation tools。
- 增加服务端 pending intent 和 explicit confirmation。
- 内部助手返回平台小窗可展示的 inline confirmation data。
- 外部 MCP 返回平台 confirmation URL，不要求 MCP client 自己实现复杂确认 UI。

### 阶段 4：高风险操作

- 只有在 MFA step-up 集成后，再考虑 releases、rollbacks、data export authorization 和部分 admin tools。
- runtime exec 和 terminal 对外部 bearer-token client 保持 disabled，除非已经存在 browser session approval flow。

### 阶段 5：Tool 生成

- 从 OpenAPI operation IDs 生成 tool stubs。
- 生成的 tools 默认 disabled，审查后再加入 allowlist。

## 非目标

- 不把 MCP 做成隐藏 admin API。
- 不暴露 raw kubeconfig、registry credentials、OAuth secrets、session tokens、recovery codes。
- 不让模型默认执行任意 shell command。
- 不允许外部 MCP client 绕过 browser-only MFA step-up 执行 critical operations。
- 不把所有 OpenAPI paths 全量发布为 tools。
