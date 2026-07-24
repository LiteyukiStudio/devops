import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { appendFileSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

export const repositoryRoot = resolve(
  fileURLToPath(new URL("../..", import.meta.url)),
);

export function parseArguments(argv) {
  const result = new Map();

  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (!argument.startsWith("--")) {
      throw new Error(`Unexpected positional argument: ${argument}`);
    }

    const separator = argument.indexOf("=");
    if (separator !== -1) {
      result.set(argument.slice(2, separator), argument.slice(separator + 1));
      continue;
    }

    const name = argument.slice(2);
    const next = argv[index + 1];
    if (next && !next.startsWith("--")) {
      result.set(name, next);
      index += 1;
    } else {
      result.set(name, "true");
    }
  }

  return result;
}

export function requiredArgument(argumentsMap, name) {
  const value = argumentsMap.get(name);
  if (!value) {
    throw new Error(`Missing required argument: --${name}=<value>`);
  }
  return value;
}

export function readJson(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

export function run(command, args, options = {}) {
  const result = spawnSync(command, args, {
    cwd: options.cwd ?? repositoryRoot,
    env: options.env ?? process.env,
    encoding: "utf8",
    stdio: options.stdio ?? "pipe",
    timeout: options.timeout ?? 120_000,
  });

  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0 && !options.allowFailure) {
    const details = [result.stdout, result.stderr].filter(Boolean).join("\n");
    throw new Error(
      `${command} ${args.join(" ")} failed with exit code ${result.status}${
        details ? `\n${details.trim()}` : ""
      }`,
    );
  }

  return result;
}

export function digest(path, algorithm) {
  return createHash(algorithm).update(readFileSync(path)).digest("hex");
}

export function sha256(path) {
  return digest(path, "sha256");
}

export function sha512Integrity(path) {
  const value = createHash("sha512")
    .update(readFileSync(path))
    .digest("base64");
  return `sha512-${value}`;
}

export function isMainModule(metaUrl) {
  return Boolean(
    process.argv[1]
      && metaUrl === pathToFileURL(resolve(process.argv[1])).href,
  );
}

export function writeGithubOutput(values) {
  const lines = Object.entries(values).map(([key, value]) => `${key}=${value}`);
  if (process.env.GITHUB_OUTPUT) {
    appendFileSync(process.env.GITHUB_OUTPUT, `${lines.join("\n")}\n`);
    return;
  }
  process.stdout.write(`${JSON.stringify(values, null, 2)}\n`);
}

export function fail(error) {
  process.stderr.write(`${error instanceof Error ? error.message : error}\n`);
  process.exitCode = 1;
}
