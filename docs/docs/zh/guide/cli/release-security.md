# 发布与制品验证

Luna CLI 与 Luna DevOps 平台使用独立的版本和 tag 命名空间：

| Git tag | npm dist-tag | GitHub Release |
| --- | --- | --- |
| `cli-v1.2.3` | `latest` | 正式版 |
| `cli-v1.2.3-rc.1` | `next` | 预发布 |
| `cli-v1.2.3-beta.1` | `beta` | 预发布 |

普通 `v*` tag 仍用于平台发布，不会触发 CLI 发布。

## CI 门禁

CLI 相关变更会执行：

1. 使用锁文件安装 pnpm 工作区依赖。
2. 重新生成 API 契约并检查 drift。
3. TypeScript typecheck、ESLint、单元测试和构建。
4. 生成真实 npm tarball，并检查文件白名单。
5. 在干净临时目录中分别使用 npm 与 pnpm 全局安装同一个 tarball。
6. 构建当前 Linux host 的 Bun baseline 二进制并运行 smoke test。

发布工作流在这些门禁之外，还会构建明确的平台矩阵：

- Linux x64 baseline；
- Linux arm64；
- Linux x64 musl baseline，并在 Alpine 中执行额外 smoke；
- macOS arm64 和 x64 测试制品；
- Windows x64 测试制品。

## 签名边界

当前仓库没有 Apple Developer ID、公证和 Windows Authenticode 凭据。工作流不会假装这些制品已经签名：

- 正式版只包含经过目标环境 smoke 的 Linux 独立二进制；
- 预发布版可以包含名称带 `-unsigned` 的 macOS/Windows 测试制品；
- 未签名桌面制品不应进入生产环境；
- 接入对应签名和验证阶段后，才会扩展稳定制品矩阵。

## npm Trusted Publishing

npm 发布使用 GitHub OIDC Trusted Publishing，不保存长期写入 Token。维护者需要在 npm 包设置中绑定：

- Organization or user：`LiteyukiStudio`
- Repository：`devops`
- Workflow：`cli-release.yml`
- Environment：`npm`

GitHub `npm` Environment 应配置保护规则和维护者审批。发布 Job 只授予 `id-token: write`，不设置 `NPM_TOKEN` 或 `NODE_AUTH_TOKEN`。

如果同一版本已经存在，工作流会比较 npm `dist.integrity`：

- 内容一致：跳过 npm 发布，继续补齐 GitHub Release；
- 内容不同：立即失败，必须发布新版本。

## 校验下载

每个 Release 生成：

- `SHA256SUMS`；
- `RELEASE-MANIFEST.json`；
- SPDX JSON SBOM；
- GitHub OIDC build provenance；
- SBOM attestation bundle。

下载后至少校验 SHA-256：

```bash
grep " luna-linux-x64$" SHA256SUMS | sha256sum -c -
```

还应在 GitHub Release 的 Attestations 页面确认制品来自 `LiteyukiStudio/devops` 的 `cli-release.yml`，并检查 tag、commit 和制品名称是否与预期一致。
