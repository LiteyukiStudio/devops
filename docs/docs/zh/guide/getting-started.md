# Docker Compose 快速部署

这一页只做一件事：把 Liteyuki DevOps 跑起来，然后打开控制台。

## 选择镜像版本

完整部署当前使用仓库根目录的 `docker-compose.yaml`，默认直接拉取 DockerHub 镜像：

```text
liteyukistudio/devops-api:nightly
liteyukistudio/devops-worker:nightly
```

需要验证指定版本时，在启动命令前设置 `DEVOPS_IMAGE_TAG`，例如 `v0.1.0-rc.1`。

## 启动平台

在仓库根目录执行：

```bash
docker compose up -d
```

这会启动 PostgreSQL、Redis、API 和 worker。API 镜像已经内嵌前端页面，所以不需要单独启动 Vite。

如果你想从当前源码构建镜像，而不是拉取 DockerHub 镜像，使用：

```bash
docker compose -f docker-compose-build.yaml up -d --build
```

## 打开控制台

浏览器访问：

```text
http://localhost:8088
```

默认 Compose 会把 API 暴露到宿主机 `8088`，数据库和 Redis 只在容器网络里使用，不会占用宿主机的 `5432` 和 `6379`。

## 停止服务

```bash
docker compose down
```

如果你想连数据一起清理：

```bash
docker compose down -v
```

<div class="hint">
先跑起来，再慢慢配置。文档的目标是帮你少踩坑，不是让你先背完一套平台术语。
</div>
