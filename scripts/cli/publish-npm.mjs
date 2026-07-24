import { existsSync } from "node:fs";
import { resolve } from "node:path";

import {
  fail,
  isMainModule,
  parseArguments,
  readJson,
  repositoryRoot,
  requiredArgument,
  run,
  sha512Integrity,
} from "./lib.mjs";

const REGISTRY = "https://registry.npmjs.org/";

export function publishNpm({ tarball, npmTag, expectedVersion }) {
  const path = resolve(tarball);
  if (!existsSync(path)) {
    throw new Error(`npm tarball does not exist: ${path}`);
  }

  const packageJson = readJson(resolve(repositoryRoot, "cli/package.json"));
  if (expectedVersion && packageJson.version !== expectedVersion) {
    throw new Error(
      `Expected version ${expectedVersion}, found ${packageJson.version}`,
    );
  }

  const packageVersion = `${packageJson.name}@${packageJson.version}`;
  const localIntegrity = sha512Integrity(path);
  const remote = run(
    "npm",
    ["view", packageVersion, "dist.integrity", "--json", `--registry=${REGISTRY}`],
    { allowFailure: true, timeout: 60_000 },
  );

  if (remote.status === 0) {
    const remoteIntegrity = JSON.parse(remote.stdout);
    if (remoteIntegrity !== localIntegrity) {
      throw new Error(
        `${packageVersion} is already published with different integrity. `
        + "Published npm versions are immutable; release a new version.",
      );
    }
    process.stdout.write(
      `${packageVersion} already exists with matching integrity; skipping publish.\n`,
    );
    return { published: false, integrity: localIntegrity };
  }

  const combinedError = `${remote.stdout ?? ""}\n${remote.stderr ?? ""}`;
  if (!/E404|404 Not Found|is not in this registry/i.test(combinedError)) {
    throw new Error(
      `Unable to determine whether ${packageVersion} exists:\n${combinedError.trim()}`,
    );
  }

  run(
    "npm",
    [
      "publish",
      path,
      "--access=public",
      `--tag=${npmTag}`,
      `--registry=${REGISTRY}`,
    ],
    { timeout: 180_000, stdio: "inherit" },
  );
  return { published: true, integrity: localIntegrity };
}

async function main() {
  const args = parseArguments(process.argv.slice(2));
  const result = publishNpm({
    tarball: requiredArgument(args, "tarball"),
    npmTag: requiredArgument(args, "npm-tag"),
    expectedVersion: args.get("version"),
  });
  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
}

if (isMainModule(import.meta.url)) {
  main().catch(fail);
}
