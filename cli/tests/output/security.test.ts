import { describe, expect, it } from "vitest";
import {
  redactSensitiveText,
  redactValue,
  sanitizeTerminalText,
} from "../../src/errors/index.js";
import { stringifyJson } from "../../src/output/index.js";

describe("output safety", () => {
  it("removes terminal control sequences and bidi controls", () => {
    const hostile = "\u001B]8;;https://evil.example\u0007click\u001B]8;;\u0007\u202Etxt";
    expect(sanitizeTerminalText(hostile)).toBe("clicktxt");
  });

  it("redacts credentials in objects and diagnostic text", () => {
    expect(redactValue({
      authorization: "Bearer secret",
      nested: { password: "secret" },
    })).toEqual({
      authorization: "[REDACTED]",
      nested: { password: "[REDACTED]" },
    });
    expect(redactSensitiveText("Authorization: Bearer abc.def")).not.toContain("abc.def");
  });

  it("keeps JSON parseable while escaping unsafe terminal characters", () => {
    const output = stringifyJson({ message: "safe\u202Etext", token: "secret" });
    expect(output).toContain("\\u202e");
    expect(JSON.parse(output)).toEqual({
      message: "safe\u202Etext",
      token: "[REDACTED]",
    });
  });
});
