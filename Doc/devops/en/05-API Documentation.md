# API Documentation

**Project Name**: Liteyuki DevOps  
**Author**: sfkm  
**Date**: 2026-06-22  
**Version**: 1.26.622.52

## Change Log

| Version | Date | Author | Changes |
| --- | --- | --- | --- |
| 1.26.622.52 | 2026-06-22 | sfkm | Created standard engineering documentation and API overview |

---

## 1. Overview

Business APIs are mounted under `/api/v1`. Health check is available at `/healthz`. APIs use JSON by default and support both HttpOnly Cookie sessions and Bearer Access Tokens.

| Group | Prefix | Description |
| --- | --- | --- |
| Public config | `/public`, `/configs` | Public site config and admin config |
| Auth | `/auth` | Login, OIDC, admission policy, bootstrap |
| Users | `/users` | Current user, user management, external identities |
| Git | `/git` | Git providers, accounts, repositories, webhooks |
| Registries | `/registries`, `/container-images` | Registries, credentials, images |
| Build variables | `/build/variable-sets` | Build variables and secret sets |
| Runtime clusters | `/runtime/clusters` | Kubernetes clusters and resources |
| App templates | `/app-templates` | Template list and installation |
| Project spaces | `/projects` | Projects, members, apps, builds, releases, routes |
| Billing | `/billing` | Wallet summary, rates, usage, ledger |
| Access Tokens | `/access-tokens` | Personal API tokens |

## 2. Common Conventions

### 2.1 Paginated Response

```json
{
  "items": [],
  "page": 1,
  "pageSize": 20,
  "sortBy": "createdAt",
  "sortOrder": "desc",
  "total": 0,
  "totalPages": 0
}
```

### 2.2 Error Response

```json
{
  "error": "request failed",
  "code": "resource.not_found",
  "detail": "optional detail"
}
```

### 2.3 Authentication

| Method | Usage |
| --- | --- |
| Cookie Session | Console user access |
| Bearer Access Token | Scripts, external automation, and API triggers |

## 3. API Summary

### 3.1 Authentication and Users

| Method | Path | Description |
| --- | --- | --- |
| GET | `/auth/bootstrap` | Check whether admin bootstrap is required |
| POST | `/auth/bootstrap/admin` | Initialize first admin |
| POST | `/auth/login` | Local login |
| POST | `/auth/logout` | Logout |
| GET | `/auth/providers` | List OIDC providers |
| POST | `/auth/providers` | Create OIDC provider |
| PUT | `/auth/providers/:providerId` | Update OIDC provider |
| GET | `/users/me` | Get current user |
| PUT | `/users/me` | Update current user |
| GET | `/users` | List users |
| POST | `/users` | Create user |

### 3.2 Projects and Applications

| Method | Path | Description |
| --- | --- | --- |
| GET | `/projects` | List project spaces |
| POST | `/projects` | Create project space |
| GET | `/projects/:projectId` | Get project space |
| PUT | `/projects/:projectId` | Update project space |
| DELETE | `/projects/:projectId` | Delete project space |
| GET | `/projects/:projectId/members` | List members |
| POST | `/projects/:projectId/members` | Add member |
| GET | `/projects/:projectId/applications` | List applications |
| POST | `/projects/:projectId/applications` | Create application |
| GET | `/projects/:projectId/applications/:applicationId` | Get application |
| PUT | `/projects/:projectId/applications/:applicationId` | Update application |
| DELETE | `/projects/:projectId/applications/:applicationId` | Delete application |

### 3.3 Builds and Releases

| Method | Path | Description |
| --- | --- | --- |
| GET | `/projects/:projectId/build-runs` | List build runs |
| POST | `/projects/:projectId/build-runs/trigger` | Trigger build |
| GET | `/projects/:projectId/build-runs/:runId` | Get build run |
| POST | `/projects/:projectId/build-runs/:runId/retry` | Retry build |
| POST | `/projects/:projectId/build-runs/:runId/cancel` | Cancel build |
| GET | `/projects/:projectId/build-jobs/:jobId/logs` | Get build logs |
| GET | `/projects/:projectId/build-jobs/:jobId/logs/stream` | Stream build logs |
| GET | `/projects/:projectId/releases` | List releases |
| POST | `/projects/:projectId/releases` | Create release |
| POST | `/projects/:projectId/releases/:releaseId/rollback` | Roll back release |
| GET | `/projects/:projectId/releases/:releaseId/runtime-logs` | Get runtime logs |
| POST | `/projects/:projectId/releases/:releaseId/exec` | Execute runtime command |
| GET | `/projects/:projectId/releases/:releaseId/terminal` | Open runtime terminal |

### 3.4 Runtime Clusters

| Method | Path | Description |
| --- | --- | --- |
| GET | `/runtime/clusters` | List clusters |
| POST | `/runtime/clusters` | Create cluster |
| PUT | `/runtime/clusters/:clusterId` | Update cluster |
| POST | `/runtime/clusters/:clusterId/test` | Test kubeconfig |
| GET | `/runtime/clusters/:clusterId/resources` | List resources |
| GET | `/runtime/clusters/:clusterId/resource-yaml` | Get resource YAML |
| GET | `/runtime/clusters/:clusterId/resource-events` | Get resource events |
| DELETE | `/runtime/clusters/:clusterId/resources` | Delete resource |

### 3.5 Billing

| Method | Path | Description |
| --- | --- | --- |
| GET | `/billing/summary` | Wallet and billing summary |
| GET | `/billing/application-spend` | Application-level spend |
| GET | `/billing/ledger` | Ledger entries |
| GET | `/billing/usage-records` | Usage records |
| GET | `/billing/rate-rules` | Rate rules |
| PUT | `/billing/rate-rules` | Update rate rules |
| POST | `/billing/wallet-transactions` | Create recharge or compensation |
| POST | `/billing/gateway-traffic` | Create gateway traffic usage |

## 4. Exception Scenarios

| Scenario | HTTP Status | Handling |
| --- | --- | --- |
| Not logged in | 401 | Redirect to login |
| Forbidden | 403 | Show friendly forbidden page |
| Not found | 404 | Show empty or error state |
| Form conflict | 409 | Show field or toast error |
| External platform failure | 502/500 | Return stable code without sensitive details |
| Build/deploy failure | 200 + status | Show failed BuildRun/Release with logs |
