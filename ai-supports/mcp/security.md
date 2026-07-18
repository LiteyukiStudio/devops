# MCP Security Model

## Threat Model

MCP gives a model a structured way to call platform actions. That improves usability, but it also increases the blast radius of prompt injection, confused-deputy flows, excessive permission grants, and accidental destructive operations.

The MCP layer must be treated as an API surface with the same security bar as the web console.

## Identity and Authorization

Every MCP request must resolve an actor before listing or calling tools.

Supported identities:

| Client | Auth mode | Notes |
| --- | --- | --- |
| Embedded web assistant | Existing browser session | Best for high-risk actions because MFA step-up already relies on sessions |
| External MCP client | Bearer access token | Read and low-risk tools only by default |
| Internal automation | Service token | Must be explicit, scoped, audited, and rate limited |

Rules:

- Tool availability is filtered by user role, token scopes, project membership, feature flags, and risk policy.
- Tool calls must run the same project-level authz checks as REST endpoints.
- Access tokens should use existing scopes such as `project:read`, `build:trigger`, and `deployment:release`.
- External bearer tokens must not satisfy browser-session-only step-up requirements.

## Risk Levels

| Risk | Meaning | Default policy |
| --- | --- | --- |
| `read` | Read-only, secret-safe output | Allowed with matching read scope |
| `low` | Reversible or low-impact mutation | Requires matching write scope and audit |
| `medium` | Starts work or changes runtime behavior | Requires preflight confirmation |
| `high` | Release, rollback, delete, credential, billing, or admin change | Requires preflight confirmation and MFA step-up |
| `critical` | Runtime exec, terminal, data export, retention cleanup, security policy change | Browser session only, MFA step-up, explicit confirmation, strong audit |

## Confirmation Flow

High-impact tools must never execute on the first model call.

Recommended flow:

```text
tools/call luna.release.create
  -> server validates input and permissions
  -> server creates a pending intent
  -> server returns confirmation_required
  -> frontend shows human-readable diff and risk
  -> user confirms and completes MFA step-up if required
  -> tools/call luna.confirm.execute with confirmationToken
  -> server revalidates digest, actor, authz, expiry, and resource versions
  -> server executes the original operation
```

Pending intent fields:

| Field | Purpose |
| --- | --- |
| `id` | Server-generated confirmation ID |
| `actorId` | User who requested the action |
| `sessionId` | Required for browser-session high/critical tools |
| `toolName` | Original MCP tool |
| `inputDigest` | SHA-256 of canonicalized input |
| `resourceDigest` | Optional digest of current target resource state |
| `risk` | Tool risk level |
| `summary` | Human-readable action summary |
| `requiredPhrase` | Optional typed phrase for destructive operations |
| `expiresAt` | Short expiry, recommended 5 minutes |
| `usedAt` | Replay protection |

Execution must fail if:

- token is expired or already used
- actor/session changed
- input digest changed
- current resource version no longer matches the preflight
- required MFA assertion is missing
- authz no longer allows the action

## Step-Up Mapping

Reuse existing step-up purposes where possible:

| MCP action family | Existing purpose |
| --- | --- |
| runtime exec / terminal | `runtime_exec`, `runtime_terminal` |
| persistent data export | `data_export` |
| secret or variable-set mutation | `secret_update` |
| registry credential mutation | `registry_credential_update` |
| kubeconfig or runtime cluster mutation | `kubeconfig_update` |
| auth provider mutation | `auth_provider_update` |
| user admin mutation | `user_admin_update` |
| MFA and security settings | `mfa_manage`, `security_settings_update` |
| data retention cleanup | `data_retention_cleanup` |

Add new purpose values before enabling high-risk release tools:

| MCP action family | Proposed purpose |
| --- | --- |
| release creation | `deployment_release` |
| release rollback | `deployment_rollback` |
| build trigger in protected environments | `build_trigger` |

Build trigger may start as `medium` with confirmation. If a release updates production traffic or performs rollback, classify it as `high` and require the dedicated release step-up purpose.

## Output Safety

All MCP tool outputs must be projected and redacted.

Requirements:

- never return secret values, tokens, passwords, kubeconfig contents, recovery codes, private keys, or raw credential payloads
- truncate logs by default and provide pagination or tail parameters
- strip ANSI control sequences unless explicitly requested by a trusted console view
- cap list page size
- include resource IDs and stable status codes, not only prose
- avoid returning full database rows
- treat user-provided fields as untrusted text

## Audit Requirements

Every MCP tool call should write an audit record.

Minimum fields:

- actor ID
- auth mode: `session`, `access_token`, or `service_token`
- MCP client name/version if provided
- tool name
- risk level
- project/application/resource IDs
- sanitized input digest
- confirmation ID when used
- result status
- latency
- request ID / trace ID

For runtime commands, audit the command byte length and SHA-256 digest, not the full command text.

## Rate Limits

Apply layered limits:

- per actor
- per access token
- per IP or trusted client
- per tool
- per project space

Suggested defaults:

| Tool family | Limit |
| --- | --- |
| read/list | generous, paginated |
| log tail | moderate, stream-safe |
| build trigger | low burst, project-scoped |
| release/rollback | very low burst |
| confirmation attempts | strict and audited |

## Prompt Injection Controls

MCP tools should assume repository files, build logs, events, and application descriptions may contain hostile instructions.

Controls:

- tag untrusted content in tool responses
- keep tool descriptions explicit about allowed behavior
- require confirmation for state changes
- do not let tools chain into unrelated high-risk actions automatically
- require resource IDs instead of acting only on fuzzy natural-language names when mutating state
