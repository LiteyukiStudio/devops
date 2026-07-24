# Release Security

Luna CLI and the Luna DevOps platform use separate version and tag namespaces:

| Git tag | npm dist-tag | GitHub Release |
| --- | --- | --- |
| `cli-v1.2.3` | `latest` | Stable |
| `cli-v1.2.3-rc.1` | `next` | Prerelease |
| `cli-v1.2.3-beta.1` | `beta` | Prerelease |

Plain `v*` tags remain reserved for platform releases and do not trigger CLI publishing.

## CI gates

CLI changes run these checks:

1. Install the locked pnpm workspace.
2. Regenerate the API contract and reject drift.
3. Run TypeScript typecheck, ESLint, unit tests, and the build.
4. Create a real npm tarball and validate its file allowlist.
5. Install the same tarball globally with npm and pnpm in clean temporary directories.
6. Build a Bun baseline binary for the Linux CI host and run command smoke tests.

The release workflow also builds an explicit target matrix:

- Linux x64 baseline;
- Linux arm64;
- Linux x64 musl baseline, with an additional Alpine smoke test;
- macOS arm64 and x64 test artifacts;
- Windows x64 test artifacts.

## Signing boundary

The repository does not currently have Apple Developer ID/notarization or Windows Authenticode credentials. The workflow does not claim that these artifacts are signed:

- stable releases contain only target-smoked Linux standalone binaries;
- prereleases may contain macOS and Windows test artifacts suffixed with `-unsigned`;
- unsigned desktop artifacts are not intended for production;
- desktop binaries enter the stable matrix only after platform signing and verification are integrated.

## npm Trusted Publishing

npm publishing uses GitHub OIDC Trusted Publishing without a long-lived write token. Maintainers configure the npm package with:

- Organization or user: `LiteyukiStudio`
- Repository: `devops`
- Workflow: `cli-release.yml`
- Environment: `npm`

The GitHub `npm` Environment should require protected tags and maintainer approval. The publishing job grants `id-token: write` and does not set `NPM_TOKEN` or `NODE_AUTH_TOKEN`.

When a version already exists, the workflow compares npm `dist.integrity`:

- matching content: skip npm publishing and continue repairing the GitHub Release;
- different content: fail immediately and require a new version.

## Verify downloads

Each GitHub Release contains:

- `SHA256SUMS`;
- `RELEASE-MANIFEST.json`;
- an SPDX JSON SBOM;
- GitHub OIDC build provenance;
- an SBOM attestation bundle.

At minimum, verify SHA-256:

```bash
grep " luna-linux-x64$" SHA256SUMS | sha256sum -c -
```

Also inspect GitHub Release Attestations and confirm that the artifact was produced by `LiteyukiStudio/devops`, the `cli-release.yml` workflow, and the expected tag and commit.
