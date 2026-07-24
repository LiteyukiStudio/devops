import type { ConfigPort } from '../commands/types.js'
import type { OAuthCredential, StoredLunaConfig } from '../config/schema.js'
import type { StoreOAuthCredentialInput } from './types.js'
import { CliCommandError } from '../commands/errors.js'
import {
  ensureInstance,
  normalizeContextName,
  normalizeServerOrigin,
  pruneUnreferencedContextResources,
  upsertContext,
} from '../config/context.js'
import { updateConfig } from '../config/store.js'
import {
  assertIsoDate,
  normalizeCredentialName,
  normalizeScopes,
} from './validation.js'

export async function storeValidatedOAuthCredential(
  store: ConfigPort,
  input: StoreOAuthCredentialInput,
): Promise<StoredLunaConfig> {
  const accessToken = input.accessToken.trim()
  const refreshToken = input.refreshToken?.trim() || undefined
  if (!accessToken) {
    throw new CliCommandError(
      'oauth_access_token_required',
      'A validated OAuth access token is required.',
      { status: 422 },
    )
  }
  assertIsoDate(input.expiresAt)
  const contextName = normalizeContextName(input.context)

  return updateConfig(store, (config) => {
    const origin = normalizeServerOrigin(input.server)
    ensureInstance(config, origin)
    const previousContext = config.contexts[contextName]
    const previousCredential = previousContext?.credential
    const credentialName = input.credential
      ? normalizeCredentialName(input.credential)
      : reusableCredentialName(config, contextName, previousCredential)
    const credential: OAuthCredential = {
      type: 'oauth',
      accessToken,
      refreshToken,
      tokenType: input.tokenType?.trim() || undefined,
      scopes: normalizeScopes(input.scopes),
      user: input.user,
      expiresAt: input.expiresAt,
      createdAt: new Date().toISOString(),
    }

    config.credentials[credentialName] = credential
    config.contexts[contextName] = upsertContext(config, {
      name: contextName,
      server: origin,
      credential: credentialName,
      project: input.project,
    })
    if (input.makeCurrent ?? !config.currentContext) {
      config.currentContext = contextName
    }
    pruneUnreferencedContextResources(config, {
      credential: previousCredential === credentialName ? undefined : previousCredential,
      instance: previousContext?.instance,
    })
  })
}

function reusableCredentialName(
  config: StoredLunaConfig,
  contextName: string,
  previousCredential: string | undefined,
): string {
  if (previousCredential && config.credentials[previousCredential]?.type === 'oauth') {
    return previousCredential
  }

  const base = normalizeCredentialName(`${contextName}-oauth`)
  if (!Object.hasOwn(config.credentials, base))
    return base
  let suffix = 2
  while (Object.hasOwn(config.credentials, `${base}-${suffix}`)) suffix += 1
  return `${base}-${suffix}`
}
