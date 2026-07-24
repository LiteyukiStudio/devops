export interface StableCliError {
  readonly error: {
    readonly code: string
    readonly message: string
    readonly status: number
    readonly retryable: boolean
    readonly details: Readonly<Record<string, unknown>>
  }
}

export class CliCommandError extends Error {
  readonly code: string
  readonly status: number
  readonly exitCode: number
  readonly retryable: boolean
  readonly details: Readonly<Record<string, unknown>>

  constructor(
    code: string,
    message: string,
    options: {
      status?: number
      exitCode?: number
      retryable?: boolean
      details?: Readonly<Record<string, unknown>>
      cause?: unknown
    } = {},
  ) {
    super(message, { cause: options.cause })
    this.name = 'CliCommandError'
    this.code = code
    this.status = options.status ?? 400
    this.exitCode = options.exitCode ?? exitCodeForStatus(this.status)
    this.retryable = options.retryable ?? false
    this.details = options.details ?? {}
  }
}

export function exitCodeForStatus(status: number): number {
  if (status === 401)
    return 3
  if (status === 403)
    return 4
  if (status === 404)
    return 5
  if (status === 409 || status === 412 || status === 422)
    return 6
  if (status === 429)
    return 7
  if (status >= 500)
    return 8
  if (status >= 400)
    return 2
  return 1
}

export function toCliCommandError(error: unknown): CliCommandError {
  if (error instanceof CliCommandError)
    return error

  if (isErrorLike(error)) {
    const status = numberValue(error.status) ?? numberValue(error.statusCode) ?? 500
    const code = stringValue(error.code) ?? 'internal_error'
    return new CliCommandError(code, stringValue(error.message) ?? 'Command failed.', {
      status,
      retryable: Boolean(error.retryable),
      details: recordValue(error.details),
      cause: error,
    })
  }

  return new CliCommandError('internal_error', 'Command failed.', {
    status: 500,
    cause: error,
  })
}

export function stableError(error: unknown): StableCliError {
  const normalized = toCliCommandError(error)
  return {
    error: {
      code: normalized.code,
      message: normalized.message,
      status: normalized.status,
      retryable: normalized.retryable,
      details: normalized.details,
    },
  }
}

function isErrorLike(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function numberValue(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function stringValue(value: unknown): string | undefined {
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

function recordValue(value: unknown): Readonly<Record<string, unknown>> {
  return isErrorLike(value) ? value : {}
}
