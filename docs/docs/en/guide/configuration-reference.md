# Configuration Reference

This page explains the configuration used when starting the platform. For a first deployment, keep the defaults. Adjust these values when you connect a public domain, production mode, build cluster, or registry.

## Configuration files

| File | Purpose | Best for |
| --- | --- | --- |
| `.env` | Keeps only the base runtime mode, such as `APP_ENV=development`. | Read by both local processes and Compose. |
| `.env.worker` | Worker container settings, interpreted from inside the Compose network. | Worker in `docker compose up -d` and `docker-compose-dev.yaml`. |
| `.env.development` | Host development settings, interpreted from the host network. | `go run ./cmd/api` or `go run ./cmd/worker`. |

<div class="hint">
Inside a container, `localhost` means the container itself. When Worker runs in Compose, use service names such as `postgres` and `redis`.
</div>

## Shared settings

Both API and Worker read these settings.

| Key | Default | Meaning | Change when |
| --- | --- | --- | --- |
| `APP_ENV` | `development` | Runtime mode. `development` enables local convenience behavior; production mode requires admin bootstrap. | Deploying to a real environment. |
| `LOG_LEVEL` | `debug` | Log level. | Production usually uses `info`. |
| `SECRET_ENCRYPTION_KEY` | Empty | Encrypts Git client secrets, tokens, registry credentials, and other sensitive values. | Required in production; keep it stable across restarts. |
| `DATABASE_URL` | `postgres://devops:devops@postgres:5432/devops?sslmode=disable` | PostgreSQL connection string. | Using an external database, or host development with `localhost:5432`. |
| `REDIS_ADDR` | `redis:6379` | Redis address. | Using an external Redis, or host development with `localhost:6379`. |
| `ENV_FILE` | Empty | Extra override file read after `.env` and the mode-specific file. | Temporary local overrides, for example `ENV_FILE=.env.local go run ./cmd/api`. |

## API settings

| Key | Compose default | Meaning | Change when |
| --- | --- | --- | --- |
| `API_ADDR` | `:8080` | API listen address inside the container. | Usually keep it; change only for custom container networking or ports. |
| `PUBLIC_BASE_URL` | `http://localhost:8088` | Public platform URL used for OIDC callbacks, Git webhook callbacks, and public frontend links. | You have a public domain, reverse proxy, or HTTPS endpoint. |
| `APP_CORS_ORIGINS` | `http://localhost:8088` | Frontend origins allowed to call the API. Separate multiple origins with commas. | Frontend and API use different domains or ports. |

## Worker settings

| Key | Default | Meaning | Change when |
| --- | --- | --- | --- |
| `DEPLOY_ROLLOUT_TIMEOUT_SECONDS` | `600` | Timeout while waiting for Kubernetes rollout after release. | Applications start slowly or image pulls take longer. |
| `CERT_MANAGER_CLUSTER_ISSUER` | `letsencrypt-http01` | cert-manager ClusterIssuer used for certificate requests. | Your cluster uses a different Issuer name. |
| `BUILD_EXECUTOR_IMAGE` | `moby/buildkit:v0.24.0-rootless` | BuildKit rootless image used by build Jobs. | You need an internal image mirror or a different BuildKit version. |
| `BUILD_JOB_TIMEOUT_SECONDS` | `5400` | Timeout for one build Job. | Large projects need longer build time. |
| `BUILD_JOB_TTL_SECONDS` | `3600` | How long completed build Jobs / Pods stay around. | You want a longer log inspection window. |
| `BUILD_CACHE_ENABLED` | `false` | Enables build cache. | You want faster repeated builds and your registry supports the cache flow. |
| `BUILD_CACHE_TAG` | `buildcache` | Tag used for build cache. | You need to isolate cache by environment or project. |
| `BUILD_NPM_REGISTRY` | Empty | npm registry injected into Node builds. | You use an internal npm mirror. |
| `BUILD_PRIVATE_EGRESS_CIDRS` | Empty | Private CIDRs build Jobs may access. Separate multiple values with commas. | Builds need access to internal registries or mirrors. |
| `BUILD_BLOCKED_EGRESS_CIDRS` | Empty | Extra CIDRs build Jobs must not access. Separate multiple values with commas. | You need stricter build network isolation. |

## Docker Compose settings

| Key | Default | Meaning | Change when |
| --- | --- | --- | --- |
| `DEVOPS_IMAGE_TAG` | `nightly` | API / Worker image tag. Images are `liteyukistudio/devops-api:${DEVOPS_IMAGE_TAG}` and `liteyukistudio/devops-worker:${DEVOPS_IMAGE_TAG}`. | Pinning a release, rolling back, or testing a specific tag. |
| `8088:8080` | Host port `8088` | Host port used to open the console. | Port `8088` is occupied, or you want another entry port. |
| `POSTGRES_DB` | `devops` | Built-in PostgreSQL database name. | Usually keep it; for external DBs, change `DATABASE_URL` instead. |
| `POSTGRES_USER` | `devops` | Built-in PostgreSQL username. | Only when keeping the built-in PostgreSQL and changing credentials. |
| `POSTGRES_PASSWORD` | `devops` | Built-in PostgreSQL password. | For production, prefer an external database or change this to a strong password. |

## Minimal production checklist

| Key | Recommendation |
| --- | --- |
| `APP_ENV` | Set to `production`. |
| `SECRET_ENCRYPTION_KEY` | Use a stable random value; do not change it during restarts or upgrades. |
| `PUBLIC_BASE_URL` | Use the real external HTTPS URL. |
| `APP_CORS_ORIGINS` | Keep only trusted frontend origins. |
| `DATABASE_URL` | Use a reliable PostgreSQL instance with backups. |
| `DEVOPS_IMAGE_TAG` | Use an explicit release tag instead of relying on `nightly` long term. |
