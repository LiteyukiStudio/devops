import { describe, expect, it, vi } from "vitest"

import {
  FetchHttpTransport,
  LunaClient,
  type FetchLike,
  type LunaSseEvent,
} from "../src/index.js"

describe("Luna SSE adapter", () => {
  it("parses SSE events without relying on EventSource globals", async () => {
    const fetch = vi.fn<FetchLike>(async (_input, init) => {
      expect(new Headers(init?.headers).get("accept")).toBe("text/event-stream")
      return new Response([
        "id: evt_1",
        "event: progress",
        "data: {\"percent\":25}",
        "",
        "id: evt_2",
        "event: summary",
        "data: {\"status\":\"succeeded\"}",
        "",
        "",
      ].join("\n"), {
        headers: { "content-type": "text/event-stream" },
        status: 200,
      })
    })
    const client = new LunaClient({
      baseUrl: "https://luna.example",
      retry: false,
      transport: new FetchHttpTransport({ fetch }),
    })

    const result = await client.openSse<Record<string, unknown>>({
      parseData: raw => JSON.parse(raw) as Record<string, unknown>,
      path: "/api/v1/builds/1/events",
    })

    expect(result.ok).toBe(true)
    const events: LunaSseEvent<Record<string, unknown>>[] = []
    if (result.ok) {
      for await (const event of result.data) {
        events.push(event)
      }
    }
    expect(events).toEqual([
      {
        data: { percent: 25 },
        event: "progress",
        id: "evt_1",
        rawData: "{\"percent\":25}",
      },
      {
        data: { status: "succeeded" },
        event: "summary",
        id: "evt_2",
        rawData: "{\"status\":\"succeeded\"}",
      },
    ])
  })
})
