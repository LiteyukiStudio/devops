# 通知

通知用于把平台内部事件发送到外部协作工具或邮箱。当前版本先覆盖失败类事件，避免把构建、发布和网关同步中的噪音全部推给管理员。

## 工作方式

通知链路分为四层：

1. 平台在构建、发布、Hook 或访问入口失败时产生结构化事件。
2. 通知规则按事件类型、严重级别、项目空间、应用和部署配置过滤事件。
3. 规则命中后生成投递记录，并交给 worker 异步发送。
4. 通知适配器把事件渲染为目标平台需要的请求或邮件。

业务模块只负责产生事件，不直接关心飞书、企业微信、SMTP 或自定义 Webhook 的细节。

## 渠道

通知渠道目前支持：

- Webhook：自定义 `method`、`url`、`headers` 和 JSON Body 模板，适合飞书、企业微信、Slack 类机器人，也可以对接自建告警入口。
- SMTP：通过标准 SMTP、STARTTLS 或 TLS 发送邮件。

Webhook 渠道会限制请求方法为 `POST`、`PUT` 或 `PATCH`，并按平台公共出站策略校验目标 URL，避免把通知发到内网敏感地址。渠道密钥写入 Secret Store，业务表只保存 secret 引用，API 响应只返回 `secretSet`。

## 预设快照

平台内置飞书 Bot 和企业微信 Bot 预设。通过预设创建渠道时，平台会把预设转成普通 Webhook 渠道和默认模板快照：

- 你只需要填写预设要求的 token 或 key。
- 已创建渠道不会跟随未来预设变更自动修改。
- 如果需要调整消息格式，可以编辑生成的模板或创建新的模板。

## 模板变量

通知模板使用 Go template，并开启缺失字段报错，避免变量写错后静默发送空内容。常用变量：

| 变量 | 说明 |
| --- | --- |
| `.Event.Type` | 事件类型，例如 `build.failed`。 |
| `.Event.Severity` | 严重级别，例如 `error`。 |
| `.Event.Message` | 失败摘要。 |
| `.Event.Project.Name` / `.Event.Project.Slug` | 项目空间名称和标识。 |
| `.Event.Application.Name` / `.Event.Application.Slug` | 应用名称和标识。 |
| `.Event.DeploymentTarget.Name` / `.Event.DeploymentTarget.Slug` | 部署配置名称和阶段。 |
| `.Event.Build.ID` / `.Event.Build.Image` / `.Event.Build.GitRef` | 构建上下文。 |
| `.Event.Release.ID` / `.Event.Release.ImageRef` / `.Event.Release.Revision` | 发布上下文。 |
| `.Event.Hook.Name` / `.Event.Hook.Phase` | Hook 上下文。 |
| `.Event.Gateway.Domain` / `.Event.Gateway.Path` | 访问入口上下文。 |
| `.Secrets.<Name>` | 渠道密钥值，仅在渲染时注入，不会回显到 API。 |

可用函数：

- `json`：把值安全编码为 JSON 字符串。
- `time`：格式化时间。
- `default`：为空时使用默认值。
- `truncate`：截断长文本。

## 规则

规则至少需要选择一个通知渠道。当前支持的失败事件：

- `build.failed`
- `release.failed`
- `hook.failed`
- `gateway.apply_failed`

过滤条件 JSON 示例：

```json
{
  "severities": ["error"],
  "projectIds": ["prj_xxx"],
  "applicationIds": [],
  "deploymentTargetIds": []
}
```

数组为空表示不过滤该维度。当前 UI 先提供全局管理员规则，后续可以扩展项目空间级规则。

## SMTP 示例

渠道配置 JSON 示例：

```json
{
  "host": "smtp.example.com",
  "port": 587,
  "security": "starttls",
  "username": "notice@example.com",
  "from": "Liteyuki DevOps <notice@example.com>",
  "to": ["ops@example.com"],
  "timeoutSeconds": 15
}
```

在密钥键值中填写：

```text
password=邮箱或 SMTP 密码
```

保存后密码会写入 Secret Store，后续编辑时不用再次填写；需要替换密码时重新填写 `password=...`。

## 验收

建议按以下顺序验收：

1. 创建一个 Webhook 或 SMTP 渠道。
2. 点击渠道的测试按钮，确认目标平台收到测试消息。
3. 创建或确认通知模板。
4. 创建规则，选择失败事件和通知渠道。
5. 人为触发一次构建失败或 Hook 失败。
6. 在投递记录里确认状态、尝试次数、错误信息和脱敏后的请求快照。
