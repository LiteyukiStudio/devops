# AI 开发规范

本文档给 AI 编码代理使用，定义本项目的技术栈、目录、命令和实现边界。编码前必须先阅读：

0. `README.md`
1. `docs/01-产品与一体化方案.md`
2. `docs/02-项目技术栈要求.md`
3. `TODO.md`

## 1. 总体原则

- 先读现有代码和文档，再写代码。
- 不引入未讨论的新框架。
- 优先实现 MVP 主链路。
- 不把 Gitea/GitHub Actions 作为强制主路径。
- 平台构建主路径是 Kubernetes Job + BuildKit rootless。
- 部署必须由平台执行和记录。
- Secret、Token、Registry Credential 不允许明文落业务表。

## 2. TODO 与验收闭环

- 开始实现前必须查看 `TODO.md`，确认本次任务对应的待办项。
- 每次需求新增或变更时，必须同步更新 `docs/` 下对应产品文档；如果影响开发计划、验收项或完成状态，也必须同步更新 `TODO.md`。
- 完成实现后必须进行验收和测试。
- 前端页面、交互、原型和可视化功能优先使用浏览器验收；如环境支持 browser/computer 工具，应自动打开页面完成关键路径检查。
- 后端、接口、构建、部署和安全策略必须运行对应的自动化测试或最小可验证命令。
- 如果无法自动验收，必须在最终回复中明确请用户验收，并说明需要检查的页面、命令或关键路径。
- 验收通过后，必须把 `TODO.md` 中对应项标记为完成。
- 不允许只完成代码修改而遗漏测试、验收说明或 TODO 状态更新。

## 3. 技术栈

后端：

- 语言：Go
- HTTP 框架：Gin
- ORM：GORM
- 数据库：PostgreSQL
- 迁移：golang-migrate
- 缓存/任务：Redis + Asynq
- Kubernetes：client-go
- API 契约：OpenAPI

前端：

- Vite
- React
- TypeScript
- Tailwind CSS
- shadcn/ui
- TanStack Query
- React Router
- React Hook Form + Zod
- i18next + react-i18next
- Sonner toast
- @antfu/eslint-config
- 包管理器：pnpm

Python：

- 必须使用 uv。
- 不使用 pip 直接管理项目依赖。
- Python 项目必须使用 `pyproject.toml` 和 `uv.lock`。

## 4. 命令规范

前端：

```bash
pnpm install
pnpm dev
pnpm build
pnpm lint
```

Python：

```bash
uv sync
uv add <package>
uv run <script>
```

Go：

```bash
go mod tidy
go test ./...
go run ./cmd/api
ENV_FILE=.env.local go run ./cmd/api
```

容器：

```bash
docker compose -f docker-compose-dev.yaml up -d
docker compose up --build
```

## 5. 推荐目录

仓库采用 monorepo：

- Go 后端在仓库根目录。
- 前端在 `web/` 目录。
- 本地中间件通过根目录 `docker-compose-dev.yaml` 启动 PostgreSQL 和 Redis。
- 不引入 SQLite，避免 CGO 和跨平台问题。
- 后端配置默认读取环境变量，也支持通过 `ENV_FILE=.env.local` 加载本地 `.env.*` 文件。
- `.env.*` 不提交，`.env.example` 作为可提交模板。
- 开发环境前端通过 Vite proxy 请求 `/api/v1`，不要默认直连后端绝对地址。
- 容器运行时 `web` 使用 Nginx 承载前端静态文件，并反代 `/api/` 到 `api:8080`。

```text
cmd/
  api/
  worker/
internal/
  auth/
  project/
  application/
  repository/
  registry/
  build/
  cluster/
  deployment/
  gateway/
  release/
  audit/
  config/
  secret/
  provider/
migrations/
openapi/

web/
  src/
    app/
    api/
    auth/
    components/
      ui/
      common/
    i18n/
    layouts/
    lib/
    pages/
    stores/
    styles/
```

## 6. 后端实现要求

- 第一阶段采用模块化单体 + 多进程部署。
- 后端至少包含 `cmd/api` 和 `cmd/worker` 两个入口。
- `cmd/api` 负责 HTTP API、Webhook、OAuth 回调、CRUD、权限校验和任务投递。
- `cmd/worker` 负责构建、部署、状态同步、证书申请、资源清理等异步任务。
- `api` 与 `worker` 共享内部领域模块，但不通过内部 HTTP 互相调用。
- 跨进程协作通过 PostgreSQL 状态表和 Redis/Asynq 任务队列完成。
- 长耗时任务必须进入 worker，不在 HTTP 请求中同步执行。
- Handler 只负责参数解析、权限校验入口和响应。
- 业务逻辑放 service 层。
- 数据访问放 repository 层。
- GORM 查询不要散落在 handler 中。
- 外部系统调用放 provider 层。
- Kubernetes 操作通过 RuntimeProvider 封装。
- 构建通过 BuildProvider 封装。
- 所有危险操作写 AuditLog。

## 7. 构建系统要求

MVP 主路径：

```text
Git webhook
  -> Platform API
  -> BuildRun
  -> Build Queue
  -> Kubernetes Job
  -> BuildKit rootless
  -> ContainerImage
```

要求：

- 每个 BuildRun 对应一个 Kubernetes Job。
- 构建 Job 运行在 build namespace。
- 不挂载宿主机 Docker socket。
- 不默认使用 privileged。
- Git token 和 registry token 通过 Secret 临时注入。
- 默认构建参数可被用户按权限修改。
- MVP 不做持久构建缓存，但保留 cache 字段。
- 构建 Job 默认必须启用 restricted 网络模式。
- 不允许构建 Job 默认拥有无限制 egress。
- 必须通过 BuildNetworkPolicy 描述构建出口策略。
- 默认允许公开 Git、公开 registry、公开包管理源和全局镜像加速源。
- 内网 registry 或内网包镜像源只能通过白名单或私有网段 TCP 443 放行。
- 私有网段非 443 端口默认禁止。
- 默认禁止访问 `169.254.169.254/32`、`127.0.0.0/8`、Kubernetes API Server 和 Service CIDR。
- DNS 只能访问平台配置的可信 DNS 解析器。

默认构建参数：

```text
cpuRequest: 500m
cpuLimit: 1
memoryRequest: 1Gi
memoryLimit: 2Gi
timeout: 20m
retry: 0
successJobRetention: 10m
failedJobRetention: 24h
failureLogTail: 200
```

## 8. 前端实现要求

- 前端只能使用 pnpm。
- 页面模块按 `src/pages/<module>` 拆分。
- shadcn/ui 组件放 `src/components/ui`。
- 跨页面业务组件放 `src/components/common`。
- 只被一个页面使用的组件可以留在页面模块内。
- 被两个及以上页面稳定复用的组件，必须抽离到 `src/components/common` 或更合适的共享目录。
- 相同的表格空状态、错误提示、确认弹窗、页面标题、详情块、状态 Badge、表单布局不得在多个页面重复手写。
- 错误展示必须用户友好，不允许直接把后端原始错误、OIDC 原始异常或技术堆栈展示给用户。
- OIDC 回调失败、state 不匹配、权限不足、未命中 OIDC 允许组、账号未被邀请、Token 过期等场景必须使用统一错误页或统一错误组件展示。
- 错误组件需要可复用，优先沉淀为 `ErrorState`、`AuthErrorPage`、`ForbiddenPage`、`NotFoundPage` 等公共组件。
- 错误页面必须包含清晰标题、用户能理解的原因、下一步操作和必要的返回入口。
- 服务端状态使用 TanStack Query。
- 表单使用 React Hook Form + Zod。
- 用户可见文案必须走 i18n。
- 通知使用 Sonner toast。
- 主题必须支持 light、dark、system 三态。
- 主题偏好写入 localStorage，system 模式监听系统主题变化。

## 9. 权限与安全

- 平台支持本地账号和 OIDC 登录。
- 内部平台不开放自由注册。
- 本地账号必须由管理员创建、邀请或导入。
- OIDC 登录成功后必须校验平台准入策略。
- OIDC 必须支持允许组白名单，group claim 默认优先使用 `groups`，但需要可配置。
- 可支持邮箱域白名单和邀请邮箱白名单。
- 未命中准入策略的 OIDC 用户不得自动创建平台账号。
- OIDC 组不能直接替代 ProjectMember，最终业务权限必须落入平台 RBAC。
- 权限由后端最终判断。
- 前端隐藏按钮只是体验优化。
- Access Token 不是 JWT，只保存 hash。
- Access Token 支持 scope、过期时间和撤销。
- Webhook 必须校验签名。
- 只接受已绑定仓库事件。
- Secret 不回显明文。

## 10. 不要做

MVP 不做：

- 完整计费系统
- 持久构建缓存
- Service Mesh
- CRD/Operator
- Docker Compose Runtime
- Vault / External Secrets
- 复杂模板市场
- 复杂 DAG 编排
- DNS Challenge
