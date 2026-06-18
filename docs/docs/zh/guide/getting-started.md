# Docker Compose 快速部署

这一页只做一件事：把 Liteyuki DevOps 跑起来，然后打开控制台。

## 准备环境文件

完整部署当前使用仓库根目录的 `docker-compose.yaml`。先准备 worker 需要的环境文件：

```bash
cp .env.worker.example .env.worker
```

如果你已经有自己的 Kubernetes 集群、镜像站或构建配置，可以稍后再编辑 `.env.worker`。第一次体验时，先保留默认值更省心。

## 启动平台

在仓库根目录执行：

```bash
docker compose up -d --build
```

这会启动 PostgreSQL、Redis、API 和 worker。API 镜像已经内嵌前端页面，所以不需要单独启动 Vite。

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
先跑起来，再慢慢配置。文档的目标是帮你少踩坑，不是让你先背完一套平台术语喵。
</div>
