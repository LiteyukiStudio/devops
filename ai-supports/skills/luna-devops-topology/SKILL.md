---
name: luna-devops-topology
description: 使用已安装的 Luna DevOps CLI 管理项目拓扑、服务绑定、手工拓扑边、服务引用、环境变量注入、依赖检查和重新发布指引；CLI 可用前仅用于规划。
---

# 服务拓扑 Skill

先遵循 `luna-devops-cli`，并从机器可读 Help 发现 topology 和 service binding 命令。

## 两类关系

- 服务引用：影响部署结果，为源 deployment target 注入目标服务地址。
- 手工关系：只展示逻辑关系，不注入环境变量，不触发发布。

## 操作流程

1. 确认源应用和目标应用。
2. 如果需要运行时地址，使用 ServiceBinding。
3. 如果只是画架构关系，使用 ProjectTopologyEdge。
4. ServiceBinding 必须选择源/目标 deployment target、target port、protocol、injection mode。
5. 保存 ServiceBinding 后提示需要重新发布源 deployment target。
6. 诊断时使用 CLI 暴露的检查命令，区分 Service 不存在、端口不匹配、Endpoint 不可用或跨集群。

## 安全边界

- 不把用户名、密码、token 拼进服务地址。
- 跨项目空间和跨集群 ServiceBinding 第一版不支持。
- 删除被引用服务前必须先检查影响列表。
