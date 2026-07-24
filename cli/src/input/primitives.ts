import type { InputSchema, InputValueType } from './types.js'
import { invalidInput } from '../errors/index.js'

const INTEGER_PATTERN = /^-?(?:0|[1-9]\d*)$/u
const NUMBER_PATTERN = /^-?(?:(?:0|[1-9]\d*)(?:\.\d+)?|\.\d+)(?:e[+-]?\d+)?$/iu
const DURATION_PATTERN = /^[+-]?(?:0|(?:\d+(?:\.\d+)?(?:ns|us|µs|μs|ms|[smh]))+)$/u
const DURATION_PART_PATTERN = /(\d+(?:\.\d+)?)(ns|us|µs|μs|ms|[smh])/gu
const RFC3339_PATTERN
  = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/u

const DURATION_MULTIPLIERS: Readonly<Record<string, number>> = {
  ns: 0.000001,
  us: 0.001,
  µs: 0.001,
  μs: 0.001,
  ms: 1,
  s: 1_000,
  m: 60_000,
  h: 3_600_000,
}

export function parseBoolean(value: string, key = 'value'): boolean {
  if (value === 'true')
    return true
  if (value === 'false')
    return false
  throw invalidInput('invalid_boolean', `${key} must be "true" or "false".`, {
    fields: [{ key, code: 'boolean', expected: ['true', 'false'], actual: value }],
  })
}

export function parseInteger(value: string, key = 'value'): number {
  if (!INTEGER_PATTERN.test(value)) {
    throw invalidInput('invalid_integer', `${key} must be a decimal integer.`, {
      fields: [{ key, code: 'integer', actual: value }],
    })
  }
  const parsed = Number(value)
  if (!Number.isSafeInteger(parsed)) {
    throw invalidInput('integer_out_of_range', `${key} is outside the safe integer range.`, {
      fields: [{ key, code: 'safe_integer', actual: value }],
    })
  }
  return parsed
}

export function parseNumberValue(value: string, key = 'value'): number {
  if (!NUMBER_PATTERN.test(value)) {
    throw invalidInput('invalid_number', `${key} must be a finite decimal number.`, {
      fields: [{ key, code: 'number', actual: value }],
    })
  }
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) {
    throw invalidInput('invalid_number', `${key} must be a finite decimal number.`, {
      fields: [{ key, code: 'finite', actual: value }],
    })
  }
  return parsed
}

export function parseDurationMilliseconds(value: string, key = 'duration'): number {
  if (!DURATION_PATTERN.test(value)) {
    throw invalidInput('invalid_duration', `${key} must use Go duration syntax.`, {
      fields: [{ key, code: 'duration', expected: '30s, 5m or 2h', actual: value }],
    })
  }
  if (value === '0' || value === '+0' || value === '-0')
    return 0

  const sign = value.startsWith('-') ? -1 : 1
  const unsigned = value.replace(/^[+-]/u, '')
  let consumed = ''
  let milliseconds = 0
  for (const match of unsigned.matchAll(DURATION_PART_PATTERN)) {
    const [, quantity, unit] = match
    consumed += match[0]
    milliseconds += Number(quantity) * DURATION_MULTIPLIERS[unit!]!
  }
  if (consumed !== unsigned || !Number.isFinite(milliseconds)) {
    throw invalidInput('invalid_duration', `${key} must use Go duration syntax.`, {
      fields: [{ key, code: 'duration', actual: value }],
    })
  }
  return sign * milliseconds
}

export function parseRfc3339(value: string, key = 'value'): string {
  if (!RFC3339_PATTERN.test(value) || Number.isNaN(Date.parse(value))) {
    throw invalidInput('invalid_date_time', `${key} must be an RFC 3339 timestamp.`, {
      fields: [{ key, code: 'date-time', actual: value }],
    })
  }
  return value
}

export function parseInlinePrimitive(
  value: string,
  schema: InputSchema = {},
  key = 'value',
): unknown {
  const types = schemaTypes(schema)
  if (value === 'null' && types.includes('null'))
    return null

  const type = preferredType(types)
  let parsed: unknown
  switch (type) {
    case 'boolean':
      parsed = parseBoolean(value, key)
      break
    case 'integer':
      parsed = parseInteger(value, key)
      break
    case 'number':
      parsed = parseNumberValue(value, key)
      break
    case 'object':
    case 'array':
      parsed = parseJson(value, key)
      break
    case 'binary':
      throw invalidInput('binary_inline_forbidden', `${key} must be read from a file.`, {
        fields: [{ key, code: 'value_source', expected: 'file', actual: 'inline' }],
      })
    default:
      parsed = parseStringFormat(value, schema, key)
  }

  validatePrimitiveConstraints(parsed, schema, key)
  return parsed
}

export function schemaTypes(schema: InputSchema): readonly InputValueType[] {
  if (Array.isArray(schema.type))
    return schema.type
  return typeof schema.type === 'string' ? [schema.type] : ['string']
}

function preferredType(types: readonly InputValueType[]): InputValueType {
  return types.find(type => type !== 'null') ?? 'null'
}

function parseStringFormat(value: string, schema: InputSchema, key: string): string {
  if (schema.format === 'duration') {
    parseDurationMilliseconds(value, key)
  }
  else if (schema.format === 'date-time') {
    parseRfc3339(value, key)
  }
  return value
}

function parseJson(value: string, key: string): unknown {
  try {
    return JSON.parse(value)
  }
  catch (error) {
    throw invalidInput('invalid_json', `${key} must contain valid JSON.`, {
      fields: [{ key, code: 'json' }],
      cause: error,
    })
  }
}

function validatePrimitiveConstraints(value: unknown, schema: InputSchema, key: string): void {
  if (schema.enum && !schema.enum.some(candidate => Object.is(candidate, value))) {
    throw invalidInput('invalid_enum', `${key} is not an allowed value.`, {
      fields: [{ key, code: 'enum', expected: schema.enum, actual: value }],
    })
  }
  if (typeof value === 'string') {
    if (schema.minLength !== undefined && value.length < schema.minLength) {
      throw invalidInput('string_too_short', `${key} is shorter than allowed.`, {
        fields: [{ key, code: 'minLength', expected: schema.minLength, actual: value.length }],
      })
    }
    if (schema.maxLength !== undefined && value.length > schema.maxLength) {
      throw invalidInput('string_too_long', `${key} is longer than allowed.`, {
        fields: [{ key, code: 'maxLength', expected: schema.maxLength, actual: value.length }],
      })
    }
  }
  if (typeof value === 'number') {
    if (schema.minimum !== undefined && value < schema.minimum) {
      throw invalidInput('number_too_small', `${key} is smaller than allowed.`, {
        fields: [{ key, code: 'minimum', expected: schema.minimum, actual: value }],
      })
    }
    if (schema.maximum !== undefined && value > schema.maximum) {
      throw invalidInput('number_too_large', `${key} is larger than allowed.`, {
        fields: [{ key, code: 'maximum', expected: schema.maximum, actual: value }],
      })
    }
  }
}
