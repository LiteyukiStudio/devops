import type {
  ConfigPort,
  LunaContext,
  OutputFormat,
  ProjectContextSnapshot,
} from '../commands/types.js'

import type { StoredLunaConfig } from './schema.js'
import { createHash } from 'node:crypto'
import { CliCommandError } from '../commands/errors.js'
import {
  cloneConfigDocument,
  parseConfigDocument,

} from './schema.js'
import { updateConfig } from './store.js'

export interface SetContextInput {
  readonly name: string
  readonly server?: string
  readonly credential?: string | null
  readonly project?: ProjectContextSnapshot | null
  readonly language?: string
  readonly output?: OutputFormat | ''
  readonly makeCurrent?: boolean
  readonly createOnly?: boolean
}

export interface DeleteContextOptions {
  readonly allowCurrent?: boolean
  readonly nextContext?: string | null
}

export interface ContextView {
  readonly name: string
  readonly current: boolean
  readonly context: StoredContext
  readonly instance: StoredInstance
  readonly credential?: {
    readonly name: string
    readonly type: 'oauth' | 'access_token'
    readonly scopes: readonly string[]
    readonly expiresAt?: string
    readonly user?: {
      readonly id: string
      readonly name?: string
    }
  }
}

export class ContextService {
  constructor(private readonly store: ConfigPort) {}

  async list(): Promise<ReadonlyArray<{ name: string, context: LunaContext }>> {
    const config = parseConfigDocument(await this.store.read())
    return Object.entries(config.contexts)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([name, context]) => ({ name, context }))
  }

  async current(): Promise<{ name: string, context: LunaContext } | null> {
    const config = parseConfigDocument(await this.store.read())
    if (!config.currentContext)
      return null
    return {
      name: config.currentContext,
      context: config.contexts[config.currentContext],
    }
  }

  async get(name: string): Promise<LunaContext> {
    const config = parseConfigDocument(await this.store.read())
    return requireContext(config, normalizeContextName(name))
  }

  async view(name: string): Promise<ContextView> {
    const config = parseConfigDocument(await this.store.read())
    const normalizedName = normalizeContextName(name)
    const context = requireContext(config, normalizedName)
    const credential = context.credential
      ? config.credentials[context.credential]
      : undefined

    return {
      name: normalizedName,
      current: config.currentContext === normalizedName,
      context,
      instance: config.instances[context.instance],
      credential: credential && context.credential
        ? {
            name: context.credential,
            type: credential.type,
            scopes: [...credential.scopes],
            expiresAt: credential.expiresAt,
            user: credential.user,
          }
        : undefined,
    }
  }

  async use(name: string): Promise<StoredLunaConfig> {
    const normalizedName = normalizeContextName(name)
    return updateConfig(this.store, (config) => {
      requireContext(config, normalizedName)
      config.currentContext = normalizedName
    })
  }

  async set(input: SetContextInput): Promise<StoredLunaConfig> {
    const name = normalizeContextName(input.name)
    return updateConfig(this.store, (config) => {
      const existing = config.contexts[name]
      if (input.createOnly && existing) {
        throw new CliCommandError(
          'context_already_exists',
          `Context "${name}" already exists.`,
          { status: 409 },
        )
      }

      const next = upsertContext(config, { ...input, name })
      config.contexts[name] = next
      if (input.makeCurrent || !config.currentContext)
        config.currentContext = name
      pruneUnreferencedContextResources(config, {
        credential: existing?.credential,
        instance: existing?.instance,
      })
    })
  }

  async rename(name: string, newName: string): Promise<StoredLunaConfig> {
    const oldKey = normalizeContextName(name)
    const newKey = normalizeContextName(newName)
    return updateConfig(this.store, (config) => {
      const context = requireContext(config, oldKey)
      if (Object.hasOwn(config.contexts, newKey)) {
        throw new CliCommandError(
          'context_already_exists',
          `Context "${newKey}" already exists.`,
          { status: 409 },
        )
      }
      config.contexts[newKey] = context
      delete config.contexts[oldKey]
      if (config.currentContext === oldKey)
        config.currentContext = newKey
    })
  }

  async delete(
    name: string,
    options: DeleteContextOptions = {},
  ): Promise<StoredLunaConfig> {
    const normalizedName = normalizeContextName(name)
    return updateConfig(this.store, (config) => {
      const context = requireContext(config, normalizedName)
      if (config.currentContext === normalizedName && !options.allowCurrent) {
        throw new CliCommandError(
          'current_context_delete_requires_confirmation',
          `Context "${normalizedName}" is current and requires explicit confirmation before deletion.`,
          { status: 409 },
        )
      }

      delete config.contexts[normalizedName]
      if (config.currentContext === normalizedName) {
        if (options.nextContext) {
          const nextContext = normalizeContextName(options.nextContext)
          requireContext(config, nextContext)
          config.currentContext = nextContext
        }
        else {
          config.currentContext = null
        }
      }
      pruneUnreferencedContextResources(config, context)
    })
  }

  async setProject(
    name: string,
    project: ProjectContextSnapshot | null,
  ): Promise<StoredLunaConfig> {
    return this.set({ name, project })
  }

  async bindCredential(
    name: string,
    credential: string | null,
  ): Promise<StoredLunaConfig> {
    return this.set({ name, credential })
  }
}

export function upsertContext(
  config: StoredLunaConfig,
  input: SetContextInput,
): StoredContext {
  const existing = config.contexts[input.name]
  let instanceName = existing?.instance
  let originChanged = false

  if (input.server !== undefined) {
    const origin = normalizeServerOrigin(input.server)
    const previousOrigin = existing
      ? normalizeServerOrigin(config.instances[existing.instance].server)
      : undefined
    instanceName = ensureInstance(config, origin)
    originChanged = previousOrigin !== undefined && previousOrigin !== origin
  }

  if (!instanceName) {
    throw new CliCommandError(
      'context_server_required',
      `Context "${input.name}" requires a server.`,
      { status: 422 },
    )
  }

  const credential = input.credential === null
    ? undefined
    : input.credential ?? (originChanged ? undefined : existing?.credential)
  if (credential && !Object.hasOwn(config.credentials, credential)) {
    throw new CliCommandError(
      'credential_not_found',
      `Credential "${credential}" does not exist.`,
      { status: 404 },
    )
  }

  return {
    ...existing,
    instance: instanceName,
    credential,
    project: input.project === undefined
      ? (originChanged ? null : existing?.project)
      : input.project === null
        ? null
        : { ...input.project },
    language: input.language ?? existing?.language ?? '',
    output: input.output ?? existing?.output ?? '',
  }
}

export function normalizeServerOrigin(server: string): string {
  let url: URL
  try {
    url = new URL(server)
  }
  catch (error) {
    throw new CliCommandError(
      'server_url_invalid',
      `Server "${server}" is not a valid absolute URL.`,
      { status: 422, cause: error },
    )
  }

  if (!['http:', 'https:'].includes(url.protocol)) {
    throw new CliCommandError(
      'server_url_invalid',
      'Server URL must use http or https.',
      { status: 422 },
    )
  }
  if (url.username || url.password || url.hash || url.search) {
    throw new CliCommandError(
      'server_url_invalid',
      'Server URL cannot contain credentials, query parameters, or a fragment.',
      { status: 422 },
    )
  }
  if (url.pathname !== '/' && url.pathname !== '') {
    throw new CliCommandError(
      'server_url_subpath_unsupported',
      'Server URL must not contain a path.',
      { status: 422 },
    )
  }

  return url.origin
}

export function ensureInstance(config: StoredLunaConfig, origin: string): string {
  const existing = Object.entries(config.instances).find(
    ([, instance]) => normalizeServerOrigin(instance.server) === origin,
  )
  if (existing)
    return existing[0]

  const base = `instance-${createHash('sha256').update(origin).digest('hex').slice(0, 12)}`
  let candidate = base
  let suffix = 2
  while (Object.hasOwn(config.instances, candidate)) {
    candidate = `${base}-${suffix}`
    suffix += 1
  }
  config.instances[candidate] = defaultInstance(origin)
  return candidate
}

export function defaultInstance(server: string): StoredInstance {
  return {
    server,
    tls: { caFile: '', insecureSkipVerify: false },
    network: { proxy: '', noProxy: '' },
  }
}

export function cloneStoredConfig(config: StoredLunaConfig): StoredLunaConfig {
  return cloneConfigDocument(config)
}

export function normalizeContextName(name: string): string {
  const normalized = name.trim()
  if (!/^[A-Z0-9][\w.-]{0,62}$/i.test(normalized)) {
    throw new CliCommandError(
      'context_name_invalid',
      'Context names must be 1-63 characters using letters, numbers, dot, underscore, or hyphen.',
      { status: 422 },
    )
  }
  return normalized
}

export function pruneUnreferencedContextResources(
  config: StoredLunaConfig,
  references: {
    readonly credential?: string
    readonly instance?: string
  },
): void {
  removeUnreferencedCredential(config, references.credential)
  if (references.instance)
    removeUnreferencedInstance(config, references.instance)
}

function requireContext(config: StoredLunaConfig, name: string): StoredContext {
  const context = config.contexts[name]
  if (!context) {
    throw new CliCommandError(
      'context_not_found',
      `Context "${name}" does not exist.`,
      { status: 404 },
    )
  }
  return context
}

function removeUnreferencedCredential(
  config: StoredLunaConfig,
  credential: string | undefined,
): void {
  if (
    credential
    && !Object.values(config.contexts).some(context => context.credential === credential)
  ) {
    delete config.credentials[credential]
  }
}

function removeUnreferencedInstance(config: StoredLunaConfig, instance: string): void {
  if (!Object.values(config.contexts).some(context => context.instance === instance)) {
    delete config.instances[instance]
  }
}

type StoredContext = StoredLunaConfig['contexts'][string]
type StoredInstance = StoredLunaConfig['instances'][string]
