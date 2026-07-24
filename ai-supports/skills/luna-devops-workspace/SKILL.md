---
name: luna-devops-workspace
description: 使用已安装的 Luna DevOps CLI 管理看板、项目空间、成员、固定项、排序、摘要和项目级访问检查；CLI 可用前仅用于规划，不执行平台操作。
---

# 项目空间 Skill

先遵循 `luna-devops-cli` 的可用性、命令发现、上下文、输出和确认规范。

## 命令发现

- 从 `luna help catalog query=project category=project limit=20 agent=true` 查找 dashboard、project 和成员相关命令，再通过 `luna help command path=<category.tool> agent=true` 读取参数契约。
- 目录中没有对应能力时，明确报告 CLI 尚未支持，不根据 API 名称猜命令。

## 操作流程

1. 获取当前身份、上下文和可见项目空间。
2. 将用户输入的项目名称解析为稳定 `projectId`。
3. 读取项目概览、应用、构建、发布、入口和事件摘要。
4. 成员变更前检查 actor 的项目角色。
5. 删除前读取依赖和影响，等待明确确认后再使用 Help 声明的确认参数。

## 风险边界

- 删除项目空间是高风险操作。
- 成员角色变更会改变授权范围。
- 普通 Viewer 只查看，不创建、修改或删除。
