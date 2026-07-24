import {
  existsSync,
  mkdirSync,
  readFileSync,
  rmSync,
} from "node:fs";
import { basename, resolve } from "node:path";

import {
  fail,
  isMainModule,
  parseArguments,
  readJson,
  repositoryRoot,
  run,
  writeGithubOutput,
} from "./lib.mjs";

const ALLOWED_TOP_LEVEL = new Set([
  "LICENSE",
  "README.md",
  "bin",
  "dist",
  "package.json",
]);

export function validatePublishManifest(packageJson) {
  for (const name of ["preinstall", "install", "postinstall"]) {
    if (packageJson.scripts?.[name]) {
      throw new Error(`Published package must not define ${name}`);
    }
  }

  const workspaceDependencies = Object.entries(packageJson.dependencies ?? {})
    .filter(([, version]) => String(version).startsWith("workspace:"))
    .map(([name]) => name);
  if (workspaceDependencies.length > 0) {
    throw new Error(
      `Published runtime dependencies still use workspace protocol: ${workspaceDependencies.join(", ")}. `
      + "Bundle or publish those dependencies before releasing the CLI.",
    );
  }
}

export function validatePackFiles(files) {
  const rejected = [];
  for (const file of files) {
    const path = file.path.replace(/^package\//, "");
    const topLevel = path.split("/")[0];
    if (
      !ALLOWED_TOP_LEVEL.has(topLevel)
      || path.includes("/tests/")
      || path.endsWith(".map")
      || /(^|\/)\.env(?:\.|$)/.test(path)
    ) {
      rejected.push(path);
    }
  }

  if (rejected.length > 0) {
    throw new Error(`npm tarball contains disallowed files:\n${rejected.join("\n")}`);
  }
}

export function packNpm(outputDirectory) {
  const cliDirectory = resolve(repositoryRoot, "cli");
  const packageJson = readJson(resolve(cliDirectory, "package.json"));
  validatePublishManifest(packageJson);

  for (const required of ["bin/luna.js", "dist"]) {
    if (!existsSync(resolve(cliDirectory, required))) {
      throw new Error(`CLI build output is missing: cli/${required}`);
    }
  }

  const destination = resolve(repositoryRoot, outputDirectory);
  mkdirSync(destination, { recursive: true });
  for (const name of ["package.json"]) {
    rmSync(resolve(destination, name), { force: true });
  }

  const packed = run(
    "npm",
    ["pack", "--json", `--pack-destination=${destination}`],
    { cwd: cliDirectory, timeout: 120_000 },
  );
  const result = JSON.parse(packed.stdout);
  if (!Array.isArray(result) || result.length !== 1) {
    throw new Error(`npm pack returned an unexpected result: ${packed.stdout}`);
  }

  validatePackFiles(result[0].files ?? []);
  const tarball = resolve(destination, result[0].filename);
  if (!existsSync(tarball)) {
    throw new Error(`npm pack did not create ${tarball}`);
  }

  return {
    tarball,
    filename: basename(tarball),
    integrity: result[0].integrity,
    packageSize: result[0].size,
    unpackedSize: result[0].unpackedSize,
  };
}

async function main() {
  const args = parseArguments(process.argv.slice(2));
  const result = packNpm(args.get("output") ?? "release/npm");
  writeGithubOutput({
    tarball: result.tarball,
    filename: result.filename,
  });
  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
}

if (isMainModule(import.meta.url)) {
  main().catch(fail);
}
