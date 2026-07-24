import { describe, expect, it } from "vitest";

import {
  CONTRACT_SCHEMA_IDENTITY,
  CONTRACT_SCHEMA_NAME,
  CONTRACT_SCHEMA_VERSION,
  isCompatibleContractSchema,
  isSha256Digest,
} from "../src/schema.js";

describe("contract schema identity", () => {
  it("publishes versioned and digest-addressable metadata", () => {
    expect(CONTRACT_SCHEMA_IDENTITY).toMatchObject({
      name: CONTRACT_SCHEMA_NAME,
      version: CONTRACT_SCHEMA_VERSION,
    });
    expect(isSha256Digest(CONTRACT_SCHEMA_IDENTITY.openapiDigest)).toBe(true);
    expect(isSha256Digest(CONTRACT_SCHEMA_IDENTITY.catalogDigest)).toBe(true);
  });

  it("checks major schema compatibility without runtime-specific state", () => {
    expect(
      isCompatibleContractSchema({
        name: CONTRACT_SCHEMA_NAME,
        version: CONTRACT_SCHEMA_VERSION,
      }),
    ).toBe(true);
    expect(
      isCompatibleContractSchema({
        name: CONTRACT_SCHEMA_NAME,
        version: 2 as typeof CONTRACT_SCHEMA_VERSION,
      }),
    ).toBe(false);
    expect(isSha256Digest("sha256:not-a-digest")).toBe(false);
  });
});
