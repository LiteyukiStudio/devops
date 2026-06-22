# Deployment & User Manual

**Project Name**: Liteyuki DevOps  
**Author**: sfkm  
**Date**: 2026-06-22  
**Version**: 1.26.622.52

## Change Log

| Version | Date | Author | Changes |
| --- | --- | --- | --- |
| 1.26.622.52 | 2026-06-22 | sfkm | Created standard engineering documentation and deployment/user manual |

---

# Part 1: Deployment Manual

## 1. Environment Requirements

| Item | Requirement |
| --- | --- |
| OS | Linux, macOS, or Windows Docker environment |
| Container runtime | Docker + Docker Compose |
| Database | PostgreSQL 17 or compatible |
| Cache/queue | Redis 8 or compatible |
| Runtime cluster | Kubernetes/K3s with kubeconfig |
| Certificates | cert-manager recommended for HTTPS |
| Registry | DockerHub, Harbor, Gitea Registry, or OCI-compatible registry |

## 2. Quick Deployment

Start the full platform:

```bash
docker compose up -d
```

Default URL:

```text
http://localhost:8088
```

Default images:

```text
liteyukistudio/devops-api:${DEVOPS_IMAGE_TAG:-nightly}
liteyukistudio/devops-worker:${DEVOPS_IMAGE_TAG:-nightly}
```

Use a specific version:

```bash
DEVOPS_IMAGE_TAG=v0.1.0 docker compose up -d
```

## 3. Key Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `APP_ENV` | `development` | Runtime mode; use `production` for production |
| `API_ADDR` | `:8080` | API listen address |
| `DATABASE_URL` | Compose PostgreSQL | Database connection string |
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `PUBLIC_BASE_URL` | `http://localhost:8088` | External base URL |
| `APP_CORS_ORIGINS` | `http://localhost:8088` | Allowed origins |
| `SECRET_ENCRYPTION_KEY` | empty | Must be stable random value in production |
| `BUILD_EXECUTOR_IMAGE` | `moby/buildkit:v0.24.0-rootless` | Build executor image |
| `BUILD_EGRESS_MODE` | `permissive` | Build egress mode |
| `BUILD_PRIVATE_EGRESS_CIDRS` | empty | Allowed private CIDRs for build |
| `BUILD_PRIVATE_EGRESS_PORTS` | `443` | Allowed private ports |
| `CERT_MANAGER_CLUSTER_ISSUER` | `letsencrypt-http01` | cert-manager issuer |

## 4. Local Development

Start development dependencies:

```bash
docker compose -f docker-compose-dev.yaml up -d --build
```

Start API:

```bash
go run ./cmd/api
```

Start worker:

```bash
go run ./cmd/worker
```

Start frontend:

```bash
pnpm --dir web install
pnpm --dir web dev
```

Start docs site:

```bash
pnpm --dir docs install
pnpm --dir docs dev --port 5274
```

## 5. Production Initialization

1. Start the platform.
2. Open `PUBLIC_BASE_URL`.
3. Create the first administrator.
4. Configure site title, logo, and favicon.
5. Configure OIDC providers and admission policy.
6. Configure runtime clusters.
7. Configure registries and credentials.
8. Create project spaces and applications.

# Part 2: User Manual

## 1. Deploy a Web Project

1. Create a project space.
2. Create an application.
3. Connect GitHub or Gitea account.
4. Bind repository in the application.
5. Create deployment target with Dockerfile, build context, registry, port, and resources.
6. Trigger build.
7. Create Release after build succeeds, or enable auto deploy.
8. Create gateway route with generated or custom domain.
9. Open the route and verify the service.

## 2. Install an App Marketplace Template

1. Open App Marketplace.
2. Select a template.
3. Confirm image, website, and official repository.
4. Select project space and runtime cluster.
5. Fill in template parameters.
6. Keep "Deploy after install" enabled, or deploy manually later.

## 3. Observe Build and Deployment State

- Build page shows BuildRun, BuildJob, stage progress, and logs.
- Deploy page shows Release, runtime logs, terminal, and rollback.
- Cluster page shows Kubernetes resources, YAML, and events.
- Billing page shows wallet, usage records, ledger, and application spend.

## 4. Troubleshooting

| Problem | Possible Cause | Solution |
| --- | --- | --- |
| Base image pull fails | DNS or egress policy blocks traffic | Check NetworkPolicy, DNS 53, and registry allowlist |
| OIDC login fails | Callback URL or admission policy is wrong | Check redirect URI, email, and group settings |
| Deployment stays pending | Insufficient cluster resources or image pull failure | Check Release logs, Pod events, and registry credentials |
| HTTPS certificate fails | DNS or issuer unavailable | Check domain, Ingress, and cert-manager issuer |
| Secret update has no effect | Empty edit means unchanged | Enter a new value and save |
| Billing is not generated | Resource not running or billing runner not executed | Check worker, usage records, and rate rules |
