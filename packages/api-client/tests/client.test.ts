import { describe, expect, it, vi } from "vitest"

import {
  FetchHttpTransport,
  LunaClient,
  type FetchLike,
} from "../src/index.js"

function clientWithFetch(fetch: FetchLike, options: {
  retry?: false | { baseDelayMs: number, jitterRatio: number, maxAttempts: number }
  token?: string
} = {}): LunaClient {
  return new LunaClient({
    baseUrl: "https://luna.example",
    requestIdFactory: () => "req_client_test",
    retry: options.retry ?? false,
    tokenProvider: options.token
      ? { getAccessToken: () => options.token }
      : undefined,
    transport: new FetchHttpTransport({ fetch }),
  })
}

describe("LunaClient", () => {
  it("normalizes URL, query, JSON body, bearer auth and request IDs", async () => {
    const fetch = vi.fn<FetchLike>(async (_input, init) => {
      const requestHeaders = new Headers(init?.headers)
      expect(requestHeaders.get("authorization")).toBe("Bearer access-token")
      expect(requestHeaders.get("content-type")).toBe("application/json")
      expect(requestHeaders.get("x-request-id")).toBe("req_client_test")
      expect(init?.body).toBe(JSON.stringify({ name: "demo" }))
      return new Response(JSON.stringify({ id: "prj_demo" }), {
        headers: {
          "content-type": "application/json",
          "x-request-id": "req_server",
        },
        status: 201,
      })
    })
    const client = clientWithFetch(fetch, { token: "access-token" })

    const result = await client.request<{ id: string }>({
      body: { name: "demo" },
      method: "POST",
      path: "/api/v1/projects",
      query: {
        enabled: true,
        labels: ["one", "two"],
        omitted: undefined,
      },
    })

    expect(result).toMatchObject({
      data: { id: "prj_demo" },
      ok: true,
      requestId: "req_server",
      status: 201,
    })
    const requestedUrl = new URL(String(fetch.mock.calls[0]?.[0]))
    expect(requestedUrl.pathname).toBe("/api/v1/projects")
    expect(requestedUrl.searchParams.get("enabled")).toBe("true")
    expect(requestedUrl.searchParams.getAll("labels")).toEqual(["one", "two"])
    expect(requestedUrl.searchParams.has("omitted")).toBe(false)
  })

  it("rejects cross-origin requests before token injection", async () => {
    const fetch = vi.fn<FetchLike>()
    const client = clientWithFetch(fetch, { token: "must-not-leak" })

    const result = await client.request({
      path: "https://attacker.example/collect",
    })

    expect(result).toMatchObject({
      error: {
        code: "invalid_request",
        retryable: false,
      },
      ok: false,
      status: 0,
    })
    expect(JSON.stringify(result)).not.toContain("must-not-leak")
    expect(fetch).not.toHaveBeenCalled()
  })

  it("does not expose token provider failures", async () => {
    const fetch = vi.fn<FetchLike>()
    const client = new LunaClient({
      baseUrl: "https://luna.example",
      requestIdFactory: () => "req_provider",
      retry: false,
      tokenProvider: {
        getAccessToken: () => {
          throw new Error("provider leaked access-token-value")
        },
      },
      transport: new FetchHttpTransport({ fetch }),
    })

    const result = await client.request({ path: "/api/v1/projects" })

    expect(result).toMatchObject({
      error: {
        code: "invalid_request",
        message: "The access token provider failed",
      },
      ok: false,
    })
    expect(JSON.stringify(result)).not.toContain("access-token-value")
    expect(fetch).not.toHaveBeenCalled()
  })

  it("retries bounded safe requests and does not retry writes", async () => {
    const safeFetch = vi.fn<FetchLike>()
      .mockResolvedValueOnce(new Response(null, { status: 503 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ready: true }), {
        headers: { "content-type": "application/json" },
        status: 200,
      }))
    const retry = { baseDelayMs: 0, jitterRatio: 0, maxAttempts: 3 }
    const safeClient = clientWithFetch(safeFetch, { retry })

    const safeResult = await safeClient.request<{ ready: boolean }>({
      path: "/api/v1/health",
    })

    expect(safeResult).toMatchObject({ data: { ready: true }, ok: true })
    expect(safeFetch).toHaveBeenCalledTimes(2)

    const writeFetch = vi.fn<FetchLike>()
      .mockResolvedValue(new Response(null, { status: 503 }))
    const writeClient = clientWithFetch(writeFetch, { retry })
    const writeResult = await writeClient.request({
      body: { name: "unsafe-to-retry" },
      method: "POST",
      path: "/api/v1/projects",
    })

    expect(writeResult).toMatchObject({ ok: false, status: 503 })
    expect(writeFetch).toHaveBeenCalledTimes(1)
  })

  it("normalizes stable API errors and redacts sensitive details", async () => {
    const fetch = vi.fn<FetchLike>().mockResolvedValue(new Response(JSON.stringify({
      error: {
        code: "validation_failed",
        details: {
          nested: { password: "hidden", safe: "visible" },
          token: "hidden",
        },
        fields: { name: "required" },
        message: "Invalid request",
        requestId: "req_error",
      },
    }), {
      headers: {
        "content-type": "application/json",
        "retry-after": "2",
      },
      status: 429,
    }))
    const client = clientWithFetch(fetch)

    const result = await client.request({ path: "/api/v1/projects" })

    expect(result).toMatchObject({
      error: {
        code: "validation_failed",
        details: {
          nested: { password: "[REDACTED]", safe: "visible" },
          token: "[REDACTED]",
        },
        fields: { name: "required" },
        requestId: "req_error",
        retryAfterMs: 2_000,
        retryable: true,
      },
      ok: false,
      status: 429,
    })
  })

  it("supports text and binary response adapters", async () => {
    const fetch = vi.fn<FetchLike>()
      .mockResolvedValueOnce(new Response("plain text", { status: 200 }))
      .mockResolvedValueOnce(new Response(new Uint8Array([1, 2, 3]), { status: 200 }))
    const client = clientWithFetch(fetch)

    const text = await client.request<string>({
      path: "/text",
      responseType: "text",
    })
    const binary = await client.request<Uint8Array>({
      path: "/binary",
      responseType: "binary",
    })

    expect(text).toMatchObject({ data: "plain text", ok: true })
    expect(binary.ok && [...binary.data]).toEqual([1, 2, 3])
  })

  it("returns a normalized invalid-response error for malformed JSON", async () => {
    const fetch = vi.fn<FetchLike>().mockResolvedValue(new Response("not-json", {
      headers: { "content-type": "application/json" },
      status: 200,
    }))
    const client = clientWithFetch(fetch)

    const result = await client.request({ path: "/invalid" })

    expect(result).toMatchObject({
      error: { code: "invalid_response", retryable: false },
      ok: false,
      status: 0,
    })
  })
})
