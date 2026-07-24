import { Writable } from "node:stream";
import { describe, expect, it } from "vitest";
import { LunaError } from "../../src/errors/index.js";
import { OutputChannels } from "../../src/output/index.js";

function capture(): { readonly stream: Writable; read(): string } {
  let value = "";
  return {
    stream: new Writable({
      write(chunk, _encoding, callback) {
        value += chunk.toString();
        callback();
      },
    }),
    read: () => value,
  };
}

describe("stdout and stderr discipline", () => {
  it("writes successful machine results only to stdout", () => {
    const stdout = capture();
    const stderr = capture();
    const channels = new OutputChannels({ stdout: stdout.stream, stderr: stderr.stream });
    channels.writeResult({ ok: true }, { format: "json" });
    expect(JSON.parse(stdout.read())).toEqual({ ok: true });
    expect(stderr.read()).toBe("");
  });

  it("writes failed machine results only to stderr and returns the exit code", () => {
    const stdout = capture();
    const stderr = capture();
    const channels = new OutputChannels({ stdout: stdout.stream, stderr: stderr.stream });
    const exitCode = channels.writeError(
      new LunaError("forbidden", "Denied", { status: 403 }),
      true,
    );
    expect(stdout.read()).toBe("");
    expect(JSON.parse(stderr.read())).toMatchObject({
      error: { code: "forbidden", status: 403 },
    });
    expect(exitCode).toBe(4);
  });

  it("quiet mode suppresses informational diagnostics but not errors", () => {
    const stdout = capture();
    const stderr = capture();
    const channels = new OutputChannels(
      { stdout: stdout.stream, stderr: stderr.stream },
      { quiet: true },
    );
    channels.writeInfo("progress");
    channels.writeWarning("warning");
    channels.writeError(new Error("failed"));
    expect(stderr.read()).toBe("internal_error: failed\n");
  });
});
