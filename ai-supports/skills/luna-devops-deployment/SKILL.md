---
name: luna-devops-deployment
description: 使用已安装的 Luna DevOps CLI 管理应用、部署配置、运行配置、发布、回滚、重启、候选镜像、资源规格和发布状态；CLI 可用前仅用于规划。
---

# 部署 Skill

先遵循 `luna-devops-cli`，并从机器可读 Help 发现 application、deployment 和 release 命令。

## 部署检查清单

1. 确认 project space、application 和 deployment target。
2. 检查 repository binding、build result、image candidate。
3. 检查 registry、runtime cluster、service ports 和 gateway route。
4. 准备 release plan：target、image、replicas、resources、env vars、route impact。
5. 创建 release、restart 或 rollback 前说明中断风险并等待确认。
6. 发布后检查 release status、runtime events、gateway route 和最近 platform events。

## 风险边界

- release、rollback 和删除是高风险操作。
- Secret 环境变量不回显，也不通过命令参数传递。
- runtime exec、terminal 和 data export 交由 `luna-devops-runtime` 的更严格流程处理。
