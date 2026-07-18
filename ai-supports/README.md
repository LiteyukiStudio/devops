# Luna DevOps AI Supports

This directory contains the AI-facing integration plan for Luna DevOps.

The first target is MCP support. The implementation should expose a curated set of Luna DevOps backend capabilities as MCP tools, while keeping the existing REST API, JWT authentication, RBAC, audit logs, MFA step-up, and secret redaction as the security boundary.

## Directory Layout

```text
ai-supports/
  mcp/
    design.md           MCP integration architecture and rollout plan
    security.md         Risk model, confirmation flow, and audit requirements
    tools.yaml          Curated MCP tool declaration
  skills/
    luna-devops-operator/
      SKILL.md          General platform operation skill
    luna-devops-deployment/
      SKILL.md          Build and deployment workflow skill
    luna-devops-debugging/
      SKILL.md          Diagnostics and troubleshooting skill
```

## Design Principles

- Expose a small, useful MCP surface first. Do not auto-publish every REST endpoint.
- Reuse existing backend authorization. MCP tools must never bypass Luna DevOps RBAC.
- Treat destructive, billing, secret, runtime exec, terminal, and data export operations as high risk.
- Prefer preflight and confirmation over direct mutation.
- Keep tool outputs short, structured, and secret-safe.
- Add tools by declaration first, then implement adapters behind the declaration.

