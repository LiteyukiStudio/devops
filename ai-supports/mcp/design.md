# MCP Integration Design

## Goal

Luna DevOps should provide a safe MCP interface for AI assistants and external MCP clients. The MCP layer wraps selected existing backend API capabilities as tools, so users can ask an assistant to inspect workspaces, explain deployment state, trigger builds, prepare releases, and diagnose failures without learning every console page first.

The MCP feature is an accessory to the platform, not a replacement for the console. The console remains the best place for high-impact confirmation, MFA step-up, visual review, and long-running operation monitoring.

## References

The design follows the shape of the Model Context Protocol:

- MCP tools are model-callable functions with structured input schemas and structured results.
- MCP resources are server-exposed data that clients may read for context.
- MCP prompts are reusable prompt templates.
- Hosted MCP integrations should use explicit authorization and keep user consent visible.

Useful references:

- MCP specification: `https://modelcontextprotocol.io/specification/2025-06-18`
- MCP tools: `https://modelcontextprotocol.io/specification/2025-06-18/server/tools`
- MCP authorization: `https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization`
- Official MCP servers repository: `https://github.com/modelcontextprotocol/servers`

## Recommended Architecture

```text
MCP client / embedded assistant
  -> POST /api/mcp
     -> MCP transport and JSON-RPC dispatcher
     -> JWT/session authentication
     -> MCP tool registry
     -> risk and confirmation guard
     -> existing authz scope resolver
     -> REST bridge / service adapter
     -> existing handlers/services/providers
     -> response projector and redactor
```

The MVP should run inside `cmd/api` as an API module. This keeps deployment simple and lets MCP reuse:

- trusted proxy and request identity handling
- session and access-token authentication
- `authz.RequiredAccessTokenScope`
- project membership checks
- MFA step-up assertions
- audit log writes
- secret redaction and safe error responses
- existing rate limit helpers

An independent `cmd/mcp` service can be considered later if MCP traffic becomes operationally separate. It should still call the API through a first-party internal client and reuse the same authz package.

## Internal Module Shape

Recommended Go package layout:

```text
internal/mcp/
  server.go             MCP transport, initialize, tools/list, tools/call
  registry.go           tool descriptor loader and validation
  descriptor.go         declaration structs
  bridge.go             REST bridge for existing API endpoints
  auth.go               session/JWT/PAT identity extraction
  confirmation.go       preflight token, digest, expiry, replay guard
  projection.go         response shaping, truncation, redaction
  audit.go              MCP audit helpers
  rate_limit.go         per-user/client/tool throttling
```

Router entry:

```go
v1.POST("/mcp", handlers.HandleMCP)
```

For browser-embedded assistants, the frontend can call `/api/v1/mcp` with the normal session cookie and CSRF protection. For third-party MCP clients, use `Authorization: Bearer <access-token>` and require scoped tokens.

## Declaration-Driven Tool Registry

The MCP registry should load descriptors from a first-party declaration file. `ai-supports/mcp/tools.yaml` is the design seed; production can embed a generated equivalent in Go.

Each tool descriptor should include:

| Field | Purpose |
| --- | --- |
| `name` | Stable MCP tool name, for example `luna.projects.list` |
| `title` | Human-readable title |
| `description` | Short model-facing usage guidance |
| `category` | Navigation and permission grouping |
| `risk` | `read`, `low`, `medium`, `high`, or `critical` |
| `authz` | Existing Luna DevOps action scope, for example `project:read` |
| `http` | Existing REST method and path template |
| `inputSchema` | JSON Schema exposed to MCP clients |
| `confirmation` | Confirmation and step-up policy for mutations |
| `output` | Redaction, truncation, and field projection rules |

The registry must validate descriptors at startup:

- tool names are unique
- methods and paths are known
- required authz actions exist
- high/critical tools have confirmation policy
- secret fields are never returned
- list tools have pagination defaults and maximum page size

## Why Not Generate Every Tool From OpenAPI?

OpenAPI is useful for schema reuse, but a direct OpenAPI-to-MCP export is too broad for Luna DevOps.

The API contains destructive endpoints, billing adjustments, security settings, terminal authorization, runtime command execution, data export, credential update, and cleanup APIs. These need product-specific guardrails that generic generators cannot infer well.

Recommended approach:

1. Add `operationId` to OpenAPI gradually.
2. Use OpenAPI schemas to avoid manually duplicating request/response shapes.
3. Keep `tools.yaml` as the allowlist.
4. Generate descriptor stubs only for selected operations.
5. Review every new tool for risk, authz, audit, and output redaction before enabling it.

## Tool Categories

Initial categories should match the console mental model:

- `workspace`: dashboard, project spaces, applications, members
- `source`: Git accounts, repositories, branches, repository bindings
- `registry`: artifact registries, credentials summary, image records
- `build`: build jobs, build runs, logs, trigger/cancel/retry
- `deployment`: deployment targets, releases, rollback
- `runtime`: runtime clusters, workload resources, events, logs
- `gateway`: gateway routes, domain checks, access URLs
- `billing`: summary, ledger, usage records
- `events`: platform events and notifications
- `system`: platform settings, users, identity providers

MVP should only enable read and low-risk build/deployment helpers:

- list/read projects and applications
- inspect topology, gateway routes, releases, build runs, and events
- read build logs with truncation
- trigger a build with confirmation
- prepare a release plan as dry-run output

## MCP Resources

Use resources for stable context that is not a direct action:

| Resource URI | Content |
| --- | --- |
| `luna://workspace/{projectId}` | Project summary and member role for the current actor |
| `luna://application/{projectId}/{applicationId}` | Application summary, bindings, targets, last release |
| `luna://deployment/{projectId}/{targetId}` | Deployment target runtime config summary |
| `luna://build-run/{projectId}/{runId}` | Build result, image, and compact log tail |
| `luna://platform/capabilities` | Enabled providers, feature switches, and MCP tool availability |

Resources must use the same authz checks as equivalent REST reads.

## MCP Prompts

Prompts should help clients produce consistent, safe actions:

- `luna_deploy_web_project`: guide a user from repository binding to build and release
- `luna_diagnose_failed_build`: inspect failed build run, logs, repository settings, and registry status
- `luna_explain_project_topology`: summarize applications, dependencies, gateway routes, and releases
- `luna_prepare_release`: produce a release checklist and ask for confirmation before mutation

Prompts should be stored as code or generated from the skills under `ai-supports/skills`.

## Rollout Plan

### Phase 0: Declaration and Review

- Keep `ai-supports/mcp/tools.yaml` as the canonical proposal.
- Review every listed tool with backend authz and product risk.
- Add missing `operationId` values to OpenAPI for selected tools.

### Phase 1: Read-Only MCP

- Implement `/api/v1/mcp` with initialize, tools/list, tools/call.
- Enable read-only tools for dashboard, projects, applications, build runs, releases, gateway routes, events, and billing summary.
- Add audit entries for every MCP call.
- Add response redaction and log truncation.

### Phase 2: Confirmed Mutations

- Enable low/medium mutation tools such as build trigger and build cancel.
- Add server-side preflight token and explicit confirmation.
- Add frontend confirmation dialog for the embedded assistant.

### Phase 3: High-Risk Operations

- Add releases, rollbacks, data export authorization, and selected admin tools only after MFA step-up integration.
- Keep runtime exec and terminal tools disabled for external bearer-token clients unless a web-console approval session exists.

### Phase 4: Tool Generation

- Generate tool stubs from OpenAPI operation IDs.
- Keep generated tools disabled until reviewed and added to the allowlist.

## Non-Goals

- Do not make MCP a hidden admin API.
- Do not expose raw kubeconfig, registry credentials, OAuth secrets, session tokens, or recovery codes.
- Do not let the model run arbitrary shell commands by default.
- Do not allow an external MCP client to bypass browser-only MFA step-up for critical operations.
- Do not ship all OpenAPI paths as tools.

