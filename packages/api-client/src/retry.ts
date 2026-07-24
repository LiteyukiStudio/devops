import { HttpTransportError } from "./errors.js"
import type { HttpMethod, RetryPolicy } from "./types.js"

const SAFE_METHODS = new Set<HttpMethod>(["GET", "HEAD"])
const RETRYABLE_STATUSES = new Set([408, 425, 429, 502, 503, 504])

export interface NormalizedRetryPolicy {
  baseDelayMs: number
  jitterRatio: number
  maxAttempts: number
  maxDelayMs: number
}

export function normalizeRetryPolicy(policy?: RetryPolicy | false): NormalizedRetryPolicy | false {
  if (policy === false) {
    return false
  }
  return {
    baseDelayMs: Math.max(0, policy?.baseDelayMs ?? 200),
    jitterRatio: Math.min(1, Math.max(0, policy?.jitterRatio ?? 0.2)),
    maxAttempts: Math.min(5, Math.max(1, policy?.maxAttempts ?? 3)),
    maxDelayMs: Math.max(0, policy?.maxDelayMs ?? 2_000),
  }
}

export function isSafeMethod(method: HttpMethod): boolean {
  return SAFE_METHODS.has(method)
}

export function isRetryableStatus(status: number): boolean {
  return RETRYABLE_STATUSES.has(status)
}

export function parseRetryAfter(value: string | null, now = Date.now()): number | undefined {
  if (!value) {
    return undefined
  }
  const seconds = Number(value)
  if (Number.isFinite(seconds) && seconds >= 0) {
    return Math.ceil(seconds * 1_000)
  }
  const timestamp = Date.parse(value)
  if (Number.isNaN(timestamp)) {
    return undefined
  }
  return Math.max(0, timestamp - now)
}

export function retryDelay(
  attempt: number,
  policy: NormalizedRetryPolicy,
  retryAfterMs?: number,
  random = Math.random,
): number {
  if (retryAfterMs !== undefined) {
    return Math.min(policy.maxDelayMs, Math.max(0, retryAfterMs))
  }
  const exponential = Math.min(
    policy.maxDelayMs,
    policy.baseDelayMs * 2 ** Math.max(0, attempt - 1),
  )
  const jitter = exponential * policy.jitterRatio * (random() * 2 - 1)
  return Math.max(0, Math.round(exponential + jitter))
}

export async function abortableSleep(ms: number, signal?: AbortSignal): Promise<void> {
  if (ms <= 0) {
    return
  }
  if (signal?.aborted) {
    throw new HttpTransportError("request_aborted", "The request was aborted")
  }
  await new Promise<void>((resolve, reject) => {
    let settled = false
    const finish = (callback: () => void) => {
      if (settled) {
        return
      }
      settled = true
      signal?.removeEventListener("abort", onAbort)
      callback()
    }
    const timer = setTimeout(() => finish(resolve), ms)
    const onAbort = () => {
      clearTimeout(timer)
      finish(() => reject(
        new HttpTransportError("request_aborted", "The request was aborted"),
      ))
    }
    signal?.addEventListener("abort", onAbort, { once: true })
    if (signal?.aborted) {
      onAbort()
    }
  })
}
