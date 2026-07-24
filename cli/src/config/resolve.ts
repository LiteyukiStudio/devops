import type {
  LunaContext,
  LunaInstance,
  OutputFormat,
  ProjectContextSnapshot,
} from '../commands/types.js'
import type { LunaCredential } from './schema.js'
import process from 'node:process'
import { CliCommandError } from '../commands/errors.js'
import { normalizeServerOrigin } from './context.js'
import {
  OUTPUT_FORMATS,
  parseConfigDocument,
} from './schema.js'

export type ResolutionSource
  = | 'argument'
    | 'environment'
    | 'context'
    | 'default'
    | 'none'

export interface ResolveContextOptions {
  readonly context?: string
  readonly server?: string
  readonly project?: string
  readonly output?: OutputFormat | ''
  readonly language?: string
  readonly env?: Readonly<Record<string, string | undefined>>
}

export interface ResolvedRuntimeContext {
  readonly contextName?: string
  readonly context?: LunaContext
  readonly instance?: LunaInstance
  readonly server?: string
  readonly project?: ProjectContextSnapshot
  readonly credential?: LunaCredential
  readonly output?: OutputFormat | ''
  readonly language?: string
  readonly sources: {
    readonly context: ResolutionSource
    readonly server: ResolutionSource
    readonly project: ResolutionSource
    readonly credential: ResolutionSource
    readonly output: ResolutionSource
    readonly language: ResolutionSource
  }
}

export function resolveRuntimeContext(
  rawConfig: unknown,
  options: ResolveContextOptions = {},
): ResolvedRuntimeContext {
  const config = parseConfigDocument(rawConfig)
  const env = options.env ?? process.env
  const explicitContext = nonEmpty(options.context)
  const environmentContext = nonEmpty(env.LUNA_CONTEXT)
  const contextName = explicitContext ?? environmentContext ?? config.currentContext ?? undefined
  const context = contextName ? config.contexts[contextName] : undefined
  if (contextName && !context) {
    throw new CliCommandError(
      'context_not_found',
      `Context "${contextName}" does not exist.`,
      { status: 404 },
    )
  }

  const contextInstance = context ? config.instances[context.instance] : undefined
  const explicitServer = nonEmpty(options.server)
  const environmentServer = nonEmpty(env.LUNA_SERVER)
  const serverOverride = explicitServer ?? environmentServer
  const server = serverOverride
    ? normalizeServerOrigin(serverOverride)
    : contextInstance
      ? normalizeServerOrigin(contextInstance.server)
      : undefined
  const sameOrigin = Boolean(
    server
    && contextInstance
    && server === normalizeServerOrigin(contextInstance.server),
  )
  const environmentToken = nonEmpty(env.LUNA_TOKEN)
  const credential = environmentToken
    ? {
        type: 'access_token' as const,
        token: environmentToken,
        scopes: [],
      }
    : sameOrigin && context?.credential
      ? config.credentials[context.credential]
      : undefined

  const explicitProject = nonEmpty(options.project)
  const environmentProject = nonEmpty(env.LUNA_PROJECT)
  const projectOverride = explicitProject ?? environmentProject
  const project = projectOverride
    ? { id: projectOverride }
    : sameOrigin
      ? context?.project ?? undefined
      : undefined

  const explicitOutput = options.output === '' ? undefined : options.output
  const environmentOutput = outputValue(env.LUNA_OUTPUT)
  const contextOutput = context?.output || undefined
  const output = explicitOutput ?? environmentOutput ?? contextOutput
  const explicitLanguage = nonEmpty(options.language)
  const environmentLanguage = nonEmpty(env.LUNA_LANG)
  const contextLanguage = nonEmpty(context?.language)
  const language = explicitLanguage ?? environmentLanguage ?? contextLanguage

  return {
    contextName,
    context,
    instance: sameOrigin ? contextInstance : server ? { server } : undefined,
    server,
    project,
    credential,
    output,
    language,
    sources: {
      context: explicitContext
        ? 'argument'
        : environmentContext
          ? 'environment'
          : contextName
            ? 'context'
            : 'none',
      server: explicitServer
        ? 'argument'
        : environmentServer
          ? 'environment'
          : contextInstance
            ? 'context'
            : 'none',
      project: explicitProject
        ? 'argument'
        : environmentProject
          ? 'environment'
          : project
            ? 'context'
            : 'none',
      credential: environmentToken
        ? 'environment'
        : credential
          ? 'context'
          : 'none',
      output: explicitOutput
        ? 'argument'
        : environmentOutput
          ? 'environment'
          : contextOutput
            ? 'context'
            : 'default',
      language: explicitLanguage
        ? 'argument'
        : environmentLanguage
          ? 'environment'
          : contextLanguage
            ? 'context'
            : 'default',
    },
  }
}

function nonEmpty(value: string | undefined | null): string | undefined {
  const normalized = value?.trim()
  return normalized || undefined
}

function outputValue(value: string | undefined): OutputFormat | undefined {
  const normalized = nonEmpty(value)
  if (!normalized)
    return undefined
  if (!(OUTPUT_FORMATS as readonly string[]).includes(normalized)) {
    throw new CliCommandError(
      'output_format_invalid',
      `Unsupported output format "${normalized}".`,
      { status: 422 },
    )
  }
  return normalized as OutputFormat
}
