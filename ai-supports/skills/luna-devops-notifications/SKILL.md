---
name: luna-devops-notifications
description: 使用已安装的 Luna DevOps CLI 管理通知预设、渠道、模板、规则、投递、测试通知、投递诊断和事件订阅；CLI 可用前仅用于规划。
---

# 通知 Skill

先遵循 `luna-devops-cli`，并从机器可读 Help 发现 notification 命令。

## 操作流程

1. 先确认用户想通知哪些事件。
2. 选择 channel preset 或自定义 channel。
3. 配置 template 和 rule。
4. 保存后发送 test notification。
5. 投递失败时检查 delivery status、channel 配置、外部 webhook/SMTP 响应。

## 风险边界

- Webhook URL 和 SMTP secret 不回显。
- 删除 channel/template/rule 会影响告警，应要求 confirmation。
- 测试通知可能触达外部系统，执行前说明目标。
