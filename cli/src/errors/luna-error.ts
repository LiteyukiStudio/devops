import { redactValue, sanitizeTerminalText } from './sanitize.js'

export const EXIT_CODE = Object.freeze({
  success: 0,
  unknown: 1,
  invalidInput: 2,
  unauthenticated: 3,
  forbidden: 4,
  notFound: 5,
  conflict: 6,
  retryLater: 7,
  serviceFailure: 8,
  partialSuccess: 9,
} as const)

export type ExitCode = (typeof EXIT_CODE)[keyof typeof EXIT_CODE]

export interface LunaFieldError {
  readonly key: string
  readonly code: string
  readonly expected?: unknown
  readonly actual?: unknown
  readonly message?: string
}

export interface LunaErrorOptions {
  readonly status?: number
  readonly exitCode?: ExitCode
  readonly retryable?: boolean
  readonly requestId?: string
  readonly retryAfter?: number
  readonly purpose?: string
  readonly fields?: readonly LunaFieldError[] | Readonly<Record<string, string>>
  readonly details?: Readonly<Record<string, unknown>>
  readonly cause?: unknown
}

export interface LunaErrorDocument {
  readonly error: {
    readonly code: string
    readonly message: string
    readonly status: number
    readonly requestId?: string
    readonly retryable: boolean
    readonly retryAfter?: number
    readonly purpose?: string
    readonly fields?: readonly LunaFieldError[] | Readonly<Record<string, string>>
    readonly details: Readonly<Record<string, unknown>>
  }
}

const CODE_EXIT_MAP: Readonly<Record<string, ExitCode>> = {
  authentication_failed: EXIT_CODE.unauthenticated,
  token_expired: EXIT_CODE.unauthenticated,
  token_refresh_failed: EXIT_CODE.unauthenticated,
  unauthenticated: EXIT_CODE.unauthenticated,
  forbidden: EXIT_CODE.forbidden,
  mfa_required: EXIT_CODE.forbidden,
  permission_denied: EXIT_CODE.forbidden,
  not_found: EXIT_CODE.notFound,
  resource_not_found: EXIT_CODE.notFound,
  conflict: EXIT_CODE.conflict,
  precondition_failed: EXIT_CODE.conflict,
  resource_version_conflict: EXIT_CODE.conflict,
  state_conflict: EXIT_CODE.conflict,
  rate_limited: EXIT_CODE.retryLater,
  retry_later: EXIT_CODE.retryLater,
  dependency_unavailable: EXIT_CODE.serviceFailure,
  network_error: EXIT_CODE.serviceFailure,
  server_error: EXIT_CODE.serviceFailure,
  service_unavailable: EXIT_CODE.serviceFailure,
  partial_success: EXIT_CODE.partialSuccess,
}

export class LunaError extends Error {
  readonly code: string
  readonly status: number
  readonly exitCode: ExitCode
  readonly retryable: boolean
  readonly requestId?: string
  readonly retryAfter?: number
  readonly purpose?: string
  readonly fields?: readonly LunaFieldError[] | Readonly<Record<string, string>>
  readonly details: Readonly<Record<string, unknown>>

  constructor(code: string, message: string, options: LunaErrorOptions = {}) {
    super(sanitizeTerminalText(message), { cause: options.cause })
    this.name = 'LunaError'
    this.code = sanitizeTerminalText(code)
    this.status = options.status ?? 400
    this.exitCode = options.exitCode ?? exitCodeFor(code, this.status)
    this.retryable = options.retryable ?? false
    this.requestId = options.requestId ? sanitizeTerminalText(options.requestId) : undefined
    this.retryAfter = options.retryAfter
    this.purpose = options.purpose ? sanitizeTerminalText(options.purpose) : undefined
    this.fields = options.fields
      ? redactValue(options.fields) as LunaError['fields']
      : undefined
    this.details = asRecord(redactValue(options.details ?? {}))
  }
}

export function invalidInput(
  code: string,
  message: string,
  options: Omit<LunaErrorOptions, 'exitCode' | 'status'> = {},
): LunaError {
  return new LunaError(code, message, {
    ...options,
    status: 400,
    exitCode: EXIT_CODE.invalidInput,
  })
}

export function exitCodeFor(code: string, status: number): ExitCode {
  const mapped = CODE_EXIT_MAP[code]
  if (mapped !== undefined)
    return mapped
  if (status === 401)
    return EXIT_CODE.unauthenticated
  if (status === 403)
    return EXIT_CODE.forbidden
  if (status === 404)
    return EXIT_CODE.notFound
  if (status === 409 || status === 412 || status === 422)
    return EXIT_CODE.conflict
  if (status === 429)
    return EXIT_CODE.retryLater
  if (status >= 500)
    return EXIT_CODE.serviceFailure
  if (status >= 400)
    return EXIT_CODE.invalidInput
  return EXIT_CODE.unknown
}

export function normalizeLunaError(error: unknown): LunaError {
  if (error instanceof LunaError)
    return error

  if (isRecord(error)) {
    const nested = isRecord(error.error) ? error.error : error
    const status = finiteNumber(nested.status) ?? finiteNumber(nested.statusCode) ?? 500
    const code = nonEmptyString(nested.code) ?? 'internal_error'
    const message
      = nonEmptyString(nested.message)
        ?? nonEmptyString(nested.detail)
        ?? 'The command failed.'
    return new LunaError(code, message, {
      status,
      retryable: nested.retryable === true,
      requestId: nonEmptyString(nested.requestId),
      retryAfter: finiteNumber(nested.retryAfter),
      purpose: nonEmptyString(nested.purpose),
      fields: normalizeFields(nested.fields),
      details: isRecord(nested.details) ? nested.details : {},
      cause: error,
    })
  }

  if (error instanceof Error) {
    return new LunaError('internal_error', error.message || 'The command failed.', {
      status: 500,
      cause: error,
    })
  }

  return new LunaError('internal_error', 'The command failed.', {
    status: 500,
    cause: error,
  })
}

export function toErrorDocument(error: unknown): LunaErrorDocument {
  const normalized = normalizeLunaError(error)
  const document: LunaErrorDocument = {
    error: {
      code: normalized.code,
      message: sanitizeTerminalText(normalized.message),
      status: normalized.status,
      retryable: normalized.retryable,
      details: asRecord(redactValue(normalized.details)),
      ...(normalized.requestId ? { requestId: normalized.requestId } : {}),
      ...(normalized.retryAfter !== undefined ? { retryAfter: normalized.retryAfter } : {}),
      ...(normalized.purpose ? { purpose: normalized.purpose } : {}),
      ...(normalized.fields
        ? { fields: redactValue(normalized.fields) as LunaErrorDocument['error']['fields'] }
        : {}),
    },
  }
  return document
}

function normalizeFields(
  value: unknown,
): readonly LunaFieldError[] | Readonly<Record<string, string>> | undefined {
  if (Array.isArray(value)) {
    return value
      .filter(isRecord)
      .map(field => ({
        key: nonEmptyString(field.key) ?? '',
        code: nonEmptyString(field.code) ?? 'invalid',
        ...(field.expected !== undefined ? { expected: field.expected } : {}),
        ...(field.actual !== undefined ? { actual: field.actual } : {}),
        ...(nonEmptyString(field.message) ? { message: nonEmptyString(field.message) } : {}),
      }))
  }
  if (!isRecord(value))
    return undefined
  return Object.fromEntries(
    Object.entries(value)
      .filter((entry): entry is [string, string] => typeof entry[1] === 'string'),
  )
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function asRecord(value: unknown): Readonly<Record<string, unknown>> {
  return isRecord(value) ? value : {}
}

function nonEmptyString(value: unknown): string | undefined {
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

function finiteNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}
