import { readFileSync } from "node:fs";
import { resolve } from "node:path";

import { fail, isMainModule, repositoryRoot, run } from "./lib.mjs";

export function verifyContractDrift() {
  const generatedFiles = [
    resolve(repositoryRoot, "packages/api-contract/src/generated/schema.ts"),
    resolve(repositoryRoot, "packages/api-contract/src/generated/operations.ts"),
  ];
  const before = generatedFiles.map(path => readFileSync(path, "utf8"));

  run("pnpm", ["--filter", "@luna-devops/api-contract", "generate"], {
    stdio: "inherit",
  });

  const changed = generatedFiles.filter(
    (path, index) => readFileSync(path, "utf8") !== before[index],
  );

  if (changed.length > 0) {
    throw new Error(
      `Generated API contract has drifted:\n${changed.join("\n")}\n`
      + "Run the contract generator and commit its output.",
    );
  }
}

if (isMainModule(import.meta.url)) {
  try {
    verifyContractDrift();
  } catch (error) {
    fail(error);
  }
}
