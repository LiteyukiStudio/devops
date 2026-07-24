import type { LunaError } from "./types.js"

export type HttpTransportErrorCode =
  | "invalid_request"
  | "invalid_response"
  | "request_aborted"
  | "request_timeout"
  | "network_error"
  | "redirect_limit_exceeded"
  | "unsafe_redirect"

export class HttpTransportError extends Error {
  readonly code: HttpTransportErrorCode
  readonly retryable: boolean

  constructor(code: HttpTransportErrorCode, message: string, retryable = false) {
    super(message)
    this.name = "HttpTransportError"
    this.code = code
    this.retryable = retryable
  }
}

export class LunaRequestError extends Error {
  readonly error: LunaError

  constructor(error: LunaError) {
    super(error.message)
    this.name = "LunaRequestError"
    this.error = error
  }
}
