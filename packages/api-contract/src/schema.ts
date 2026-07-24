import { OPERATION_CATALOG_METADATA } from "./catalog.js";

export const CONTRACT_SCHEMA_NAME = "luna-api-contract" as const;
export const CONTRACT_SCHEMA_VERSION = 1 as const;

export interface ContractSchemaIdentity {
  readonly name: typeof CONTRACT_SCHEMA_NAME;
  readonly version: typeof CONTRACT_SCHEMA_VERSION;
  readonly apiVersion: string;
  readonly catalogVersion: string;
  readonly openapiDigest: `sha256:${string}`;
  readonly catalogDigest: `sha256:${string}`;
}

export const CONTRACT_SCHEMA_IDENTITY: ContractSchemaIdentity = Object.freeze({
  name: CONTRACT_SCHEMA_NAME,
  version: CONTRACT_SCHEMA_VERSION,
  apiVersion: OPERATION_CATALOG_METADATA.apiVersion,
  catalogVersion: OPERATION_CATALOG_METADATA.catalogVersion,
  openapiDigest: OPERATION_CATALOG_METADATA.openapiDigest,
  catalogDigest: OPERATION_CATALOG_METADATA.catalogDigest,
});

export function isSha256Digest(value: string): value is `sha256:${string}` {
  return /^sha256:[a-f0-9]{64}$/.test(value);
}

export function isCompatibleContractSchema(
  identity: Pick<ContractSchemaIdentity, "name" | "version">,
): boolean {
  return (
    identity.name === CONTRACT_SCHEMA_NAME &&
    identity.version === CONTRACT_SCHEMA_VERSION
  );
}
