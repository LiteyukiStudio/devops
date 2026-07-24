import { normalizeRequestBody } from "./body.js"
import { HttpTransportError } from "./errors.js"
import { FetchHttpTransport } from "./fetch-transport.js"
import { createRequestId } from "./request-id.js"
import {
  parseFailure,
  parseSuccess,
  transportFailure,
} from "./response.js"
import {
  abortableSleep,
  isRetryableStatus,
  isSafeMethod,
  normalizeRetryPolicy,
  parseRetryAfter,
  retryDelay,
} from "./retry.js"
import { parseSseStream } from "./sse.js"
import type {
  HttpMethod,
  HttpTransport,
  HttpTransportRequest,
  HttpTransportResponse,
  LunaClientOptions,
  LunaRequestOptions,
  LunaResult,
  LunaSseEvent,
  LunaSseOptions,
} from "./types.js"
import { normalizeBaseUrl, resolveRequestUrl, sameOrigin } from "./url.js"

const DEFAULT_TIMEOUT_MS = 30_000

function normalizeMethod(method?: HttpMethod): HttpMethod {
  return method ?? "GET"
}

function isValidBearerToken(token: string): boolean {
  return token.length > 0 && !/[\r\n]/.test(token)
}

export class LunaClient {
  private readonly allowCrossOriginRequests: boolean
  private readonly baseUrl: URL
  private readonly requestIdFactory: () => string
  private readonly retryPolicy: ReturnType<typeof normalizeRetryPolicy>
  private readonly timeoutMs: number
  private readonly tokenProvider: LunaClientOptions["tokenProvider"]
  private readonly transport: HttpTransport

  constructor(options: LunaClientOptions) {
    this.baseUrl = normalizeBaseUrl(options.baseUrl)
    this.allowCrossOriginRequests = options.allowCrossOriginRequests ?? false
    this.requestIdFactory = options.requestIdFactory ?? createRequestId
    this.retryPolicy = normalizeRetryPolicy(options.retry)
    this.timeoutMs = Math.max(1, options.timeoutMs ?? DEFAULT_TIMEOUT_MS)
    this.tokenProvider = options.tokenProvider
    this.transport = options.transport ?? new FetchHttpTransport()
  }

  async request<T = unknown>(options: LunaRequestOptions): Promise<LunaResult<T>> {
    const requestId = options.requestId ?? this.requestIdFactory()
    let request: HttpTransportRequest
    try {
      request = await this.prepareRequest(options, requestId)
    }
    catch (error) {
      const normalized = error instanceof HttpTransportError
        ? error
        : new HttpTransportError(
            "invalid_request",
            error instanceof Error ? error.message : "The request is invalid",
          )
      return transportFailure(normalized, requestId)
    }

    const attempts = this.retryPolicy && isSafeMethod(request.method)
      ? this.retryPolicy.maxAttempts
      : 1

    for (let attempt = 1; attempt <= attempts; attempt += 1) {
      try {
        const response = await this.transport.send(request)
        if (
          attempt < attempts
          && this.retryPolicy
          && isRetryableStatus(response.status)
        ) {
          const retryAfterMs = parseRetryAfter(response.headers.get("retry-after"))
          response.body?.cancel().catch(() => undefined)
          await abortableSleep(
            retryDelay(attempt, this.retryPolicy, retryAfterMs),
            options.signal,
          )
          continue
        }
        if (response.status >= 200 && response.status < 300) {
          return await parseSuccess<T>(
            response,
            options.responseType ?? "json",
            requestId,
          )
        }
        return await parseFailure(response, requestId)
      }
      catch (error) {
        const failure = transportFailure(error, requestId)
        if (
          attempt >= attempts
          || !this.retryPolicy
          || !failure.error.retryable
        ) {
          return failure
        }
        try {
          await abortableSleep(
            retryDelay(attempt, this.retryPolicy),
            options.signal,
          )
        }
        catch (sleepError) {
          return transportFailure(sleepError, requestId)
        }
      }
    }

    return transportFailure(undefined, requestId)
  }

  async openSse<T = unknown>(
    options: LunaSseOptions<T>,
  ): Promise<LunaResult<AsyncIterable<LunaSseEvent<T>>>> {
    const headers = new Headers(options.headers)
    if (!headers.has("accept")) {
      headers.set("accept", "text/event-stream")
    }
    const result = await this.request<HttpTransportResponse>({
      ...options,
      headers,
      responseType: "stream",
    })
    if (!result.ok) {
      return result
    }
    if (!result.data.body) {
      return {
        error: {
          code: "invalid_response",
          details: {},
          message: "The SSE response did not contain a body",
          requestId: result.requestId,
          retryable: false,
          status: result.status,
        },
        headers: result.headers,
        ok: false,
        requestId: result.requestId,
        status: result.status,
      }
    }
    return {
      ...result,
      data: parseSseStream(result.data.body, { parseData: options.parseData }),
    }
  }

  private async prepareRequest(
    options: LunaRequestOptions,
    requestId: string,
  ): Promise<HttpTransportRequest> {
    const method = normalizeMethod(options.method)
    const url = resolveRequestUrl(this.baseUrl, options.path, options.query)
    if (!this.allowCrossOriginRequests && !sameOrigin(url, this.baseUrl)) {
      throw new HttpTransportError(
        "invalid_request",
        "Cross-origin API requests are disabled",
      )
    }

    const normalized = normalizeRequestBody(options.body, options.headers)
    normalized.headers.set("accept", normalized.headers.get("accept") ?? "application/json")
    normalized.headers.set("x-request-id", requestId)

    if ((options.auth ?? true) && sameOrigin(url, this.baseUrl) && this.tokenProvider) {
      let token: string | undefined
      try {
        token = await this.tokenProvider.getAccessToken()
      }
      catch {
        throw new HttpTransportError(
          "invalid_request",
          "The access token provider failed",
        )
      }
      if (token !== undefined) {
        if (!isValidBearerToken(token)) {
          throw new HttpTransportError("invalid_request", "The access token is invalid")
        }
        normalized.headers.set("authorization", `Bearer ${token}`)
      }
    }

    return {
      body: normalized.body,
      headers: normalized.headers,
      method,
      signal: options.signal,
      timeoutMs: options.timeoutMs ?? this.timeoutMs,
      url,
    }
  }
}
