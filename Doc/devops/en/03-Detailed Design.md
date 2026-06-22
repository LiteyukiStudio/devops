# Detailed Design

**Project Name**: Liteyuki DevOps  
**Author**: sfkm  
**Date**: 2026-06-22  
**Version**: 1.26.622.52

## Change Log

| Version | Date | Author | Changes |
| --- | --- | --- | --- |
| 1.26.622.52 | 2026-06-22 | sfkm | Created standard engineering documentation and detailed design |

---

## 1. API Startup and Routing

1. `cmd/api/main.go` loads configuration.
2. Database connection is initialized.
3. SQL migrations are executed.
4. Gin router is created.
5. Middleware is registered: logger, recovery, error response, security headers, CORS, CSRF Origin Guard.
6. `/api/v1` routes and `/healthz` are registered.
7. Embedded SPA is served in `embed_web` builds.

Handlers parse parameters, resolve the current user, run permission checks, call business logic, and map responses. Frontend-localizable messages are returned as stable codes.

## 2. Authentication Design

### 2.1 Local Login

1. User submits username and password.
2. Backend loads the user and validates password hash.
3. Backend creates a session.
4. Session cookie is HttpOnly. Frontend calls `/users/me` to read user state.

### 2.2 OIDC Login

1. Frontend opens `/auth/oidc/:providerId/start`.
2. Backend generates state and redirects to the provider.
3. Callback validates state, token, and userinfo.
4. Admission policy checks email, groups, invite state, and domains.
5. Backend binds to an existing user or rejects login.

### 2.3 Access Token

- Plain token is shown only once.
- Database stores token hash, scopes, expiration, and revocation time.
- API matches method/path against scopes. Unknown APIs are rejected by default.

## 3. Build Design

### 3.1 BuildRun Creation

1. User triggers a build for a deployment target.
2. API copies deployment target build settings into BuildRun.
3. API creates BuildRun with `queued` status.
4. API creates BuildJob with `queued` status.
5. API enqueues the build task.

### 3.2 Worker Execution

1. Worker loads BuildJob and BuildRun.
2. Worker validates project, application, deployment target, repository binding, and registry credentials.
3. Worker renders variables and secrets.
4. Worker renders Kubernetes Job spec.
5. Worker creates NetworkPolicy according to build egress configuration.
6. BuildKit rootless builds and pushes the image.
7. Logs are appended to BuildLog.
8. Success records image ref, digest, resource usage, and ContainerImage.
9. Failure records message and logs.

### 3.3 Auto Deploy

If the deployment target enables auto deploy:

1. Worker checks branch and tag patterns.
2. Worker creates a Release referencing BuildRun and image ref.
3. Worker enqueues deploy task.

## 4. Deployment Design

Release represents a deploy or rollback operation. Kubernetes provider renders and applies:

- Namespace
- Deployment
- Service
- ConfigMap
- Secret
- PVC
- Ingress

Runtime resource names are derived from internal IDs, not user-provided display names. This avoids invalid Kubernetes names caused by localized names, spaces, or long text.

Worker waits for rollout until timeout. Success sets Release to `succeeded`; failure writes ReleaseLog and sets status to `failed`.

## 5. App Marketplace Design

Templates are embedded from `internal/appstore/templates.json`. Each template contains:

- Basic metadata: id, slug, name, description, category, icon.
- Source metadata: website, repository, image, version.
- Runtime defaults: port, replicas, CPU, memory.
- Injection data: env, secret env, config files, secret files.
- User inputs: values.

Install flow:

1. Read template and render user input.
2. Create Application.
3. Create DeploymentTarget.
4. Persist secret references and config files.
5. Create AppTemplateInstallation.
6. Optionally create first Release.

## 6. Billing Design

- `UserWallet` stores balance.
- `BillingRateRule` stores meter prices.
- `BillingUsageRecord` stores usage by project, application, resource, and meter.
- `BillingLedgerEntry` stores balance changes and idempotency key.

Project spaces hold `billingOwnerUserId`. Usage before owner transfer is billed to the previous owner; usage after transfer is billed to the new owner. Recharge and compensation are user-level wallet events.

## 7. Frontend Design

Main pages include Dashboard, Projects, Applications, AppTemplates, CodeRepositories, Registries, Clusters, Billing, and Settings.

Frontend state design:

- TanStack Query handles server state, caching, refresh, and invalidation.
- API client centralizes errors and JSON handling.
- i18next manages Chinese and English text.
- Shared components such as `ContentTabs`, `DataList`, and `StatusBadge` keep management pages consistent.

Interaction rules:

- Management lists should use DataList.
- Status values should use semantic badges.
- Complex form fields should include help hints.
- Secret fields never show existing values; empty means unchanged.
