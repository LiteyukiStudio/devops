# API 文档

**项目名称**：Liteyuki DevOps  
**作者**：sfkm  
**日期**：2026-06-22  
**版本**：1.26.622.52

## 变更日志

| 版本 | 日期 | 作者 | 变更内容 |
| --- | --- | --- | --- |
| 1.26.622.52 | 2026-06-22 | sfkm | 建立标准工程文档，梳理 API 分组 |

---

## 一、接口总览

所有业务接口位于 `/api/v1`。健康检查位于 `/healthz`。API 默认使用 JSON，请求认证支持 HttpOnly Cookie Session 和 Bearer Access Token。

| 分组 | 路径前缀 | 说明 |
| --- | --- | --- |
| 公共配置 | `/public`, `/configs` | 站点公开配置和后台配置 |
| 认证 | `/auth` | 登录、OIDC、准入策略、bootstrap |
| 用户 | `/users` | 当前用户、用户管理、外部身份 |
| Git | `/git` | Git Provider、Git 账号、仓库、Webhook |
| 镜像站 | `/registries`, `/container-images` | 镜像站、凭据、镜像 |
| 构建变量 | `/build/variable-sets` | 构建变量和密钥集合 |
| 运行集群 | `/runtime/clusters` | Kubernetes 集群和资源 |
| 应用市场 | `/app-templates` | 模板列表和模板安装 |
| 项目空间 | `/projects` | 项目、成员、应用、部署、发布、访问入口 |
| 计费 | `/billing` | 钱包摘要、费率、用量、账本 |
| Access Token | `/access-tokens` | 个人 API Token |

## 二、通用约定

### 2.1 分页响应

潜在大列表返回统一分页结构：

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

### 2.2 错误响应

错误响应应包含稳定 `code`，前端按 code 做本地化：

```json
{
  "error": "request failed",
  "code": "resource.not_found",
  "detail": "optional detail"
}
```

### 2.3 鉴权

| 方式 | 用途 |
| --- | --- |
| Cookie Session | 控制台用户访问 |
| Bearer Access Token | 脚本、Webhook 外部调用和自动化 |

## 三、接口详情摘要

### 3.1 认证与用户

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/auth/bootstrap` | 查询是否需要初始化管理员 |
| POST | `/auth/bootstrap/admin` | 初始化首个管理员 |
| POST | `/auth/login` | 本地账号登录 |
| POST | `/auth/logout` | 退出登录 |
| GET | `/auth/providers` | 列出 OIDC Provider |
| POST | `/auth/providers` | 创建 OIDC Provider |
| PUT | `/auth/providers/:providerId` | 更新 OIDC Provider |
| GET | `/users/me` | 当前用户 |
| PUT | `/users/me` | 更新当前用户资料 |
| GET | `/users` | 用户列表 |
| POST | `/users` | 创建用户 |

### 3.2 项目空间与应用

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/projects` | 项目空间列表 |
| POST | `/projects` | 创建项目空间 |
| GET | `/projects/:projectId` | 项目空间详情 |
| PUT | `/projects/:projectId` | 更新项目空间 |
| DELETE | `/projects/:projectId` | 删除项目空间 |
| GET | `/projects/:projectId/members` | 成员列表 |
| POST | `/projects/:projectId/members` | 添加成员 |
| GET | `/projects/:projectId/applications` | 应用列表 |
| POST | `/projects/:projectId/applications` | 创建应用 |
| GET | `/projects/:projectId/applications/:applicationId` | 应用详情 |
| PUT | `/projects/:projectId/applications/:applicationId` | 更新应用 |
| DELETE | `/projects/:projectId/applications/:applicationId` | 删除应用 |

### 3.3 构建与部署

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/projects/:projectId/build-runs` | 构建运行列表 |
| POST | `/projects/:projectId/build-runs/trigger` | 触发构建 |
| GET | `/projects/:projectId/build-runs/:runId` | 构建详情 |
| POST | `/projects/:projectId/build-runs/:runId/retry` | 重试构建 |
| POST | `/projects/:projectId/build-runs/:runId/cancel` | 取消构建 |
| GET | `/projects/:projectId/build-jobs/:jobId/logs` | 构建日志 |
| GET | `/projects/:projectId/build-jobs/:jobId/logs/stream` | 构建日志 SSE |
| GET | `/projects/:projectId/releases` | Release 列表 |
| POST | `/projects/:projectId/releases` | 创建 Release |
| POST | `/projects/:projectId/releases/:releaseId/rollback` | 回滚 Release |
| GET | `/projects/:projectId/releases/:releaseId/runtime-logs` | 运行时日志 |
| POST | `/projects/:projectId/releases/:releaseId/exec` | 运行时命令执行 |
| GET | `/projects/:projectId/releases/:releaseId/terminal` | 运行时终端 |

### 3.4 运行集群

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/runtime/clusters` | 集群列表 |
| POST | `/runtime/clusters` | 创建集群 |
| PUT | `/runtime/clusters/:clusterId` | 更新集群 |
| POST | `/runtime/clusters/:clusterId/test` | 测试 kubeconfig |
| GET | `/runtime/clusters/:clusterId/resources` | 查询集群资源 |
| GET | `/runtime/clusters/:clusterId/resource-yaml` | 查询资源 YAML |
| GET | `/runtime/clusters/:clusterId/resource-events` | 查询资源事件 |
| DELETE | `/runtime/clusters/:clusterId/resources` | 删除资源 |

### 3.5 计费

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/billing/summary` | 余额和摘要 |
| GET | `/billing/application-spend` | 应用维度花费 |
| GET | `/billing/ledger` | 账本列表 |
| GET | `/billing/usage-records` | 用量记录 |
| GET | `/billing/rate-rules` | 费率规则 |
| PUT | `/billing/rate-rules` | 更新费率 |
| POST | `/billing/wallet-transactions` | 创建充值或补偿 |
| POST | `/billing/gateway-traffic` | 写入网关流量用量 |

## 四、异常场景

| 场景 | HTTP 状态 | 处理 |
| --- | --- | --- |
| 未登录 | 401 | 前端跳转登录页 |
| 权限不足 | 403 | 展示友好权限页 |
| 资源不存在 | 404 | 展示空状态或错误状态 |
| 表单冲突 | 409 | 展示字段级或 toast 错误 |
| 外部平台失败 | 502/500 | 后端屏蔽敏感细节，返回稳定 code |
| 构建/部署失败 | 200 + 状态字段 | 在 BuildRun/Release 中展示失败状态和日志 |
