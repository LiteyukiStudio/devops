import { CliCommandError } from '../commands/errors.js'

export function normalizeCredentialName(name: string): string {
  const normalized = name.trim()
  if (!/^[A-Z0-9][\w.-]{0,95}$/i.test(normalized)) {
    throw new CliCommandError(
      'credential_name_invalid',
      'Credential names must be 1-96 characters using letters, numbers, dot, underscore, or hyphen.',
      { status: 422 },
    )
  }
  return normalized
}

export function normalizeScopes(scopes: readonly string[] | undefined): string[] {
  const normalized = [...new Set(
    (scopes ?? []).map(scope => scope.trim()).filter(Boolean),
  )].sort()
  for (const scope of normalized) {
    if (/\s/.test(scope)) {
      throw new CliCommandError(
        'credential_scope_invalid',
        `Credential scope "${scope}" must not contain whitespace.`,
        { status: 422 },
      )
    }
  }
  return normalized
}

export function assertIsoDate(value: string | undefined): void {
  if (value === undefined)
    return
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp) || new Date(timestamp).toISOString() !== value) {
    throw new CliCommandError(
      'credential_expiry_invalid',
      'Credential expiration must be an ISO 8601 UTC timestamp.',
      { status: 422 },
    )
  }
}
