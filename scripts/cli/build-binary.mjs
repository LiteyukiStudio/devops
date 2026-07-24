import { chmodSync, mkdirSync } from "node:fs";
import { dirname, resolve } from "node:path";

import {
  fail,
  isMainModule,
  parseArguments,
  repositoryRoot,
  requiredArgument,
  run,
} from "./lib.mjs";

export function buildBinary({ target, output }) {
  const outputPath = resolve(repositoryRoot, output);
  mkdirSync(dirname(outputPath), { recursive: true });

  run(
    "bun",
    [
      "build",
      "cli/src/entry.ts",
      "--compile",
      `--target=${target}`,
      `--outfile=${outputPath}`,
    ],
    { stdio: "inherit", timeout: 300_000 },
  );

  if (process.platform !== "win32") {
    chmodSync(outputPath, 0o755);
  }
  return outputPath;
}

async function main() {
  const args = parseArguments(process.argv.slice(2));
  buildBinary({
    target: requiredArgument(args, "target"),
    output: requiredArgument(args, "output"),
  });
}

if (isMainModule(import.meta.url)) {
  main().catch(fail);
}
