import { describe, expect, it } from "vitest";
import {
  CLI_API_VERSION,
  CLI_STREAM_VERSION,
  createStreamEvent,
  createStreamSummary,
  createStreamVersionEvent,
  createSuccessEnvelope,
} from "../../src/output/index.js";

describe("versioned output envelopes", () => {
  it("creates a stable success envelope and redacts sensitive metadata", () => {
    const envelope = createSuccessEnvelope(
      "project.list/v1",
      "project.list",
      "project.list",
      { items: [{ id: "prj_1", token: "do-not-print" }] },
      { requestId: "req_1" },
    );
    expect(envelope).toMatchObject({
      apiVersion: CLI_API_VERSION,
      schemaVersion: "project.list/v1",
      operationId: "project.list",
      command: "project.list",
      meta: { requestId: "req_1" },
    });
    expect(envelope.data).toEqual({ items: [{ id: "prj_1", token: "[REDACTED]" }] });
  });

  it("creates version, correlated data and summary stream events", () => {
    const version = createStreamVersionEvent({
      cliVersion: "0.1.0",
      serverVersion: "0.1.0",
      schemaDigest: "sha256:test",
    });
    const event = createStreamEvent({
      type: "progress",
      sequence: 1,
      eventId: "evt_1",
      correlationId: "req_1",
      operationId: "build.wait",
      resourceRef: { kind: "BuildRun", id: "bldr_1" },
      occurredAt: "2026-07-25T10:00:00Z",
      data: { percent: 20 },
    });
    const summary = createStreamSummary({
      sequence: 2,
      eventId: "evt_2",
      correlationId: "req_1",
      operationId: "build.wait",
      resourceRef: { kind: "BuildRun", id: "bldr_1" },
      occurredAt: "2026-07-25T10:01:00Z",
      data: { status: "succeeded", exitCode: 0 },
    });

    expect(version).toMatchObject({ type: "version", streamVersion: CLI_STREAM_VERSION });
    expect(event).toMatchObject({ type: "progress", sequence: 1, eventId: "evt_1" });
    expect(summary).toMatchObject({ type: "summary", sequence: 2, eventId: "evt_2" });
  });
});
