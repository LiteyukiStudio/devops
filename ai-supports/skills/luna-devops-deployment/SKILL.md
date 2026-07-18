# Luna DevOps 部署 Skill

当需要通过 Luna DevOps MCP tools 帮用户构建、发布或开放访问 application 时使用这个 skill。

## 部署检查清单

1. 确认 project space 和 application。
2. 检查 repository binding 和 branch。
3. 检查 default registry 和 deployment target。
4. 查看近期 build runs 和 latest release。
5. 如需触发构建，必须先获得用户明确确认。
6. 等待 build result，并识别产出的 image。
7. 准备 release plan，包含 target、image、replicas、resources 和 route impact。
8. 创建 release 或 rollback 前，必须再次要求用户确认。
9. 发布后检查 status、gateway route 和 events。

## 确认规则

- trigger build 需要 confirmation。
- create release 需要 confirmation。
- rollback 需要 confirmation，确认文本应包含 release ID。
- 不要代替用户批准 runtime exec、terminal、data export 或 cleanup。

## 诊断提示

- 如果 build 在拉取 base image 前失败，检查 build logs 和 registry/network events。
- 如果 deployment 成功但访问失败，检查 gateway routes 和 runtime events。
- 如果 image tag 没变化，检查 deployment 是否使用 image digest、pull policy 或 force rollout strategy。
