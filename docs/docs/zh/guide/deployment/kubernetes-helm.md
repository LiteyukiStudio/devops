# Kubernetes (Helm) 部署

如果平台本身也要跑在 Kubernetes 或 K3s 里，推荐使用 Helm。Chart 会一起创建 API、worker、PostgreSQL 和 Redis。

## 准备

你需要：

- 一个可用的 Kubernetes 或 K3s 集群。
- 本机已经配置好 `kubectl` 和 `helm`。
- 集群能拉取 DockerHub 镜像。
- 默认 StorageClass 可用，用来保存 PostgreSQL 和 Redis 数据。

## 安装

在仓库根目录执行：

```bash
helm install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace
```

这会启动：

```text
liteyukistudio/devops-api:nightly
liteyukistudio/devops-worker:nightly
postgres:17-alpine
redis:8-alpine
```

## 打开控制台

先把 API Service 转发到本机：

```bash
kubectl -n liteyuki-devops port-forward svc/liteyuki-devops-api 8088:80
```

然后访问：

```text
http://localhost:8088
```

## 使用固定版本

```bash
helm upgrade --install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace \
  --set api.image.tag=v0.1.0-rc.1 \
  --set worker.image.tag=v0.1.0-rc.1
```

## 配置公网域名

如果通过 Ingress 暴露控制台，把 `app.publicBaseUrl` 改成用户真实访问的地址：

```bash
helm upgrade --install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace \
  --set app.publicBaseUrl=https://devops.example.com \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=devops.example.com
```

`app.publicBaseUrl` 会影响 OIDC 回调、Webhook 回调和浏览器跨域校验，不要写成集群内 Service 地址。

## 使用外部 PostgreSQL 或 Redis

内置数据库适合先跑起来。生产环境如果已有托管数据库，可以关闭内置组件：

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

然后安装：

```bash
helm upgrade --install liteyuki-devops ./charts/liteyuki-devops \
  --namespace liteyuki-devops \
  --create-namespace \
  -f values-prod.yaml
```

## 常用配置

| 配置项 | 默认值 | 说明 |
| --- | --- | --- |
| `app.publicBaseUrl` | `http://localhost:8088` | 控制台对外访问地址。启用 Ingress 后必须改成公网地址。 |
| `app.secretEncryptionKey` | 首次安装自动生成 | 用于加密 Git、镜像站和 OIDC 密钥。生产环境要保持稳定。 |
| `api.image.tag` / `worker.image.tag` | `nightly` | API 和 worker 镜像版本。 |
| `postgresql.enabled` | `true` | 是否安装内置 PostgreSQL。 |
| `redis.enabled` | `true` | 是否安装内置 Redis。 |
| `worker.buildEgressMode` | `permissive` | 构建 Job 出站网络模式。需要强隔离时改为 `restricted`。 |

## 卸载

```bash
helm uninstall liteyuki-devops -n liteyuki-devops
```

PVC 默认不会自动删除。如果要清理数据：

```bash
kubectl -n liteyuki-devops delete pvc -l app.kubernetes.io/instance=liteyuki-devops
```
