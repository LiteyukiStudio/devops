import { resolve } from "node:path";

import {
  fail,
  isMainModule,
  parseArguments,
  readJson,
  repositoryRoot,
  requiredArgument,
  writeGithubOutput,
} from "./lib.mjs";

const SEMVER_PATTERN
  = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/;

export function resolveReleaseMetadata(tag, packageVersion) {
  if (!tag.startsWith("cli-v")) {
    throw new Error(`CLI release tag must start with cli-v: ${tag}`);
  }

  const version = tag.slice("cli-v".length);
  const match = SEMVER_PATTERN.exec(version);
  if (!match) {
    throw new Error(`CLI release tag contains an invalid SemVer: ${tag}`);
  }
  const prereleaseIdentifiers = (match[4] ?? "").split(".").filter(Boolean);
  if (
    prereleaseIdentifiers.some(
      identifier => /^\d+$/.test(identifier) && identifier.length > 1 && identifier.startsWith("0"),
    )
  ) {
    throw new Error(`CLI release tag contains an invalid SemVer: ${tag}`);
  }
  if (version !== packageVersion) {
    throw new Error(
      `Tag version ${version} does not match cli/package.json version ${packageVersion}`,
    );
  }

  const prerelease = match[4] ?? "";
  const npmTag = prerelease
    ? prerelease === "beta" || prerelease.startsWith("beta.")
      ? "beta"
      : "next"
    : "latest";

  return {
    tag,
    version,
    npm_tag: npmTag,
    prerelease: String(Boolean(prerelease)),
  };
}

async function main() {
  const args = parseArguments(process.argv.slice(2));
  const tag = requiredArgument(args, "tag");
  const packageJson = readJson(resolve(repositoryRoot, "cli/package.json"));
  writeGithubOutput(resolveReleaseMetadata(tag, packageJson.version));
}

if (isMainModule(import.meta.url)) {
  main().catch(fail);
}
