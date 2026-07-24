import { HttpTransportError } from "./errors.js"
import { parseRetryAfter } from "./retry.js"
import type {
  HttpTransportResponse,
  LunaError,
  LunaFailure,
  LunaResponseType,
  LunaSuccess,
} from "./types.js"

const SENSITIVE_KEY = /(?:authorization|cookie|credential|password|recovery.?code|secret|token)/i

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
}

function redactSensitive(value: unknown, depth = 0): unknown {
  if (depth > 8) {
    return "[REDACTED]"
  }
  if (Array.isArray(value)) {
    return value.map(item => redactSensitive(item, depth + 1))
  }
  if (!isRecord(value)) {
    return value
  }
  const result: Record<string, unknown> = {}
  for (const [key, item] of Object.entries(value)) {
    result[key] = SENSITIVE_KEY.test(key) ? "[REDACTED]" : redactSensitive(item, depth + 1)
  }
  return result
}

async function parseJsonOrUndefined(response: HttpTransportResponse): Promise<unknown> {
  const text = await response.text()
  if (text.trim() === "") {
    return undefined
  }
  return JSON.parse(text) as unknown
}

export function responseRequestId(response: HttpTransportResponse, fallback: string): string {
  return response.headers.get("x-request-id")
    ?? response.headers.get("x-correlation-id")
    ?? fallback
}

export async function parseSuccess<T>(
  response: HttpTransportResponse,
  responseType: LunaResponseType,
  requestId: string,
): Promise<LunaSuccess<T>> {
  let data: unknown
  if (response.status === 204 || response.status === 205) {
    data = undefined
    response.dispose()
  }
  else if (responseType === "text") {
    data = await response.text()
  }
  else if (responseType === "binary") {
    data = new Uint8Array(await response.arrayBuffer())
  }
  else if (responseType === "stream") {
    data = response
  }
  else {
    try {
      data = await parseJsonOrUndefined(response)
    }
    catch {
      throw new HttpTransportError(
        "invalid_response",
        "The server returned invalid JSON",
      )
    }
  }

  return {
    data: data as T,
    headers: response.headers,
    ok: true,
    requestId: responseRequestId(response, requestId),
    status: response.status,
  }
}

function normalizeFields(value: unknown): Readonly<Record<string, string>> | undefined {
  if (!isRecord(value)) {
    return undefined
  }
  const fields: Record<string, string> = {}
  for (const [key, reason] of Object.entries(value)) {
    if (typeof reason === "string") {
      fields[key] = reason
    }
  }
  return Object.keys(fields).length > 0 ? fields : undefined
}

export async function parseFailure(
  response: HttpTransportResponse,
  fallbackRequestId: string,
): Promise<LunaFailure> {
  let payload: unknown
  try {
    payload = await parseJsonOrUndefined(response)
  }
  catch {
    payload = undefined
  }

  const envelope = isRecord(payload) && isRecord(payload.error)
    ? payload.error
    : isRecord(payload)
      ? payload
      : {}
  const oauthCode = typeof envelope.error === "string" ? envelope.error : undefined
  const requestId = typeof envelope.requestId === "string"
    ? envelope.requestId
    : responseRequestId(response, fallbackRequestId)
  const retryAfterMs = parseRetryAfter(response.headers.get("retry-after"))
    ?? (typeof envelope.retryAfter === "number" ? envelope.retryAfter * 1_000 : undefined)
  const code = typeof envelope.code === "string"
    ? envelope.code
    : oauthCode ?? `http_${response.status}`
  const diagnostic = typeof envelope.message === "string"
    ? envelope.message
    : typeof envelope.detail === "string"
      ? envelope.detail
      : typeof envelope.error_description === "string"
        ? envelope.error_description
        : `The server returned HTTP ${response.status}`
  const details = isRecord(envelope.details)
    ? redactSensitive(envelope.details)
    : {}
  const error: LunaError = {
    code,
    details: isRecord(details) ? details : {},
    fields: normalizeFields(envelope.fields),
    message: diagnostic,
    purpose: typeof envelope.purpose === "string" ? envelope.purpose : undefined,
    requestId,
    retryAfterMs,
    retryable: response.status === 408
      || response.status === 425
      || response.status === 429
      || response.status === 502
      || response.status === 503
      || response.status === 504,
    status: response.status,
  }
  return {
    error,
    headers: response.headers,
    ok: false,
    requestId,
    status: response.status,
  }
}

export function transportFailure(error: unknown, requestId: string): LunaFailure {
  const transportError = error instanceof HttpTransportError
    ? error
    : new HttpTransportError("network_error", "The request could not reach the server", true)
  return {
    error: {
      code: transportError.code,
      details: {},
      message: transportError.message,
      requestId,
      retryable: transportError.retryable,
      status: 0,
    },
    headers: new Headers(),
    ok: false,
    requestId,
    status: 0,
  }
}
