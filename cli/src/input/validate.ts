import type { LunaFieldError } from '../errors/index.js'
import type { InputSchema } from './types.js'
import { LunaError } from '../errors/index.js'
import { parseDurationMilliseconds, parseRfc3339, schemaTypes } from './primitives.js'

export function validateInputSchema(
  value: unknown,
  schema: InputSchema,
  rootKey = 'params',
): readonly LunaFieldError[] {
  const errors: LunaFieldError[] = []
  visit(value, schema, rootKey, errors)
  return errors
}

export function assertValidInputSchema(
  value: unknown,
  schema: InputSchema,
  rootKey = 'params',
): void {
  const fields = validateInputSchema(value, schema, rootKey)
  if (fields.length > 0) {
    throw new LunaError('invalid_arguments', 'Input validation failed.', {
      status: 400,
      exitCode: 2,
      fields,
    })
  }
}

function visit(
  value: unknown,
  schema: InputSchema,
  path: string,
  errors: LunaFieldError[],
): void {
  const types = schemaTypes(schema)
  if (!matchesAnyType(value, types)) {
    errors.push({ key: path, code: 'type', expected: types, actual: jsonType(value) })
    return
  }
  if (value === null)
    return

  if (schema.enum && !schema.enum.some(candidate => Object.is(candidate, value))) {
    errors.push({ key: path, code: 'enum', expected: schema.enum, actual: value })
  }

  if (typeof value === 'string') {
    if (schema.minLength !== undefined && value.length < schema.minLength) {
      errors.push({ key: path, code: 'minLength', expected: schema.minLength, actual: value.length })
    }
    if (schema.maxLength !== undefined && value.length > schema.maxLength) {
      errors.push({ key: path, code: 'maxLength', expected: schema.maxLength, actual: value.length })
    }
    if (schema.format === 'duration')
      capture(() => parseDurationMilliseconds(value, path), path, 'duration', errors)
    if (schema.format === 'date-time')
      capture(() => parseRfc3339(value, path), path, 'date-time', errors)
  }

  if (typeof value === 'number') {
    if (!Number.isFinite(value))
      errors.push({ key: path, code: 'finite', actual: value })
    if (types.includes('integer') && !Number.isSafeInteger(value)) {
      errors.push({ key: path, code: 'integer', actual: value })
    }
    if (schema.minimum !== undefined && value < schema.minimum) {
      errors.push({ key: path, code: 'minimum', expected: schema.minimum, actual: value })
    }
    if (schema.maximum !== undefined && value > schema.maximum) {
      errors.push({ key: path, code: 'maximum', expected: schema.maximum, actual: value })
    }
  }

  if (Array.isArray(value)) {
    if (schema.minItems !== undefined && value.length < schema.minItems) {
      errors.push({ key: path, code: 'minItems', expected: schema.minItems, actual: value.length })
    }
    if (schema.maxItems !== undefined && value.length > schema.maxItems) {
      errors.push({ key: path, code: 'maxItems', expected: schema.maxItems, actual: value.length })
    }
    if (schema.items)
      value.forEach((item, index) => visit(item, schema.items!, `${path}[${index}]`, errors))
  }

  if (isRecord(value)) {
    for (const required of schema.required ?? []) {
      if (!(required in value))
        errors.push({ key: `${path}.${required}`, code: 'required' })
    }
    for (const [key, child] of Object.entries(value)) {
      const childSchema = schema.properties?.[key]
      if (childSchema) {
        visit(child, childSchema, `${path}.${key}`, errors)
      }
      else if (schema.additionalProperties === false) {
        errors.push({ key: `${path}.${key}`, code: 'additionalProperties' })
      }
      else if (isRecord(schema.additionalProperties)) {
        visit(child, schema.additionalProperties, `${path}.${key}`, errors)
      }
    }
  }
}

function capture(
  action: () => unknown,
  key: string,
  code: string,
  errors: LunaFieldError[],
): void {
  try {
    action()
  }
  catch {
    errors.push({ key, code })
  }
}

function matchesAnyType(value: unknown, types: readonly string[]): boolean {
  return types.some((type) => {
    switch (type) {
      case 'null': return value === null
      case 'array': return Array.isArray(value)
      case 'object': return isRecord(value)
      case 'integer': return typeof value === 'number' && Number.isSafeInteger(value)
      case 'number': return typeof value === 'number' && Number.isFinite(value)
      case 'binary': return value instanceof Uint8Array
      case 'boolean': return typeof value === 'boolean'
      case 'string': return typeof value === 'string'
      default: return false
    }
  })
}

function jsonType(value: unknown): string {
  if (value === null)
    return 'null'
  if (Array.isArray(value))
    return 'array'
  if (value instanceof Uint8Array)
    return 'binary'
  return typeof value
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
