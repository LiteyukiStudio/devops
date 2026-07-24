import { describe, expect, it } from "vitest";

import {
  digestJson,
  sha256Hex,
  stableStringify,
} from "../src/digest.js";

describe("digest utilities", () => {
  it("matches standard SHA-256 vectors", () => {
    expect(sha256Hex("")).toBe(
      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    );
    expect(sha256Hex("abc")).toBe(
      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
    );
    expect(sha256Hex("轻雪")).toBe(
      "b23e705e9c749bd8d052a3db596529edef529f0485eac0d94189f319cd6779b6",
    );
  });

  it("serializes nested object keys deterministically", () => {
    const left = { z: 1, nested: { b: true, a: ["x", null] } };
    const right = { nested: { a: ["x", null], b: true }, z: 1 };

    expect(stableStringify(left)).toBe(
      '{"nested":{"a":["x",null],"b":true},"z":1}',
    );
    expect(stableStringify(left)).toBe(stableStringify(right));
    expect(digestJson(left)).toBe(digestJson(right));
  });

  it("rejects inputs that cannot produce stable JSON", () => {
    const cyclic: { self?: unknown } = {};
    cyclic.self = cyclic;

    expect(() => stableStringify(cyclic)).toThrow(/cyclic/i);
    expect(() => stableStringify(Number.NaN)).toThrow(/non-finite/i);
    expect(() => stableStringify(1n)).toThrow(/bigint/i);
  });
});
