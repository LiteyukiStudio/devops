import { createHash } from "node:crypto";
import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { parse } from "yaml";

const packageDirectory = resolve(fileURLToPath(new URL("..", import.meta.url)));
const repositoryRoot = resolve(packageDirectory, "../..");
const sourcePath = resolve(repositoryRoot, "openapi/openapi.yaml");
const outputPath = resolve(packageDirectory, "src/generated/operations.ts");
const source = await readFile(sourcePath, "utf8");
const document = parse(source);
const methods = ["get", "put", "post", "delete", "options", "head", "patch", "trace"];
const snapshots = [];

for (const [path, pathItem] of Object.entries(document.paths ?? {})) {
  for (const method of methods) {
    const operation = pathItem?.[method];
    if (!operation) continue;
    const parameters = [
      ...(pathItem.parameters ?? []),
      ...(operation.parameters ?? []),
    ].map(parameterSnapshot);
    const responses = Object.entries(operation.responses ?? {}).map(
      ([status, response]) => responseSnapshot(status, response),
    );
    snapshots.push(compact({
      method,
      path,
      tags: operation.tags ?? [],
      deprecated: operation.deprecated ?? false,
      security: operation.security ?? document.security ?? [],
      parameters,
      responses,
      summary: operation.summary,
      description: operation.description,
      operationId: operation.operationId,
      requestBody: operation.requestBody
        ? requestBodySnapshot(operation.requestBody)
        : undefined,
      xLunaCli: operation["x-luna-cli"],
    }));
  }
}

const digest = createHash("sha256").update(source).digest("hex");
const metadata = {
  source: "openapi/openapi.yaml",
  openapiVersion: document.openapi,
  apiVersion: document.info?.version ?? "unknown",
  sourceDigest: `sha256:${digest}`,
  operationCount: snapshots.length,
};
const generated = `// Generated from openapi/openapi.yaml. Do not edit manually.
import type { OpenApiOperationSnapshot, OpenApiSnapshotMetadata } from "../types.js";

export const OPENAPI_SNAPSHOT_METADATA = ${JSON.stringify(metadata, null, 2)} as const satisfies OpenApiSnapshotMetadata;

export const OPENAPI_OPERATION_SNAPSHOTS = ${JSON.stringify(snapshots, null, 2)} as const satisfies readonly OpenApiOperationSnapshot[];
`;

await writeFile(outputPath, generated, "utf8");

function parameterSnapshot(parameter) {
  const resolved = resolveReference(parameter);
  return compact({
    name: resolved?.name,
    in: resolved?.in,
    required: resolved?.required,
    description: resolved?.description,
    ref: parameter?.$ref,
    schema: schemaSummary(resolveReference(resolved?.schema)),
  });
}

function requestBodySnapshot(requestBody) {
  const resolved = resolveReference(requestBody) ?? {};
  const content = resolved.content ?? {};
  return {
    required: resolved.required ?? false,
    contentTypes: Object.keys(content),
    schemaRefs: schemaReferences(content),
  };
}

function responseSnapshot(status, response) {
  const resolved = resolveReference(response) ?? {};
  const content = resolved.content ?? {};
  return compact({
    status,
    contentTypes: Object.keys(content),
    schemaRefs: schemaReferences(content),
    description: resolved.description,
  });
}

function schemaReferences(content) {
  return [...new Set(
    Object.values(content)
      .flatMap(media => collectSchemaReferences(media?.schema)),
  )];
}

function collectSchemaReferences(schema, refs = []) {
  if (!schema || typeof schema !== "object") return refs;
  if (typeof schema.$ref === "string") refs.push(schema.$ref);
  for (const value of Object.values(schema)) {
    if (Array.isArray(value)) {
      for (const item of value) collectSchemaReferences(item, refs);
    } else if (value && typeof value === "object") {
      collectSchemaReferences(value, refs);
    }
  }
  return refs;
}

function schemaSummary(schema) {
  if (!schema || typeof schema !== "object") return undefined;
  return compact({
    ref: schema.$ref,
    type: schema.type,
    format: schema.format,
    enum: schema.enum,
    nullable: schema.nullable,
  });
}

function resolveReference(value) {
  if (!value?.$ref) return value;
  if (!value.$ref.startsWith("#/")) return value;
  return value.$ref
    .slice(2)
    .split("/")
    .map(part => part.replaceAll("~1", "/").replaceAll("~0", "~"))
    .reduce((current, part) => current?.[part], document);
}

function compact(value) {
  return Object.fromEntries(
    Object.entries(value).filter(([, entry]) => entry !== undefined),
  );
}
