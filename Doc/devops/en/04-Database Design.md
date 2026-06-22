# Database Design

**Project Name**: Liteyuki DevOps  
**Author**: sfkm  
**Date**: 2026-06-22  
**Version**: 1.26.622.52

## Change Log

| Version | Date | Author | Changes |
| --- | --- | --- | --- |
| 1.26.622.52 | 2026-06-22 | sfkm | Created standard engineering documentation and database design |

---

## 1. Overview

The system uses PostgreSQL as the primary database, GORM as ORM, and golang-migrate for SQL migrations. API and worker share the same schema. Redis is used for queues, cache, and rate limiting, but not as the source of truth for business state.

Migration files are stored in `migrations/`.

## 2. Core Tables

### 2.1 Users and Auth

| Model | Purpose | Key Fields |
| --- | --- | --- |
| User | Platform user | id, username, email, role, language, avatar |
| UserSession | Login session | userId, tokenHash, expiresAt |
| UserRememberToken | Login resume token | userId, tokenHash, expiresAt |
| AccessToken | API token | userId, tokenHash, scopes, expiresAt, revokedAt |
| AuthProvider | OIDC provider | type, issuer, clientId, clientSecretRef |
| ExternalIdentity | External identity binding | userId, providerId, subject, email |
| AuthAdmissionPolicy | Login admission policy | allowedDomains, allowedEmails, requiredGroups |

### 2.2 Projects and Applications

| Model | Purpose | Key Fields |
| --- | --- | --- |
| Project | Project space | slug, name, namespaceStrategy, billingOwnerUserId, deleteStatus |
| ProjectMember | Membership | projectId, userId, role, lastUsedAt, useCount |
| ProjectPin | User pinned project | userId, projectId, pinnedAt |
| Application | Application | projectId, slug, name, icon, deleteStatus |
| AppConfig | Site configuration | key, value |
| ProjectHookConfig | Project hook configuration | projectId, script, phase, failurePolicy |
| HookRun | Hook execution | buildRunId, releaseId, status, exitCode |
| HookRunLog | Hook log | hookRunId, content |

### 2.3 Build and Deployment

| Model | Purpose | Key Fields |
| --- | --- | --- |
| RuntimeCluster | Kubernetes cluster | name, type, scope, kubeconfigRef, isDefault, status |
| Environment | Environment | projectId, slug, clusterId, namespace |
| DeploymentTarget | Deployment target | applicationId, clusterId, sourceType, imageRef, runtime config |
| BuildRun | Build business run | deploymentTargetId, status, sourceCommit, imageRef, creditCost |
| BuildJob | Build executor task | buildRunId, status, leaseToken, executorName, attempts |
| BuildLog | Build log | buildRunId, buildJobId, content |
| Release | Release record | deploymentTargetId, buildRunId, imageRef, status, revision |
| ReleaseLog | Release log | releaseId, content |

### 2.4 Repository and Registry

| Model | Purpose | Key Fields |
| --- | --- | --- |
| GitProvider | Git platform config | type, baseURL, clientId, clientSecretRef |
| GitAccount | Git account auth | providerId, userId, accessTokenRef, status |
| RepositoryBinding | Repository binding | projectId, applicationId, providerId, owner, repo |
| ArtifactRegistry | Image registry | provider, endpoint, scope, credentialRef, isDefault |
| RegistryCredential | Registry credential | registryId, username, passwordRef, tokenRef |
| ContainerImage | Image record | registryId, repository, tag, digest, buildRunId |

### 2.5 Gateway and Billing

| Model | Purpose | Key Fields |
| --- | --- | --- |
| GatewayRoute | Public route | projectId, applicationId, deploymentTargetId, domain, tlsEnabled |
| UserWallet | User wallet | userId, balanceCredits |
| BillingRateRule | Rate rule | meter, unit, creditsPerUnit, enabled |
| BillingUsageRecord | Usage record | billedUserId, projectId, applicationId, resourceId, meter |
| BillingLedgerEntry | Ledger entry | userId, type, amountCredits, balanceAfterCredits, idempotencyKey |
| AuditLog | Audit log | actorId, action, resourceType, resourceId, metadata |
| SecretValue | Secret value | key, encryptedValue |
| WorkerTaskEvent | Worker event | taskId, type, message |

## 3. Relationships

```text
User 1 -> N ProjectMember N -> 1 Project
Project 1 -> N Application
Application 1 -> N DeploymentTarget
DeploymentTarget 1 -> N BuildRun
BuildRun 1 -> N BuildJob
BuildRun 1 -> N Release
DeploymentTarget 1 -> N Release
Project 1 -> N GatewayRoute
Project 1 -> N BillingUsageRecord
User 1 -> 1 UserWallet
User 1 -> N BillingLedgerEntry
```

## 4. Constraints and Indexes

| Model | Constraint |
| --- | --- |
| Project | `slug` is unique among non-deleted records |
| Application | `projectId + slug` is unique among non-deleted records |
| ProjectPin | `userId + projectId` is unique |
| BillingRateRule | `meter` is unique |
| BillingUsageRecord | `resourceType + resourceId + meter` is unique |
| BuildJob | Indexed by buildRunId and status |

## 5. Deletion Strategy

- Project, Application, DeploymentTarget, and RuntimeCluster use soft delete.
- `deleteStatus` supports asynchronous deletion display.
- Billing records are retained after resource deletion.
- Secret references are cleaned by resource cleanup tasks, but historical logs never expose plaintext.

## 6. Migration Strategy

- API and worker should execute migrations before serving.
- Migration files are append-only after publication.
- Data backfills should use dedicated migrations.
- Text-encoded JSON fields can later migrate to jsonb when query requirements become stronger.
