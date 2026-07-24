---
name: luna-devops-system
description: 使用已安装的 Luna DevOps CLI 管理全局设置、公开配置、数据保留、应用模板、系统组件和平台诊断；CLI 可用前仅用于规划。
---

# 系统管理 Skill

先遵循 `luna-devops-cli`，并从机器可读 Help 发现 config、data retention、app template 和 system component 命令。

## 操作流程

1. 先确认是否需要平台管理员权限。
2. 修改站点配置前读取当前值并给出差异。
3. 数据保留 cleanup 前必须先 preview。
4. 应用市场安装前确认目标 project/application/runtime 配置。
5. 系统组件安装前检查已有安装状态。

## 风险边界

- data retention cleanup 是 critical，必须 confirmation + MFA step-up。
- 系统组件安装可能修改集群资源，至少 high risk。
- 站点配置会影响所有用户，至少 medium risk。
