# 配置项详解

这一页说明启动平台时常用的配置项。第一次部署可以先使用默认值；需要接入公网域名、生产模式、构建集群或镜像站时，再按表格调整。

## 配置文件

| 文件 | 作用 | 适合场景 |
| --- | --- | --- |
| `.env` | 只放最基础的运行模式，例如 `APP_ENV=development`。 | 本地和 Compose 都会读取。 |
| `.env.worker` | Worker 容器配置，地址按 Compose 网络解释。 | `docker compose up -d` 和 `docker-compose-dev.yaml` 的 worker。 |
| `.env.development` | 宿主机开发配置，地址按宿主机网络解释。 | `go run ./cmd/api` 或 `go run ./cmd/worker`。 |

<div class="hint">
容器里的 `localhost` 指容器自己，不是宿主机。Worker 在 Compose 里连接 PostgreSQL / Redis 时应使用 `postgres`、`redis` 这类服务名。
</div>

## 通用配置

API 和 Worker 都会读取这些配置。

| 配置项 | 默认值 | 说明 | 什么时候修改 |
| --- | --- | --- | --- |
| `APP_ENV` | `development` | 运行模式。`development` 会启用开发体验；生产模式需要完成管理员初始化。 | 部署到真实环境时改为 `production`。 |
| `LOG_LEVEL` | `debug` | 日志级别。 | 生产环境通常改为 `info`。 |
| `SECRET_ENCRYPTION_KEY` | 空 | 用于加密 Git Client Secret、Token、镜像站凭据等敏感值。 | 生产环境必须设置稳定随机值。 |
| `DATABASE_URL` | `postgres://devops:devops@postgres:5432/devops?sslmode=disable` | PostgreSQL 连接串。 | 使用外部数据库，或宿主机开发连接 `localhost:5432` 时修改。 |
| `REDIS_ADDR` | `redis:6379` | Redis 地址。 | 使用外部 Redis，或宿主机开发连接 `localhost:6379` 时修改。 |
| `ENV_FILE` | 空 | 额外覆盖文件路径，会在 `.env` 和环境模式文件之后读取。 | 临时覆盖本机配置时使用，例如 `ENV_FILE=.env.local go run ./cmd/api`。 |

## API 配置项

| 配置项 | Compose 默认值 | 说明 | 什么时候修改 |
| --- | --- | --- | --- |
| `API_ADDR` | `:8080` | API 在容器内监听的地址。 | 通常不用改；只在自定义容器网络或监听端口时修改。 |
| `PUBLIC_BASE_URL` | `http://localhost:8088` | 平台对外访问地址，用于 OIDC 回调、Git Webhook 回调和前端公开链接。 | 有公网域名、反向代理或 HTTPS 时必须改成真实外部地址。 |
| `APP_CORS_ORIGINS` | `http://localhost:8088` | 允许跨域访问 API 的前端 Origin，多个值用逗号分隔。 | 前端和 API 不同域名或端口时修改。 |

## Worker 配置项

| 配置项 | 默认值 | 说明 | 什么时候修改 |
| --- | --- | --- | --- |
| `DEPLOY_ROLLOUT_TIMEOUT_SECONDS` | `600` | 发布后等待 Kubernetes rollout 完成的超时时间。 | 应用启动较慢或镜像拉取较慢时调大。 |
| `CERT_MANAGER_CLUSTER_ISSUER` | `letsencrypt-http01` | 证书申请使用的 cert-manager ClusterIssuer 名称。 | 集群里的 Issuer 名称不同时修改。 |
| `BUILD_EXECUTOR_IMAGE` | `moby/buildkit:v0.24.0-rootless` | 构建 Job 使用的 BuildKit rootless 镜像。 | 需要固定内部镜像源或升级 BuildKit 时修改。 |
| `BUILD_JOB_TIMEOUT_SECONDS` | `5400` | 单次构建 Job 超时时间。 | 大型项目构建时间更长时调大。 |
| `BUILD_JOB_TTL_SECONDS` | `3600` | 构建完成后 Job / Pod 保留时间。 | 想保留更长日志窗口时调大。 |
| `BUILD_CACHE_ENABLED` | `false` | 是否启用构建缓存。 | 需要加速重复构建且镜像站支持缓存时开启。 |
| `BUILD_CACHE_TAG` | `buildcache` | 构建缓存使用的 tag。 | 多环境或多项目需要隔离缓存时修改。 |
| `BUILD_NPM_REGISTRY` | 空 | 构建 Node 项目时注入的 npm registry。 | 需要使用内部 npm 镜像源时设置。 |
| `BUILD_PRIVATE_EGRESS_CIDRS` | 空 | 允许构建 Job 访问的私有网段 CIDR，多个值用逗号分隔。 | 构建需要访问内网 registry 或私有镜像源时设置。 |
| `BUILD_BLOCKED_EGRESS_CIDRS` | 空 | 额外禁止构建 Job 访问的 CIDR，多个值用逗号分隔。 | 需要加强构建网络隔离时设置。 |

## Docker Compose 配置

| 配置项 | 默认值 | 说明 | 什么时候修改 |
| --- | --- | --- | --- |
| `DEVOPS_IMAGE_TAG` | `nightly` | API / Worker 镜像 tag，完整镜像为 `liteyukistudio/devops-api:${DEVOPS_IMAGE_TAG}` 和 `liteyukistudio/devops-worker:${DEVOPS_IMAGE_TAG}`。 | 需要固定版本、回滚版本或测试指定 tag 时修改。 |
| `8088:8080` | `8088` 暴露到宿主机 | 宿主机访问控制台的端口映射。 | `8088` 被占用，或希望使用其他入口端口时修改 `docker-compose.yaml`。 |
| `POSTGRES_DB` | `devops` | 内置 PostgreSQL 数据库名。 | 使用外部数据库时不需要改 Compose 内置服务，改 `DATABASE_URL` 即可。 |
| `POSTGRES_USER` | `devops` | 内置 PostgreSQL 用户名。 | 只在继续使用内置 PostgreSQL 且要改账号时修改。 |
| `POSTGRES_PASSWORD` | `devops` | 内置 PostgreSQL 密码。 | 生产环境建议使用外部数据库或至少改成强密码。 |

## 最小生产建议

生产环境至少确认这些项：

| 配置项 | 建议 |
| --- | --- |
| `APP_ENV` | 设置为 `production`。 |
| `SECRET_ENCRYPTION_KEY` | 设置为稳定随机值，升级或重启时不要变化。 |
| `PUBLIC_BASE_URL` | 设置为真实 HTTPS 外部地址。 |
| `APP_CORS_ORIGINS` | 只保留可信前端 Origin。 |
| `DATABASE_URL` | 使用可靠 PostgreSQL，并配置备份。 |
| `DEVOPS_IMAGE_TAG` | 使用明确版本 tag，不建议长期依赖 `nightly`。 |
