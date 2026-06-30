# Helm Deployment

If you want the platform itself to run in Kubernetes, install it with Helm. The chart creates API, worker, PostgreSQL, and Redis.

## Prepare

You need:

- A Kubernetes or K3s cluster.
- `kubectl` and `helm` configured locally.
- Network access from the cluster to pull DockerHub images.
- A default StorageClass for PostgreSQL and Redis data.

## Install

Run this from the repository root:

```bash
helm install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace
```

This starts:

```text
liteyukistudio/devops-api:nightly
liteyukistudio/devops-worker:nightly
postgres:17-alpine
redis:8-alpine
```

## Open The Console

Forward the API Service:

```bash
kubectl -n liteyuki-devops port-forward svc/liteyuki-devops-api 8088:80
```

Then visit:

```text
http://localhost:8088
```

## Use A Fixed Version

```bash
helm upgrade --install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace \
  --set api.image.tag=v0.1.0-rc.1 \
  --set worker.image.tag=v0.1.0-rc.1
```

## Configure A Public Domain

When exposing the console with Ingress, set `app.publicBaseUrl` to the real browser-facing URL:

```bash
helm upgrade --install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace \
  --set app.publicBaseUrl=https://devops.example.com \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=devops.example.com
```

`app.publicBaseUrl` affects OIDC callbacks, webhook callbacks, and browser origin checks. Do not set it to an internal Service address.

## Use External PostgreSQL Or Redis

The built-in database is good for getting started. For production, you can use managed services:

```yaml
postgresql:
  enabled: false
externalDatabase:
  url: postgres://devops:password@postgres.example.com:5432/devops?sslmode=disable

redis:
  enabled: false
externalRedis:
  addr: redis.example.com:6379
```

Then install:

```bash
helm upgrade --install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace \
  -f values-prod.yaml
```

## Important Values

| Value | Default | Notes |
| --- | --- | --- |
| `app.publicBaseUrl` | `http://localhost:8088` | Public console URL. Required when Ingress is enabled. |
| `app.secretEncryptionKey` | Generated on first install | Encrypts Git, registry, and OIDC secrets. Keep it stable in production. |
| `api.image.tag` / `worker.image.tag` | `nightly` | API and worker image tag. |
| `postgresql.enabled` | `true` | Install built-in PostgreSQL. |
| `redis.enabled` | `true` | Install built-in Redis. |
| `worker.buildEgressMode` | `permissive` | Build Job egress mode. Use `restricted` when you need stronger isolation. |

## Uninstall

```bash
helm uninstall liteyuki-devops -n liteyuki-devops
```

PVCs are retained by default. To remove data:

```bash
kubectl -n liteyuki-devops delete pvc -l app.kubernetes.io/instance=liteyuki-devops
```
