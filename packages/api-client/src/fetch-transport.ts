import { HttpTransportError } from "./errors.js"
import type {
  HttpMethod,
  HttpTransport,
  HttpTransportRequest,
  HttpTransportResponse,
} from "./types.js"

export type FetchLike = (
  input: RequestInfo | URL,
  init?: RequestInit,
) => Promise<Response>

export interface FetchHttpTransportOptions {
  fetch?: FetchLike
  maxRedirects?: number
}

const REDIRECT_STATUSES = new Set([301, 302, 303, 307, 308])
const CROSS_ORIGIN_SENSITIVE_HEADERS = [
  "authorization",
  "cookie",
  "proxy-authorization",
]
const CONTENT_HEADERS = ["content-length", "content-type"]

class FetchHttpResponse implements HttpTransportResponse {
  private responseBody: ReadableStream<Uint8Array> | null | undefined
  private disposed = false

  constructor(
    private readonly response: Response,
    private readonly cleanup: () => void,
    private readonly signal: AbortSignal,
    private readonly timedOut: () => boolean,
  ) {}

  get body(): ReadableStream<Uint8Array> | null {
    if (this.responseBody === undefined) {
      this.responseBody = this.response.body ? this.wrapBody(this.response.body) : null
    }
    return this.responseBody
  }

  get headers(): Headers {
    return this.response.headers
  }

  get status(): number {
    return this.response.status
  }

  get statusText(): string {
    return this.response.statusText
  }

  get url(): string {
    return this.response.url
  }

  arrayBuffer(): Promise<ArrayBuffer> {
    return this.consume(() => this.response.arrayBuffer())
  }

  dispose(): void {
    if (this.disposed) {
      return
    }
    this.disposed = true
    this.cleanup()
  }

  text(): Promise<string> {
    return this.consume(() => this.response.text())
  }

  private async consume<T>(read: () => Promise<T>): Promise<T> {
    try {
      return await read()
    }
    catch {
      throw this.readError()
    }
    finally {
      this.dispose()
    }
  }

  private wrapBody(body: ReadableStream<Uint8Array>): ReadableStream<Uint8Array> {
    const reader = body.getReader()
    return new ReadableStream<Uint8Array>({
      cancel: async (reason) => {
        try {
          await reader.cancel(reason)
        }
        finally {
          this.dispose()
        }
      },
      pull: async (controller) => {
        try {
          const chunk = await reader.read()
          if (chunk.done) {
            controller.close()
            this.dispose()
            return
          }
          controller.enqueue(chunk.value)
        }
        catch {
          controller.error(this.readError())
          this.dispose()
        }
      },
    })
  }

  private readError(): HttpTransportError {
    if (this.timedOut()) {
      return new HttpTransportError("request_timeout", "The request timed out", true)
    }
    if (this.signal.aborted) {
      return new HttpTransportError("request_aborted", "The request was aborted")
    }
    return new HttpTransportError("network_error", "The response body could not be read", true)
  }
}

function createSignal(
  externalSignal: AbortSignal | undefined,
  timeoutMs: number | undefined,
): { cleanup: () => void, signal: AbortSignal, timedOut: () => boolean } {
  const controller = new AbortController()
  let timeoutTriggered = false
  const onAbort = () => controller.abort(externalSignal?.reason)
  externalSignal?.addEventListener("abort", onAbort, { once: true })

  let timer: ReturnType<typeof setTimeout> | undefined
  if (timeoutMs !== undefined && timeoutMs > 0) {
    timer = setTimeout(() => {
      timeoutTriggered = true
      controller.abort(new Error("Request timed out"))
    }, timeoutMs)
  }

  if (externalSignal?.aborted) {
    onAbort()
  }

  return {
    cleanup: () => {
      externalSignal?.removeEventListener("abort", onAbort)
      if (timer !== undefined) {
        clearTimeout(timer)
      }
    },
    signal: controller.signal,
    timedOut: () => timeoutTriggered,
  }
}

function redirectMethod(
  status: number,
  method: HttpMethod,
): { method: HttpMethod, removeBody: boolean } {
  if (status === 303 && method !== "HEAD") {
    return { method: "GET", removeBody: true }
  }
  if ((status === 301 || status === 302) && method === "POST") {
    return { method: "GET", removeBody: true }
  }
  return { method, removeBody: false }
}

export class FetchHttpTransport implements HttpTransport {
  private readonly fetchImpl: FetchLike
  private readonly maxRedirects: number

  constructor(options: FetchHttpTransportOptions = {}) {
    if (!options.fetch && typeof globalThis.fetch !== "function") {
      throw new TypeError("A Fetch implementation is required")
    }
    this.fetchImpl = options.fetch ?? globalThis.fetch.bind(globalThis)
    this.maxRedirects = Math.min(10, Math.max(0, options.maxRedirects ?? 5))
  }

  async send(request: HttpTransportRequest): Promise<HttpTransportResponse> {
    const scopedSignal = createSignal(request.signal, request.timeoutMs)
    try {
      return await this.sendWithRedirects(
        request,
        scopedSignal.signal,
        scopedSignal.cleanup,
        scopedSignal.timedOut,
      )
    }
    catch (error) {
      if (error instanceof HttpTransportError) {
        scopedSignal.cleanup()
        throw error
      }
      if (scopedSignal.timedOut()) {
        scopedSignal.cleanup()
        throw new HttpTransportError("request_timeout", "The request timed out", true)
      }
      if (request.signal?.aborted || scopedSignal.signal.aborted) {
        scopedSignal.cleanup()
        throw new HttpTransportError("request_aborted", "The request was aborted")
      }
      scopedSignal.cleanup()
      throw new HttpTransportError("network_error", "The request could not reach the server", true)
    }
  }

  private async sendWithRedirects(
    request: HttpTransportRequest,
    signal: AbortSignal,
    cleanup: () => void,
    timedOut: () => boolean,
  ): Promise<HttpTransportResponse> {
    let currentUrl = new URL(request.url)
    let currentMethod = request.method
    let currentBody = request.body ?? null
    const headers = new Headers(request.headers)

    for (let redirectCount = 0; ; redirectCount += 1) {
      const response = await this.fetchImpl(currentUrl, {
        body: currentMethod === "GET" || currentMethod === "HEAD" ? null : currentBody,
        headers,
        method: currentMethod,
        redirect: "manual",
        signal,
      })

      if (!REDIRECT_STATUSES.has(response.status)) {
        return new FetchHttpResponse(response, cleanup, signal, timedOut)
      }

      const location = response.headers.get("location")
      if (!location) {
        return new FetchHttpResponse(response, cleanup, signal, timedOut)
      }
      if (redirectCount >= this.maxRedirects) {
        response.body?.cancel().catch(() => undefined)
        throw new HttpTransportError(
          "redirect_limit_exceeded",
          "The response exceeded the redirect limit",
        )
      }

      const nextUrl = new URL(location, currentUrl)
      if (nextUrl.username || nextUrl.password) {
        throw new HttpTransportError("unsafe_redirect", "The redirect URL contains credentials")
      }
      if (currentUrl.protocol === "https:" && nextUrl.protocol !== "https:") {
        throw new HttpTransportError(
          "unsafe_redirect",
          "Refusing to redirect from HTTPS to a less secure protocol",
        )
      }
      if (nextUrl.protocol !== "http:" && nextUrl.protocol !== "https:") {
        throw new HttpTransportError("unsafe_redirect", "The redirect protocol is not allowed")
      }

      if (nextUrl.origin !== currentUrl.origin) {
        for (const header of CROSS_ORIGIN_SENSITIVE_HEADERS) {
          headers.delete(header)
        }
      }

      const rewrite = redirectMethod(response.status, currentMethod)
      currentMethod = rewrite.method
      if (rewrite.removeBody) {
        currentBody = null
        for (const header of CONTENT_HEADERS) {
          headers.delete(header)
        }
      }
      response.body?.cancel().catch(() => undefined)
      currentUrl = nextUrl
    }
  }
}
