import type { ConfigPort } from '../commands/types.js'
import type { AccessTokenCredential, StoredLunaConfig } from '../config/schema.js'
import type { StoreAccessTokenInput } from './types.js'
import process from 'node:process'
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

export async function storeValidatedAccessToken(
  store: ConfigPort,
  input: StoreAccessTokenInput,
): Promise<StoredLunaConfig> {
  const token = input.token.trim()
  if (!token) {
    throw new CliCommandError(
      'access_token_required',
      'A validated access token is required.',
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
    const credential: AccessTokenCredential = {
      type: 'access_token',
      token,
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

export function accessTokenFromEnvironment(
  env: Readonly<Record<string, string | undefined>> = process.env,
): AccessTokenCredential | undefined {
  const token = env.LUNA_TOKEN?.trim()
  if (!token)
    return undefined
  return {
    type: 'access_token',
    token,
    scopes: [],
  }
}

function reusableCredentialName(
  config: StoredLunaConfig,
  contextName: string,
  previousCredential: string | undefined,
): string {
  if (
    previousCredential
    && config.credentials[previousCredential]?.type === 'access_token'
  ) {
    return previousCredential
  }

  const base = normalizeCredentialName(`${contextName}-access-token`)
  if (!Object.hasOwn(config.credentials, base))
    return base
  let suffix = 2
  while (Object.hasOwn(config.credentials, `${base}-${suffix}`)) suffix += 1
  return `${base}-${suffix}`
}
