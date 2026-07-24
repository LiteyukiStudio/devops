---
name: luna-devops-security
description: 使用已安装的 Luna DevOps CLI 管理认证、MFA、OIDC、OAuth 应用与授权、用户、访问令牌、Scope、准入策略和二次验证；CLI 可用前仅用于规划。
---

# 安全与身份 Skill

先遵循 `luna-devops-cli`，并从机器可读 Help 发现 auth、user、OAuth app 和 access token 命令。

## 操作流程

1. 先确认 actor 是否是本人操作、项目成员操作还是平台管理员操作。
2. 用户管理前确认平台管理员权限。
3. OIDC/Auth provider 变更前检查 callback URL 和 step-up。
4. Access Token 创建时使用最小 scopes，并显式确认有效期或无限期风险。
5. OAuth Device Code 会话可按 CLI 流程完成 Step-up MFA；静态 Access Token 不可满足 Step-up。

## 风险边界

- MFA、Auth provider、用户管理、security settings 都是 high risk。
- Access Token 只显示一次，之后不回显。
- recovery codes 只在刚生成的安全展示流程中返回，不能写入日志或摘要。
- OAuth callback、token endpoint 和设备授权轮询由 CLI 实现，不由 Agent 手工编排。
