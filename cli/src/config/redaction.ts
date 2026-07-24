import type { LunaConfigDocument } from '../commands/types.js'

const REDACTED = '[REDACTED]'
const SENSITIVE_KEY = /authorization|cookie|password|secret|token|api[-_]?key/i
const SENSITIVE_QUERY_KEY
  = /^(?:access_token|refresh_token|id_token|token|code|state|client_secret|secret|password)$/i

export function redactConfig(config: LunaConfigDocument): LunaConfigDocument {
  return redactValue(config) as LunaConfigDocument
}

export function redactValue(value: unknown): unknown {
  if (Array.isArray(value))
    return value.map(item => redactValue(item))
  if (!isRecord(value))
    return value

  const result: Record<string, unknown> = {}
  for (const [key, item] of Object.entries(value)) {
    if (SENSITIVE_KEY.test(key)) {
      result[key] = REDACTED
      continue
    }
    result[key] = redactValue(item)
  }
  return result
}

export function redactUrl(value: string): string {
  try {
    const url = new URL(value)
    for (const key of [...url.searchParams.keys()]) {
      if (SENSITIVE_QUERY_KEY.test(key))
        url.searchParams.set(key, REDACTED)
    }
    if (url.username)
      url.username = REDACTED
    if (url.password)
      url.password = REDACTED
    return url.toString()
  }
  catch {
    return value
  }
}

export function redactText(value: string, secrets: readonly string[] = []): string {
  return secrets
    .filter(secret => secret.length > 0)
    .sort((left, right) => right.length - left.length)
    .reduce((text, secret) => text.replaceAll(secret, REDACTED), value)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
