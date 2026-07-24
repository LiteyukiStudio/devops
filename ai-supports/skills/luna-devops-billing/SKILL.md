---
name: luna-devops-billing
description: 使用已安装的 Luna DevOps CLI 查询账单摘要、消费、账本、用量、费率、账户流水、网关流量和 Credits 说明；CLI 可用前仅用于规划，不执行平台操作。
---

# 账单 Skill

先遵循 `luna-devops-cli`，并从机器可读 Help 发现 billing 命令。

## 操作流程

1. 查询余额和本期消耗。
2. 按 user/project/application/deployment target 解释 ledger 和 usage。
3. 对异常计费，比较 usage records、rate rules 和 ledger entries。
4. 充值或补偿写入用户账户，不挂项目空间。
5. 费率调整前说明影响范围和生效时点。

## 风险边界

- `billing:write` 是高风险，默认只允许平台管理员或明确授权流程。
- 不自动给用户补偿或扣费。
- 充值、补偿、扣费或费率调整前必须展示账户、金额、单位、生效时间和影响。
- 账单记录不因项目或应用删除而删除。
