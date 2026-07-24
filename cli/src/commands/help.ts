import type { CommandRegistry } from './registry.js'
import type { CommandResult, NormalizedCommandMetadata } from './types.js'
import { Buffer } from 'node:buffer'
import { CliCommandError } from './errors.js'

const DEFAULT_LIMIT = 20
const MAX_LIMIT = 100

export function catalogResult(
  registry: CommandRegistry,
  params: Readonly<Record<string, unknown>>,
): CommandResult {
  const limit = integerParam(params.limit, DEFAULT_LIMIT, 1, MAX_LIMIT)
  const offset = decodeCursor(stringParam(params.cursor))
  const commands = registry.list({
    query: stringParam(params.query),
    category: stringParam(params.category),
    risk: stringParam(params.risk),
    scope: stringParam(params.scope),
    transport: stringParam(params.transport),
    includeHidden: booleanParam(params.all, false),
  })
  const items = commands.slice(offset, offset + limit).map(({ metadata }) => compactEntry(metadata))
  const nextOffset = offset + items.length

  return {
    schemaVersion: 'help.catalog/v1',
    data: {
      ...registry.catalogMetadata,
      items,
      nextCursor: nextOffset < commands.length ? encodeCursor(nextOffset) : null,
      total: commands.length,
    },
  }
}

export function commandHelpResult(
  registry: CommandRegistry,
  params: Readonly<Record<string, unknown>>,
): CommandResult {
  const path = requiredStringParam(params.path, 'path')
  const metadata = registry.require(path).metadata
  return {
    schemaVersion: 'help.command/v1',
    data: {
      ...registry.catalogMetadata,
      command: fullEntry(metadata),
    },
  }
}

function compactEntry(metadata: NormalizedCommandMetadata): Readonly<Record<string, unknown>> {
  return {
    path: metadata.canonicalPath,
    category: metadata.category,
    tool: metadata.tool,
    source: metadata.source,
    summary: metadata.summary ?? '',
    risk: metadata.risk,
    transport: metadata.transport,
    projectContext: metadata.projectContext,
    scopes: metadata.scopes,
    serverSupported: null,
  }
}

function fullEntry(metadata: NormalizedCommandMetadata): Readonly<Record<string, unknown>> {
  return {
    ...compactEntry(metadata),
    aliases: metadata.aliases,
    operationId: metadata.operationId ?? null,
    consumedOperations: metadata.consumedOperations ?? [],
    description: metadata.description ?? '',
    parameters: metadata.parameters,
    inputSchema: metadata.inputSchema ?? {},
    outputSchema: metadata.outputSchema ?? {},
    errorSchema: metadata.errorSchema ?? {},
    schemaVersion: metadata.schemaVersion ?? 'unversioned',
    schemaDigest: metadata.schemaDigest ?? 'unavailable',
    mfaPurpose: metadata.mfaPurpose ?? null,
    agentAllowed: metadata.agentAllowed,
    examples: metadata.examples ?? [],
  }
}

function encodeCursor(offset: number): string {
  return Buffer.from(`v1:${offset}`, 'utf8').toString('base64url')
}

function decodeCursor(value: string | undefined): number {
  if (!value)
    return 0
  try {
    const decoded = Buffer.from(value, 'base64url').toString('utf8')
    const match = /^v1:(\d+)$/.exec(decoded)
    if (!match)
      throw new Error('invalid cursor')
    return Number(match[1])
  }
  catch (cause) {
    throw new CliCommandError('invalid_arguments', 'Invalid catalog cursor.', {
      status: 400,
      exitCode: 2,
      details: { key: 'cursor' },
      cause,
    })
  }
}

function stringParam(value: unknown): string | undefined {
  return typeof value === 'string' && value.length > 0 ? value : undefined
}

function requiredStringParam(value: unknown, key: string): string {
  const result = stringParam(value)
  if (!result) {
    throw new CliCommandError('invalid_arguments', `Missing required argument "${key}".`, {
      status: 400,
      exitCode: 2,
      details: { key, code: 'required' },
    })
  }
  return result
}

function booleanParam(value: unknown, fallback: boolean): boolean {
  return typeof value === 'boolean' ? value : fallback
}

function integerParam(
  value: unknown,
  fallback: number,
  minimum: number,
  maximum: number,
): number {
  if (value === undefined)
    return fallback
  if (typeof value !== 'number' || !Number.isInteger(value) || value < minimum || value > maximum) {
    throw new CliCommandError(
      'invalid_arguments',
      `Value must be an integer between ${minimum} and ${maximum}.`,
      { status: 400, exitCode: 2 },
    )
  }
  return value
}
