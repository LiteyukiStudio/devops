# 部署平台

这一页只做一件事：用 Docker Compose 把 Liteyuki DevOps 跑起来，然后打开控制台。如果你想直接部署到 Kubernetes，看 [Helm 部署](/guide/helm-deployment)。

## 准备环境

你需要：

- 一台能运行 Docker 的机器。
- Docker Compose。
- 能拉取 DockerHub 镜像的网络。
- 宿主机 `8088` 端口空闲。

## 选择镜像版本

完整部署使用仓库根目录的 `docker-compose.yaml`，默认拉取：

```text
liteyukistudio/devops-api:nightly
liteyukistudio/devops-worker:nightly
```

验证指定版本时，在启动命令前设置 `DEVOPS_IMAGE_TAG`：

```bash
DEVOPS_IMAGE_TAG=v0.1.0-rc.1 docker compose up -d
```

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

## 确认服务正常

```bash
docker compose ps
docker compose logs -f api
docker compose logs -f worker
```

API 正常后，浏览器会打开控制台。worker 正常后，后续构建、部署和状态同步才会工作。

## 下一步

1. 进入 [初始化控制台](/guide/product)，创建管理员或登录。
2. 进入 [连接集群和镜像站](/guide/workspace)，准备运行集群和镜像站。
3. 按 [部署上线一个 Web 项目](/operations/deploy-web-project) 跑通第一条应用交付链路。

## 停止服务

```bash
docker compose down
```

如果你想连数据一起清理：

```bash
docker compose down -v
```

<div class="hint">
先跑起来，再慢慢配置。不要一开始就配置所有外部系统，第一目标是进入控制台。
</div>
