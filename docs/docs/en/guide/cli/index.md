# Luna CLI

Luna CLI is the command-line client for Luna DevOps. It is designed for interactive terminal use and for automation agents that need stable JSON contracts.

Commands use a fixed two-level structure:

```text
luna <category> <tool> key=value
```

For example, machine-readable command discovery uses:

```bash
luna help catalog query=project limit=5 output=json interactive=false
```

## Current development status

The CLI is under active `0.1.0` development. The source tree includes:

- multi-instance contexts and a default project context;
- Access Token login, validation, and local credential storage;
- `key=value`, JSON, file, and standard-input parameters;
- human-readable output and a versioned JSON envelope;
- local help, context, project, and completion command registration;
- all 109 operations currently documented by OpenAPI;
- a shared npm/Bun entry point, packaging, global-install smoke tests, and release gates.

Shared contracts and the API client are bundled safely into npm and Bun artifacts, so users do not need the monorepo workspace. No public package has been published yet; installation commands become available after the first `cli-v*` release.

## Design boundaries

- The CLI calls Luna DevOps backend APIs only. It does not orchestrate Kubernetes, GitHub, Gitea, or registry APIs directly.
- Automation should use `output=json interactive=false` and parse JSON from `stdout` only. Diagnostics belong on `stderr`.
- Local state defaults to `~/.luna/`. Tests and CI use a temporary `LUNA_HOME` and never read real user credentials.
- Medium-risk operations use the shared interactive confirmation flow. Non-interactive callers must set `yes=true`.
- High-risk API operations fail closed until the server-issued plan protocol exists; `yes=true` cannot bypass it.
- CLI and platform versions are independent. Compatibility is negotiated through server capabilities rather than a version-string comparison alone.

## Remaining release blockers

Before the first public release, the project must:

1. Document the remaining public backend routes in OpenAPI and complete command-coverage tests.
2. Add client capability negotiation, Authorization Code + PKCE, Device Code, and Bearer step-up MFA.
3. Complete SSE, WebSocket, download, and server-issued plan transports.
4. Configure an npm Trusted Publisher and protect the GitHub `npm` Environment.
5. Add Apple Developer ID/notarization and Windows Authenticode before desktop binaries enter stable releases.

See [Install and Use](./installation) and [Release Security](./release-security) for details.
