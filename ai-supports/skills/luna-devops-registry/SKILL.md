---
name: luna-devops-registry
description: 使用已安装的 Luna DevOps CLI 管理 OCI 镜像站、镜像凭据、镜像仓库、标签、镜像记录和镜像模板；CLI 可用前仅用于规划。
---

# 镜像站 Skill

先遵循 `luna-devops-cli`，并从机器可读 Help 发现 registry 和 image 命令。

## 操作流程

1. 读取镜像站、作用域、凭据元数据和健康状态。
2. 添加或更新凭据时通过安全输入提供 secret，不使用命令参数。
3. 变更后使用 CLI 暴露的连接测试。
4. 构建前确认 push 权限和镜像模板。
5. 发布前确认 repository、tag 或 digest 确实存在。

## 安全边界

- 凭据变更是高风险操作，由 CLI 和服务端执行 MFA、权限和审计。
- 不返回 password、token、robot account secret。
- 删除镜像站前检查 build、release 和默认镜像站引用并确认影响。
