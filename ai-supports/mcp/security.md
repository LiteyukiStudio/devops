# MCP 安全模型

## 威胁模型

MCP 让模型可以用结构化方式调用平台动作。这会提升易用性，也会放大 prompt injection、confused deputy、过度授权和误操作的风险。

MCP 层必须视为正式 API surface，安全要求应和 Web Console 一致。

## 身份和授权

每个 MCP request 在列出或调用 tools 前，都必须解析出 actor。

支持的身份类型：

| Client | Auth mode | 说明 |
| --- | --- | --- |
| 内嵌 Web assistant | Existing browser session | 适合高风险动作，因为 MFA step-up 已经依赖 session |
| 外部 MCP client | Bearer access token | 默认只开放 read 和低风险 tools |
| 内部 automation | Service token | 必须显式、scoped、audited、rate limited |

规则：

- tool 可见性由 user role、token scopes、project membership、feature flags 和 risk policy 共同决定。
- tool call 必须使用与 REST endpoint 相同的项目级 authz 检查。
- Access Token 复用现有 scopes，例如 `project:read`、`build:trigger`、`deployment:release`。
- 外部 bearer token 不能满足 browser-session-only step-up 要求。
- MCP 权限第一版复用 Luna DevOps Access Token scopes，不新增 MCP 专属权限模型。
- `X-Luna-Agent-Client` 这类请求头只能影响展示形态，不能作为安全边界。后端必须基于 auth mode、session、token、actor binding 和 pending intent 校验。

## 风险等级

| 风险等级 | 含义 | 默认策略 |
| --- | --- | --- |
| `read` | 只读，输出已脱敏 | 有匹配 read scope 即可 |
| `low` | 可逆或低影响 mutation | 需要匹配 write scope 并审计 |
| `medium` | 启动任务或改变运行状态 | 需要 preflight confirmation |
| `high` | release、rollback、delete、credential、billing、admin change | 需要 preflight confirmation 和 MFA step-up |
| `critical` | runtime exec、terminal、data export、retention cleanup、security policy change | 仅 browser session，MFA step-up，明确确认，强审计 |

## 确认流程

高影响 tools 不能在第一次模型调用时直接执行。

内部助手流程：

```text
assistant 请求 luna.release.create
  -> server 校验 input 和 permissions
  -> server 创建 pending intent
  -> server 返回 confirmation_required
  -> 前端 AI 小窗展示 diff 和 risk
  -> 用户确认，并在需要时完成 MFA step-up
  -> 前端调用 /api/v1/assistant/confirmations/{id}/approve
  -> server 重新校验 digest、actor、authz、expiry、resource version
  -> server 执行原始操作
```

外部 MCP 流程：

```text
tools/call luna.release.create
  -> server 校验 access token、scope、input、project RBAC、risk policy
  -> server 创建 pending intent
  -> server 返回 confirmation_required 和 confirmationUrl
  -> 用户打开 Luna DevOps confirmation page
  -> 平台要求登录，并在需要时完成 MFA step-up
  -> 用户 approve 或 reject
  -> MCP client 可调用 luna.confirmation.get 轮询结果
```

外部 MCP client 不要求实现复杂确认 UI。如果 client 不能处理 confirmation URL 或轮询，只能使用 read-only tools，不能执行高风险 mutation。

Pending intent 字段：

| 字段 | 用途 |
| --- | --- |
| `id` | 服务端生成的 confirmation ID |
| `actorId` | 请求操作的用户 |
| `sessionId` | browser-session high/critical tools 需要绑定 |
| `accessTokenId` | 外部 MCP bearer-token 请求需要绑定 |
| `toolName` | 原始 MCP tool |
| `canonicalInput` | 服务端保存的规范化输入；确认执行时不能接收替换输入 |
| `inputDigest` | canonicalized input 的 SHA-256 |
| `resourceDigest` | 当前目标资源状态摘要，可选 |
| `risk` | tool 风险等级 |
| `summary` | 面向用户的操作摘要 |
| `requiredPhrase` | destructive operations 可选确认短语 |
| `urlTokenHash` | external confirmation URL 中一次性 token 的 hash |
| `expiresAt` | 短过期时间，建议 5 分钟 |
| `usedAt` | 防重放 |

执行必须在以下情况失败：

- token 过期或 pending intent 已使用
- actor/session/access token 发生变化
- access token 已撤销或不再具备 required scope
- input digest 变化
- 当前资源版本和 preflight 时不一致
- 缺少 required MFA assertion
- authz 不再允许该操作

确认执行必须使用服务端保存的 `canonicalInput`，client 只能传 confirmation ID。这样可以避免“用户确认 A，实际执行 B”。

外部 confirmation URL 打开后，登录确认的用户必须和 pending intent actor 一致。第一版禁止跨用户确认。

推荐外部 MCP 响应：

```json
{
  "status": "confirmation_required",
  "confirmationId": "cfm_xxx",
  "confirmationUrl": "https://devops.example.com/confirmations/cfm_xxx?token=once_xxx",
  "expiresAt": "2026-07-19T12:00:00Z",
  "summary": "即将回滚 release rel_xxx",
  "risk": "high"
}
```

URL token 必须随机、短期、一次性，并且数据库只存 hash。

## Step-Up 映射

优先复用现有 step-up purpose：

| MCP action family | Existing purpose |
| --- | --- |
| runtime exec / terminal | `runtime_exec`、`runtime_terminal` |
| persistent data export | `data_export` |
| secret or variable-set mutation | `secret_update` |
| registry credential mutation | `registry_credential_update` |
| kubeconfig or runtime cluster mutation | `kubeconfig_update` |
| auth provider mutation | `auth_provider_update` |
| user admin mutation | `user_admin_update` |
| MFA and security settings | `mfa_manage`、`security_settings_update` |
| data retention cleanup | `data_retention_cleanup` |

启用高风险 release tools 前，需要新增 purpose：

| MCP action family | Proposed purpose |
| --- | --- |
| release creation | `deployment_release` |
| release rollback | `deployment_rollback` |
| build trigger in protected environments | `build_trigger` |

build trigger 可以先按 `medium` 处理并要求 confirmation。如果 release 会更新生产流量或执行 rollback，应按 `high` 处理，并要求专用 release step-up purpose。

## 输出安全

所有 MCP tool 输出都必须经过 projection 和 redaction。

要求：

- 不返回 secret values、tokens、passwords、kubeconfig contents、recovery codes、private keys、raw credential payloads。
- logs 默认截断，并提供 pagination 或 tail 参数。
- 除非是可信 console view，否则去除 ANSI control sequences。
- 限制 list page size。
- 输出 resource IDs 和稳定 status codes，不只输出自然语言。
- 避免返回完整数据库 row。
- 用户提供的字段视为 untrusted text。

## 审计要求

每次 MCP tool call 都应写 audit record。

最小字段：

- actor ID
- auth mode：`session`、`access_token` 或 `service_token`
- MCP client name/version，如果提供
- tool name
- risk level
- project/application/resource IDs
- sanitized input digest
- confirmation ID，如果使用了确认
- result status
- latency
- request ID / trace ID

runtime commands 只记录 command byte length 和 SHA-256 digest，不记录完整命令正文。

## 限流

需要分层限流：

- per actor
- per access token
- per IP 或 trusted client
- per tool
- per project space

建议默认值：

| Tool family | Limit |
| --- | --- |
| read/list | 宽松，但必须分页 |
| log tail | 中等，适合 stream |
| build trigger | 较低 burst，project-scoped |
| release/rollback | 很低 burst |
| confirmation attempts | 严格并审计 |

## 外部 MCP 兼容策略

外部 MCP 使用保守默认值：

| 风险等级 | 外部 MCP bearer-token 策略 |
| --- | --- |
| `read` | matching Access Token scope 即可 |
| `low` | matching scope + audit |
| `medium` | 返回 `confirmation_required`，平台确认后才执行 |
| `high` | 返回带平台 URL 的 `confirmation_required`，按配置要求 browser login 和 MFA |
| `critical` | 对外部 MCP disabled，除非已有专门 browser-session approval flow |

MCP `tools/list` 必须按 Access Token scope 和 risk policy 过滤。MCP `tools/call` 必须重复所有检查。

## Prompt Injection 防护

MCP tools 必须假设 repository files、build logs、events 和 application descriptions 里可能包含恶意指令。

控制措施：

- 在 tool responses 中标记 untrusted content。
- tool descriptions 要明确工具允许做什么。
- state changes 必须要求 confirmation。
- 不允许 tools 自动串联到无关高风险动作。
- mutation 必须使用 resource IDs，不要只根据模糊自然语言名称执行。
