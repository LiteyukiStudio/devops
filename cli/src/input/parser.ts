import type { LunaFieldError } from '../errors/index.js'
import type {
  CommandInputSpec,
  InputField,
  InputLimits,
  InputSourceReader,
  ParsedCommandInput,
} from './types.js'
import { invalidInput, LunaError, REDACTED_VALUE } from '../errors/index.js'
import { parseInlinePrimitive } from './primitives.js'
import {
  isSensitiveField,
  NodeInputSourceReader,
  parseStructuredBytes,
  parseValueSource,
  resolveFieldValue,
} from './sources.js'
import { DEFAULT_INPUT_LIMITS } from './types.js'
import { validateInputSchema } from './validate.js'

const KEY_PATTERN = /^[A-Za-z][A-Za-z0-9]*$/u

export interface ParseCommandInputOptions {
  readonly reader?: InputSourceReader
  readonly limits?: Partial<InputLimits>
}

export async function parseCommandInput(
  tokens: readonly string[],
  spec: CommandInputSpec,
  options: ParseCommandInputOptions = {},
): Promise<ParsedCommandInput> {
  const reader = options.reader ?? new NodeInputSourceReader()
  const limits: InputLimits = { ...DEFAULT_INPUT_LIMITS, ...options.limits }
  const fields = new Map(spec.fields.map(field => [field.name, field]))
  const values: Record<string, unknown> = {}
  const errors: LunaFieldError[] = []
  let stdinUsed = false
  let paramsSeen = false

  for (const token of tokens) {
    let pair: { key: string, value: string }
    try {
      pair = splitKeyValue(token)
    }
    catch (error) {
      collectError(errors, error)
      continue
    }

    if (pair.key === 'params') {
      paramsSeen = true
      try {
        if (Object.keys(values).length > 0) {
          throw invalidInput('params_conflict', 'params cannot be combined with individual arguments.', {
            fields: [{ key: 'params', code: 'conflict' }],
          })
        }
        const source = parseValueSource(pair.value)
        if (source.kind === 'inline') {
          throw invalidInput('params_inline_forbidden', 'params must use @file or @- input.', {
            fields: [{ key: 'params', code: 'value_source', expected: ['file', 'stdin'], actual: 'inline' }],
          })
        }
        if (source.kind === 'stdin' && stdinUsed) {
          throw invalidInput('stdin_already_used', 'Only one argument may read from stdin.')
        }
        const bytes = source.kind === 'stdin'
          ? await reader.readStdin(limits.paramsBytes)
          : await reader.readFile(source.path!, limits.paramsBytes)
        stdinUsed ||= source.kind === 'stdin'
        const params = parseStructuredBytes(bytes, source.path, 'params')
        if (!isRecord(params)) {
          throw invalidInput('params_must_be_object', 'params must contain a JSON object.', {
            fields: [{ key: 'params', code: 'type', expected: 'object', actual: jsonType(params) }],
          })
        }
        if (spec.paramsSchema)
          errors.push(...validateInputSchema(params, spec.paramsSchema))
        values.params = params
      }
      catch (error) {
        collectError(errors, error)
      }
      continue
    }

    const field = fields.get(pair.key)
    if (!field) {
      errors.push({ key: pair.key, code: 'unknown' })
      continue
    }
    if (paramsSeen) {
      errors.push({ key: pair.key, code: 'params_conflict' })
      continue
    }
    if (pair.key in values && !field.repeated) {
      errors.push({ key: pair.key, code: 'duplicate' })
      continue
    }

    try {
      const source = parseValueSource(pair.value)
      if (source.kind === 'stdin' && stdinUsed) {
        throw invalidInput('stdin_already_used', 'Only one argument may read from stdin.', {
          fields: [{ key: pair.key, code: 'stdin_already_used' }],
        })
      }
      const resolved = await resolveFieldValue(pair.value, field, reader, limits)
      stdinUsed ||= resolved.source === 'stdin'
      appendValue(values, field, resolved.value)
    }
    catch (error) {
      collectError(errors, error, field)
    }
  }

  if (!paramsSeen) {
    for (const field of spec.fields) {
      if (field.required && !(field.name in values)) {
        errors.push({ key: field.name, code: 'required' })
      }
    }
  }

  if (errors.length > 0) {
    throw new LunaError('invalid_arguments', 'Input validation failed.', {
      status: 400,
      exitCode: 2,
      fields: deduplicateErrors(errors),
      details: spec.command ? { command: spec.command } : {},
    })
  }
  return { values, stdinUsed }
}

export function splitKeyValue(token: string): { readonly key: string, readonly value: string } {
  const separator = token.indexOf('=')
  if (separator <= 0) {
    throw invalidInput('invalid_argument_syntax', `Argument "${token}" must use key=value syntax.`)
  }
  const key = token.slice(0, separator)
  if (!KEY_PATTERN.test(key)) {
    throw invalidInput('invalid_argument_key', `Argument key "${key}" is invalid.`, {
      fields: [{ key, code: 'pattern', expected: '[A-Za-z][A-Za-z0-9]*' }],
    })
  }
  return { key, value: token.slice(separator + 1) }
}

export function parseInlineValueForField(value: string, field: InputField): unknown {
  if (isSensitiveField(field)) {
    throw invalidInput('sensitive_inline_forbidden', `${field.name} cannot be provided inline.`, {
      fields: [{ key: field.name, code: 'value_source', actual: REDACTED_VALUE }],
    })
  }
  return parseInlinePrimitive(value, field.schema, field.name)
}

function appendValue(target: Record<string, unknown>, field: InputField, value: unknown): void {
  if (!field.repeated) {
    target[field.name] = value
    return
  }
  const existing = target[field.name]
  target[field.name] = existing === undefined ? [value] : [...existing as unknown[], value]
}

function collectError(errors: LunaFieldError[], error: unknown, field?: InputField): void {
  if (error instanceof LunaError && Array.isArray(error.fields)) {
    errors.push(...error.fields.map(item => ({
      ...item,
      ...(field && isSensitiveField(field) && item.actual !== undefined
        ? { actual: REDACTED_VALUE }
        : {}),
    })))
    return
  }
  errors.push({
    key: field?.name ?? 'arguments',
    code: error instanceof LunaError ? error.code : 'invalid',
  })
}

function deduplicateErrors(errors: readonly LunaFieldError[]): readonly LunaFieldError[] {
  const seen = new Set<string>()
  return errors.filter((error) => {
    const signature = `${error.key}:${error.code}`
    if (seen.has(signature))
      return false
    seen.add(signature)
    return true
  })
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function jsonType(value: unknown): string {
  if (value === null)
    return 'null'
  if (Array.isArray(value))
    return 'array'
  return typeof value
}
