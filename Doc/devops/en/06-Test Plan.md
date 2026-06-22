# Test Plan

**Project Name**: Liteyuki DevOps  
**Author**: sfkm  
**Date**: 2026-06-22  
**Version**: 1.26.622.52

## Change Log

| Version | Date | Author | Changes |
| --- | --- | --- | --- |
| 1.26.622.52 | 2026-06-22 | sfkm | Created standard engineering documentation and test plan |

---

## 1. Test Scope

| Scope | Coverage |
| --- | --- |
| Backend unit tests | Configuration, permissions, builds, Kubernetes provider, billing, migrations |
| API integration tests | Auth, projects, apps, builds, deployments, billing, pagination |
| Worker tests | BuildRun, deploy, gateway, cleanup, billing runner |
| Frontend checks | TypeScript build, ESLint, i18n keys |
| Browser smoke tests | Login, projects, apps, deployment targets, builds, app marketplace |
| Docs checks | Rspress build, multilingual pages, navigation |

## 2. Unit Test Cases

### 2.1 Configuration

| Case ID | Name | Preconditions | Steps | Expected Result |
| --- | --- | --- | --- | --- |
| UT-CONFIG-001 | Defaults | No env vars | Call `config.Load()` | Default API, DB, Redis, and build config |
| UT-CONFIG-002 | Build egress mode | `BUILD_EGRESS_MODE=restricted` | Call `config.Load()` | Mode is restricted |
| UT-CONFIG-003 | Private egress ports | Set port list | Call `config.Load()` | Invalid ports ignored, valid ports kept |

### 2.2 Kubernetes Provider

| Case ID | Name | Preconditions | Steps | Expected Result |
| --- | --- | --- | --- | --- |
| UT-K8S-001 | Namespace naming | Given projectId | Call namespace helper | Stable namespace |
| UT-K8S-002 | Deployment rendering | Given DeploymentTarget | Render specs | Name, labels, ports, resources correct |
| UT-K8S-003 | NetworkPolicy rendering | Restricted config | Render egress policy | DNS and allowlist match expectations |
| UT-K8S-004 | Certificate resources | TLS GatewayRoute | Render ingress/cert config | Issuer and host correct |

### 2.3 Build and Deploy

| Case ID | Name | Preconditions | Steps | Expected Result |
| --- | --- | --- | --- | --- |
| UT-BUILD-001 | BuildRun snapshot | DeploymentTarget exists | Trigger build | BuildRun copies target build settings |
| UT-BUILD-002 | Log append | BuildJob exists | Append log chunks | BuildLog readable by offset |
| UT-DEPLOY-001 | Release success | Rollout succeeds | Run deploy runner | Release becomes succeeded |
| UT-DEPLOY-002 | Rollout timeout | Deployment not ready | Run deploy runner | Release fails with log |

### 2.4 Billing

| Case ID | Name | Preconditions | Steps | Expected Result |
| --- | --- | --- | --- | --- |
| UT-BILL-001 | Usage settlement | Enabled rate | Create usage | Ledger is generated and wallet deducted |
| UT-BILL-002 | Idempotent settlement | Same idempotency key | Repeat settlement | No duplicate deduction |
| UT-BILL-003 | Owner transfer | Billing owner changes | Generate usage before and after | Usage is billed to different users |

## 3. Integration Test Plan

### 3.1 API Integration

1. Start PostgreSQL and Redis.
2. Start API.
3. Bootstrap admin.
4. Create project space, application, and deployment target.
5. Configure Git provider/account mock or test credentials.
6. Trigger build and assert BuildRun/BuildJob states.
7. Create Release and assert state transitions.
8. Query billing and logs APIs.

### 3.2 Worker Integration

1. Use kind, minikube, or K3s as runtime cluster.
2. Create test namespace.
3. Trigger build with a minimal Dockerfile.
4. Verify BuildKit Job, logs, and image push.
5. Verify auto deploy and Service/Ingress generation.

## 4. Browser Acceptance

| Case ID | Page | Steps | Expected Result |
| --- | --- | --- | --- |
| E2E-001 | Login | Bootstrap admin and login | Dashboard opens |
| E2E-002 | Project | Create project and member | List and detail are correct |
| E2E-003 | Application | Create app and bind repository | App detail shows repository and deployment tabs |
| E2E-004 | Deployment target | Create target | Validation works and secrets are not echoed |
| E2E-005 | Build | Trigger build and open logs | Status and SSE logs update |
| E2E-006 | App marketplace | Install template | App and deployment target are created |
| E2E-007 | Theme/language | Switch themes and languages | UI updates immediately |

## 5. Boundary Tests

- Project slug and application slug conflicts.
- Deployment target display name contains Chinese text or long text.
- Dockerfile path does not exist.
- Image tag reuses `latest`.
- Kubernetes API is unavailable.
- DNS, registry, or package manager network is unavailable.
- Redis is unavailable.
- User wallet balance is insufficient.
- Secret update is left empty.
- OIDC callback state expires.

## 6. Common Commands

```bash
go test ./...
pnpm --dir web lint
pnpm --dir web build
pnpm --dir docs build
docker compose -f docker-compose-dev.yaml up -d
```

## 7. Test Data

| Data | Purpose |
| --- | --- |
| Admin user | Bootstrap and global settings |
| Regular user | Permission and member tests |
| GitHub/Gitea test repo | Repository binding, webhook, build |
| Harbor/DockerHub credentials | Image push and tag queries |
| minikube/K3s cluster | Build Job and deployment resources |
| Simple web image | Deployment and gateway acceptance |
