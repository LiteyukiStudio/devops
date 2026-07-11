# 自动化验证

测试目录包含两类 smoke：

- `api-smoke.mjs`：覆盖健康检查、认证、用户、配置、项目、成员、应用、镜像站、镜像、Git Provider、Git 凭据、仓库绑定、Webhook、Access Token 等 API 域。
- `browser-smoke.mjs`：使用 Playwright 控制浏览器登录前端，并访问主要导航页面，确认页面可见且内容区非空。

默认地址：

- API：`http://127.0.0.1:8080/api/v1`
- Web：`http://127.0.0.1:5173`
- 开发账号：`admin@luna.dev / devops`

运行：

```bash
pnpm --dir tests install
bash tests/run-all.sh
```

如果本地端口不同：

```bash
API_BASE_URL=http://127.0.0.1:8080/api/v1 WEB_BASE_URL=http://127.0.0.1:4174 bash tests/run-all.sh
```
