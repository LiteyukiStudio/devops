# 常见启动问题

这里先放 Docker Compose 部署最常见的问题。复杂的构建和集群问题放在“使用”里慢慢查。

## `.env.worker` 不存在

如果启动时报 `env file .env.worker not found`，先执行：

```bash
cp .env.worker.example .env.worker
docker compose up -d --build
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

## 页面打开但接口失败

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

worker 负责构建、部署和状态同步。只浏览控制台时 API 可以先跑起来；要测试发布链路时，worker 必须正常运行。
