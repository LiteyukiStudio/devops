import type { ConfigPort } from '../commands/types.js'
import type { LunaCredential } from '../config/schema.js'
import type { AuthStatusEntry } from './types.js'
import { CliCommandError } from '../commands/errors.js'
import { normalizeServerOrigin } from '../config/context.js'
import {

  parseConfigDocument,
} from '../config/schema.js'
import { accessTokenFromEnvironment } from './access-token.js'

export interface AuthStatusOptions {
  readonly context?: string
  readonly all?: boolean
  readonly now?: Date
  readonly env?: Readonly<Record<string, string | undefined>>
}

export async function getAuthStatus(
  store: ConfigPort,
  options: AuthStatusOptions = {},
): Promise<readonly AuthStatusEntry[]> {
  const config = parseConfigDocument(await store.read())
  const environmentCredential = accessTokenFromEnvironment(options.env)
  const names = options.all
    ? Object.keys(config.contexts).sort()
    : [options.context ?? config.currentContext].filter(
        (name): name is string => Boolean(name),
      )
  if (names.length === 0)
    return []

  return names.map((name) => {
    const context = config.contexts[name]
    if (!context) {
      throw new CliCommandError(
        'context_not_found',
        `Context "${name}" does not exist.`,
        { status: 404 },
      )
    }
    const instance = config.instances[context.instance]
    const storedCredential = context.credential
      ? config.credentials[context.credential]
      : undefined
    const credential = environmentCredential ?? storedCredential
    const source = environmentCredential ? 'environment' : 'stored'
    return {
      context: name,
      current: config.currentContext === name,
      server: normalizeServerOrigin(instance.server),
      authenticated: credential !== undefined && !isExpired(credential, options.now),
      credential: credential && context.credential
        ? {
            name: environmentCredential ? 'LUNA_TOKEN' : context.credential,
            type: credential.type,
            scopes: [...credential.scopes],
            user: credential.user,
            expiresAt: credential.expiresAt,
            expired: isExpired(credential, options.now),
            source,
          }
        : credential
          ? {
              name: 'LUNA_TOKEN',
              type: credential.type,
              scopes: [...credential.scopes],
              user: credential.user,
              expiresAt: credential.expiresAt,
              expired: false,
              source,
            }
          : undefined,
    }
  })
}

function isExpired(credential: LunaCredential, now = new Date()): boolean {
  return credential.expiresAt !== undefined
    && Date.parse(credential.expiresAt) <= now.getTime()
}
