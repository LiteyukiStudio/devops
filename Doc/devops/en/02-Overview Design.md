# Overview Design

**Project Name**: Liteyuki DevOps  
**Author**: sfkm  
**Date**: 2026-06-22  
**Version**: 1.26.622.52

## Change Log

| Version | Date | Author | Changes |
| --- | --- | --- | --- |
| 1.26.622.52 | 2026-06-22 | sfkm | Created standard engineering documentation and overview design |

---

## 1. System Architecture

Liteyuki DevOps uses a modular monolith with multiple runtime processes:

- `cmd/api`: HTTP API, authentication, CRUD, webhook handling, task enqueueing, and embedded SPA hosting.
- `cmd/worker`: asynchronous build, deploy, certificate, cleanup, and billing tasks.
- PostgreSQL: business data, state, billing ledger, logs, and configuration.
- Redis + Asynq: asynchronous task queue.
- Kubernetes/K3s: build execution and application runtime.
- Rspress: documentation site.

```text
Browser
  -> API / embedded SPA
     -> PostgreSQL
     -> Redis / Asynq
     -> Git provider APIs
     -> Registry provider APIs
     -> Kubernetes provider APIs

Worker
  -> Redis / Asynq
  -> PostgreSQL
  -> Kubernetes Build Job / Deployment / Ingress
  -> Registry push / tag query
```

## 2. Module Breakdown

| Module | Directory/Page | Responsibility |
| --- | --- | --- |
| Auth and Users | `internal/api`, settings pages | Login, OIDC, users, Access Tokens |
| Project Spaces | `internal/model/project.go`, project pages | Projects, members, pins, hooks, runtime config sets |
| Applications | `application.go`, application pages | Application metadata and workspace |
| Repository | `internal/provider/git` | Git providers, accounts, repository binding, webhooks |
| Build | `internal/worker/build_*`, `internal/builder` | BuildRun, BuildJob, BuildKit execution, logs |
| Registry | `internal/provider/registry` | Registries, credentials, images, tag queries |
| Deployment | `internal/provider/kubernetes`, `deployment.go` | Runtime clusters, deployment targets, releases |
| Gateway | `gateway.go`, `gateway_runner.go` | GatewayRoute, Ingress, certificate flow |
| Billing | `internal/billing`, billing page | Wallet, rates, usage, ledger |
| App Marketplace | `internal/appstore`, app template page | Built-in templates and installation |
| Documentation | `docs/` | Rspress docs site |

## 3. Core Flows

### 3.1 Repository Build

1. User connects a Git account and binds a repository.
2. User creates a deployment target with Dockerfile, context, registry, and target tag.
3. Manual action, webhook, or Access Token creates a BuildRun.
4. API creates a BuildJob and enqueues an Asynq task.
5. Worker creates a Kubernetes Job running BuildKit rootless.
6. Worker pushes the image, records digest, and updates BuildRun/BuildJob.
7. If auto deploy is enabled, Worker creates a Release.

### 3.2 Image Deployment

1. User selects a deployment target and image.
2. API creates a Release.
3. Worker renders Kubernetes specs.
4. Kubernetes provider applies Deployment, Service, ConfigMap, Secret, PVC, and Ingress.
5. Worker observes rollout and updates Release status.

### 3.3 Billing

1. Worker samples runtime and build usage.
2. Billing service generates BillingUsageRecord.
3. Billing service creates BillingLedgerEntry for the billed user.
4. UserWallet balance is updated.
5. Billing UI reads summary, usage records, and ledger entries.

## 4. Dependencies

| Dependency | Version | Purpose |
| --- | --- | --- |
| Go | 1.25.4 | Backend runtime |
| Gin | 1.12.0 | HTTP API |
| GORM | 1.31.1 | ORM |
| golang-migrate | 4.19.1 | SQL migrations |
| Redis client | 9.14.1 | Redis access |
| Asynq | 0.26.0 | Task queue |
| client-go | 0.34.2 | Kubernetes API |
| React | 19.2.6 | Frontend UI |
| Vite | 8.0.12 | Frontend build |
| Tailwind CSS | 4.3.0 | Styling |

## 5. Design Constraints

- Long-running build and deploy work must run in worker tasks.
- Frontend must not orchestrate external platform APIs directly.
- Secrets must be stored as encrypted values or references.
- API and worker share the same database schema and migrations.
- User-facing docs and engineering docs are separated.
