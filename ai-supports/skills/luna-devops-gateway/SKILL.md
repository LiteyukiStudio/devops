---
name: luna-devops-gateway
description: 使用已安装的 Luna DevOps CLI 管理访问入口、域名检查、Gateway API、HTTPRoute、TLS、证书、访问地址和网关诊断；CLI 可用前仅用于规划。
---

# 网关 Skill

先遵循 `luna-devops-cli`，并从机器可读 Help 发现 gateway 命令。

## 操作流程

1. 确认 project、application、deployment target 和 service port。
2. 创建 route 前检查域名、path、protocol 和目标服务端口。
3. 发布后检查 route status、certificate status、HTTPRoute events。
4. 访问失败时先看 route，再看 Service 和 Pod readiness。

## 风险边界

- 修改或删除 gateway route 会影响公网访问，至少 medium risk。
- 泛域名证书使用 DNS-01；HTTP challenge 不支持 wildcard。
- 不把内部服务地址误认为公网访问地址。
