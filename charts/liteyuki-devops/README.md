# Luna DevOps Helm Chart

This chart installs Luna DevOps with API, worker, PostgreSQL, and Redis.

## Install

```bash
helm install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace
```

Open the console:

```bash
kubectl -n liteyuki-devops port-forward svc/liteyuki-devops-api 8088:80
```

Then visit:

```text
http://localhost:8088
```

## Set a public URL

When exposing the console with Ingress, set `app.publicBaseUrl` to the browser-facing URL.

```bash
helm upgrade --install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace \
  --set app.publicBaseUrl=https://devops.example.com \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=devops.example.com
```

## Use an external database or Redis

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

For production, keep `app.secretEncryptionKey` stable. If you do not set it, the chart creates one on first install and reuses the existing Secret during upgrades.
