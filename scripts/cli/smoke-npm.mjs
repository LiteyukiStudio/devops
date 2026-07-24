import {
  existsSync,
  mkdtempSync,
  rmSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";

import {
  fail,
  isMainModule,
  parseArguments,
  requiredArgument,
  run,
} from "./lib.mjs";
import { smokeCommand } from "./smoke-command.mjs";

function executablePath(prefix, manager) {
  const candidates = process.platform === "win32"
    ? [
        join(prefix, "luna.cmd"),
        join(prefix, "bin", "luna.cmd"),
        join(prefix, "luna.CMD"),
        join(prefix, "bin", "luna.CMD"),
      ]
    : [
        join(prefix, "bin", "luna"),
        join(prefix, "luna"),
      ];
  const executable = candidates.find(candidate => existsSync(candidate));
  if (!executable) {
    throw new Error(
      `${manager} did not install luna in any expected location: ${candidates.join(", ")}`,
    );
  }
  return executable;
}

function smokeNpmInstall(tarball, expectedVersion) {
  const root = mkdtempSync(join(tmpdir(), "luna-cli-npm-"));
  const prefix = join(root, "prefix");
  try {
    run(
      "npm",
      [
        "install",
        "--global",
        "--ignore-scripts",
        `--prefix=${prefix}`,
        tarball,
      ],
      { timeout: 180_000, stdio: "inherit" },
    );
    smokeCommand(executablePath(prefix, "npm"), { expectedVersion });
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
}

function smokePnpmInstall(tarball, expectedVersion) {
  const root = mkdtempSync(join(tmpdir(), "luna-cli-pnpm-"));
  const pnpmHome = join(root, "pnpm-home");
  try {
    run(
      "pnpm",
      ["add", "--global", "--ignore-scripts", tarball],
      {
        env: {
          ...process.env,
          PNPM_HOME: pnpmHome,
          PATH: `${join(pnpmHome, "bin")}:${pnpmHome}:${process.env.PATH ?? ""}`,
        },
        timeout: 180_000,
        stdio: "inherit",
      },
    );
    smokeCommand(executablePath(pnpmHome, "pnpm"), { expectedVersion });
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
}

export function smokeNpmTarball(tarball, expectedVersion) {
  const path = resolve(tarball);
  if (!existsSync(path)) {
    throw new Error(`npm tarball does not exist: ${path}`);
  }
  smokeNpmInstall(path, expectedVersion);
  smokePnpmInstall(path, expectedVersion);
}

async function main() {
  const args = parseArguments(process.argv.slice(2));
  smokeNpmTarball(
    requiredArgument(args, "tarball"),
    args.get("version"),
  );
}

if (isMainModule(import.meta.url)) {
  main().catch(fail);
}
