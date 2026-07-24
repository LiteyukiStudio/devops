---
name: luna-devops-runtime
description: 使用已安装的 Luna DevOps CLI 管理运行集群、Kubernetes 资源、YAML、事件、Pod 状态、日志、终端、命令执行、数据导出和运行时诊断；CLI 可用前仅用于规划。
---

# 运行集群 Skill

先遵循 `luna-devops-cli`，并从机器可读 Help 发现 `cluster` 分类下的集群与 Kubernetes 资源工具，以及相关发布运行态工具。

## 操作流程

1. 先确认 clusterId 和 project/application/deployment target 关联。
2. 集群问题先 test cluster，再查 resource events。
3. Pod 异常先看 workload、Pod status、events，再看 release/runtime logs。
4. YAML 用于诊断，不默认让用户复制执行。

## 风险边界

- kubeconfig update 是高风险操作，且 CLI 必须使用安全输入。
- 删除 cluster resource 是 high risk，默认只在用户明确指定资源后执行。
- terminal、runtime exec 和 data export 只有在 CLI Help 明确开放、OAuth 会话完成 MFA 且用户明确确认后才能执行。
- Access Token 不能替代需要 Step-up MFA 的交互式 OAuth 会话。
- 不返回 kubeconfig 内容。
