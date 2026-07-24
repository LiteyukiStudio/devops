import { describe, expect, it, vi } from "vitest"

import {
  FetchHttpTransport,
  HttpTransportError,
  type FetchLike,
} from "../src/index.js"

describe("FetchHttpTransport", () => {
  it("strips sensitive headers on cross-origin redirects", async () => {
    const fetch = vi.fn<FetchLike>()
      .mockResolvedValueOnce(new Response(null, {
        headers: { location: "https://cdn.example/artifact" },
        status: 302,
      }))
      .mockResolvedValueOnce(new Response("artifact", { status: 200 }))
    const transport = new FetchHttpTransport({ fetch })

    const response = await transport.send({
      headers: {
        authorization: "Bearer secret",
        cookie: "session=secret",
        "x-request-id": "req_redirect",
      },
      method: "GET",
      url: new URL("https://luna.example/download"),
    })

    expect(await response.text()).toBe("artifact")
    const redirectedHeaders = new Headers(fetch.mock.calls[1]?.[1]?.headers)
    expect(redirectedHeaders.has("authorization")).toBe(false)
    expect(redirectedHeaders.has("cookie")).toBe(false)
    expect(redirectedHeaders.get("x-request-id")).toBe("req_redirect")
  })

  it("rejects HTTPS downgrade redirects", async () => {
    const fetch = vi.fn<FetchLike>().mockResolvedValue(new Response(null, {
      headers: { location: "http://luna.example/insecure" },
      status: 302,
    }))
    const transport = new FetchHttpTransport({ fetch })

    await expect(transport.send({
      method: "GET",
      url: new URL("https://luna.example/secure"),
    })).rejects.toMatchObject<HttpTransportError>({
      code: "unsafe_redirect",
    })
  })

  it("normalizes request timeouts", async () => {
    const fetch = vi.fn<FetchLike>(async (_input, init) => {
      await new Promise<void>((_resolve, reject) => {
        init?.signal?.addEventListener("abort", () => reject(init.signal?.reason), {
          once: true,
        })
      })
      throw new Error("unreachable")
    })
    const transport = new FetchHttpTransport({ fetch })

    await expect(transport.send({
      method: "GET",
      timeoutMs: 5,
      url: new URL("https://luna.example/slow"),
    })).rejects.toMatchObject<HttpTransportError>({
      code: "request_timeout",
      retryable: true,
    })
  })
})

