# 平台启动问题

这里处理 Docker Compose 启动时最常见的几类问题。应用构建或 Kubernetes 运行异常，请继续看“使用”部分的排障文档。

## 使用指定版本的镜像

默认 `docker-compose.yaml` 使用 `nightly` 镜像。验证 RC 或稳定版本时，在启动命令前设置 `DEVOPS_IMAGE_TAG`：

```bash
DEVOPS_IMAGE_TAG=v0.1.0-rc.1 docker compose up -d
```

如果你要从当前源码构建镜像，而不是拉取 DockerHub 镜像，使用源码构建 compose：

```bash
docker compose -f docker-compose-build.yaml up -d --build
```

## 端口 `8088` 被占用

查看占用：

```bash
lsof -nP -iTCP:8088 -sTCP:LISTEN
```

你可以停止占用进程，或者修改 `docker-compose.yaml` 里的端口映射：

```yaml
ports:
  - "8089:8080"
```

然后访问 `http://localhost:8089`。

## 页面能打开，但接口请求失败

先查看 API 日志：

```bash
docker compose logs -f api
```

再确认数据库和 Redis 是健康状态：

```bash
docker compose ps
```

## worker 没有正常启动

查看 worker 日志：

```bash
docker compose logs -f worker
```

Worker 负责构建、部署和状态同步。只有 API 运行时可以浏览控制台，但要真正构建和发布应用，Worker 也必须保持正常。
