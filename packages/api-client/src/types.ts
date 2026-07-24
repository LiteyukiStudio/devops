export type HttpMethod =
  | "DELETE"
  | "GET"
  | "HEAD"
  | "OPTIONS"
  | "PATCH"
  | "POST"
  | "PUT"

export type QueryPrimitive = boolean | Date | number | string
export type QueryValue = QueryPrimitive | readonly QueryPrimitive[] | null | undefined
export type QueryInput = Readonly<Record<string, QueryValue>> | URLSearchParams

export type LunaResponseType = "binary" | "json" | "stream" | "text"

export interface HttpTransportRequest {
  body?: BodyInit | null
  headers?: HeadersInit
  method: HttpMethod
  signal?: AbortSignal
  timeoutMs?: number
  url: URL
}

export interface HttpTransportResponse {
  readonly body: ReadableStream<Uint8Array> | null
  readonly headers: Headers
  readonly status: number
  readonly statusText: string
  readonly url: string
  arrayBuffer(): Promise<ArrayBuffer>
  dispose(): void
  text(): Promise<string>
}

export interface HttpTransport {
  send(request: HttpTransportRequest): Promise<HttpTransportResponse>
}

export type MaybePromise<T> = Promise<T> | T

export interface BearerTokenProvider {
  getAccessToken(): MaybePromise<string | undefined>
}

export interface LunaRequestOptions {
  auth?: boolean
  body?: unknown
  headers?: HeadersInit
  method?: HttpMethod
  path: string | URL
  query?: QueryInput
  requestId?: string
  responseType?: LunaResponseType
  signal?: AbortSignal
  timeoutMs?: number
}

export interface LunaError {
  code: string
  details: Readonly<Record<string, unknown>>
  fields?: Readonly<Record<string, string>>
  message: string
  purpose?: string
  requestId: string
  retryAfterMs?: number
  retryable: boolean
  status: number
}

export interface LunaSuccess<T> {
  data: T
  headers: Headers
  ok: true
  requestId: string
  status: number
}

export interface LunaFailure {
  error: LunaError
  headers: Headers
  ok: false
  requestId: string
  status: number
}

export type LunaResult<T> = LunaFailure | LunaSuccess<T>

export interface RetryPolicy {
  baseDelayMs?: number
  jitterRatio?: number
  maxAttempts?: number
  maxDelayMs?: number
}

export interface LunaClientOptions {
  allowCrossOriginRequests?: boolean
  baseUrl: string | URL
  requestIdFactory?: () => string
  retry?: RetryPolicy | false
  timeoutMs?: number
  tokenProvider?: BearerTokenProvider
  transport?: HttpTransport
}

export interface LunaSseEvent<T = unknown> {
  data: T | string
  event?: string
  id?: string
  rawData: string
}

export interface LunaSseOptions<T> extends Omit<LunaRequestOptions, "responseType"> {
  parseData?: (rawData: string) => T
}
