import type { Readable } from 'node:stream'
import type {
  InputField,
  InputLimits,
  InputSourceKind,
  InputSourceReader,
  ParsedValueSource,
} from './types.js'
import { Buffer } from 'node:buffer'
import { readFile, stat } from 'node:fs/promises'
import { extname } from 'node:path'
import process from 'node:process'
import { parse as parseYaml } from 'yaml'
import { invalidInput, isSensitiveKey, LunaError } from '../errors/index.js'
import { parseInlinePrimitive, schemaTypes } from './primitives.js'
import { DEFAULT_INPUT_LIMITS } from './types.js'

export class NodeInputSourceReader implements InputSourceReader {
  readonly #stdin: Readable

  constructor(stdin: Readable = process.stdin) {
    this.#stdin = stdin
  }

  async readFile(path: string, maxBytes: number): Promise<Uint8Array> {
    try {
      const info = await stat(path)
      if (!info.isFile()) {
        throw invalidInput('input_not_file', `Input source "${path}" is not a regular file.`)
      }
      if (info.size > maxBytes) {
        throw sourceTooLarge('file', maxBytes, info.size)
      }
      const content = await readFile(path)
      if (content.byteLength > maxBytes) {
        throw sourceTooLarge('file', maxBytes, content.byteLength)
      }
      return content
    }
    catch (error) {
      if (error instanceof LunaError)
        throw error
      throw invalidInput('input_source_read_failed', `Unable to read input file "${path}".`, {
        fields: [{ key: 'file', code: 'read_failed' }],
        cause: error,
      })
    }
  }

  async readStdin(maxBytes: number): Promise<Uint8Array> {
    try {
      const chunks: Uint8Array[] = []
      let total = 0
      for await (const chunk of this.#stdin) {
        const bytes = Buffer.from(chunk)
        total += bytes.byteLength
        if (total > maxBytes)
          throw sourceTooLarge('stdin', maxBytes, total)
        chunks.push(bytes)
      }
      return Buffer.concat(chunks)
    }
    catch (error) {
      if (error instanceof LunaError)
        throw error
      throw invalidInput('input_source_read_failed', 'Unable to read input from stdin.', {
        fields: [{ key: 'stdin', code: 'read_failed' }],
        cause: error,
      })
    }
  }
}

export function parseValueSource(rawValue: string): ParsedValueSource {
  if (rawValue.startsWith('@@')) {
    return { kind: 'inline', inlineValue: rawValue.slice(1) }
  }
  if (rawValue === '@-')
    return { kind: 'stdin' }
  if (rawValue.startsWith('@')) {
    const path = rawValue.slice(1)
    if (!path)
      throw invalidInput('empty_input_path', 'File input path cannot be empty.')
    return { kind: 'file', path }
  }
  return { kind: 'inline', inlineValue: rawValue }
}

export function isSensitiveField(field: InputField): boolean {
  return Boolean(
    field.sensitive
    || field.schema?.writeOnly
    || field.schema?.['x-sensitive']
    || field.schema?.format === 'password'
    || isSensitiveKey(field.name),
  )
}

export function assertAllowedSource(field: InputField, source: InputSourceKind): void {
  if (isSensitiveField(field) && source === 'inline') {
    throw invalidInput(
      'sensitive_inline_forbidden',
      `${field.name} must be read from stdin, a protected file, or a secure prompt.`,
      { fields: [{ key: field.name, code: 'value_source', expected: ['file', 'stdin'], actual: source }] },
    )
  }
  const allowed = field.valueSources
  if (allowed && !allowed.includes(source)) {
    throw invalidInput('input_source_not_allowed', `${field.name} does not allow ${source} input.`, {
      fields: [{ key: field.name, code: 'value_source', expected: allowed, actual: source }],
    })
  }
  if (schemaTypes(field.schema ?? {}).includes('binary') && source !== 'file') {
    throw invalidInput('binary_inline_forbidden', `${field.name} must be read from a file.`, {
      fields: [{ key: field.name, code: 'value_source', expected: 'file', actual: source }],
    })
  }
}

export async function resolveFieldValue(
  rawValue: string,
  field: InputField,
  reader: InputSourceReader,
  limits: InputLimits = DEFAULT_INPUT_LIMITS,
): Promise<{ readonly value: unknown, readonly source: InputSourceKind }> {
  const source = parseValueSource(rawValue)
  assertAllowedSource(field, source.kind)

  if (source.kind === 'inline') {
    const value = source.inlineValue ?? ''
    const bytes = Buffer.byteLength(value)
    if (bytes > limits.inlineBytes || /[\r\n]/u.test(value)) {
      throw invalidInput(
        'inline_value_too_large',
        `${field.name} must use file or stdin input when it contains newlines or exceeds ${limits.inlineBytes} bytes.`,
        { fields: [{ key: field.name, code: 'maxBytes', expected: limits.inlineBytes, actual: bytes }] },
      )
    }
    return {
      value: parseInlinePrimitive(value, field.schema, field.name),
      source: source.kind,
    }
  }

  const bytes = source.kind === 'stdin'
    ? await reader.readStdin(limits.stdinBytes)
    : await reader.readFile(source.path!, limits.fileBytes)
  const value = parseBytes(bytes, field, source.path)
  return { value, source: source.kind }
}

export function parseStructuredBytes(
  bytes: Uint8Array,
  path: string | undefined,
  key = 'params',
): unknown {
  const text = Buffer.from(bytes).toString('utf8')
  try {
    const extension = path ? extname(path).toLocaleLowerCase() : ''
    return extension === '.yaml' || extension === '.yml' ? parseYaml(text) : JSON.parse(text)
  }
  catch (error) {
    throw invalidInput('invalid_structured_input', `${key} must contain valid JSON or YAML.`, {
      fields: [{ key, code: 'json_or_yaml' }],
      cause: error,
    })
  }
}

function parseBytes(bytes: Uint8Array, field: InputField, path?: string): unknown {
  const types = schemaTypes(field.schema ?? {})
  if (types.includes('binary'))
    return bytes
  if (types.includes('object') || types.includes('array')) {
    return parseStructuredBytes(bytes, path, field.name)
  }
  const text = Buffer.from(bytes).toString('utf8')
  return parseInlinePrimitive(text, field.schema, field.name)
}

function sourceTooLarge(source: InputSourceKind, limit: number, actual: number) {
  return invalidInput('input_source_too_large', `${source} input exceeds the configured byte limit.`, {
    fields: [{ key: source, code: 'maxBytes', expected: limit, actual }],
  })
}
