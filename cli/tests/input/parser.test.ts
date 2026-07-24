import type { CommandInputSpec, InputSourceReader } from '../../src/input/index.js'
import { Buffer } from 'node:buffer'
import { describe, expect, it } from 'vitest'
import { LunaError, REDACTED_VALUE } from '../../src/errors/index.js'
import {
  parseCommandInput,
} from '../../src/input/index.js'

class MemoryReader implements InputSourceReader {
  readonly files: Readonly<Record<string, string>>
  readonly stdin: string
  stdinReads = 0

  constructor(files: Readonly<Record<string, string>> = {}, stdin = '') {
    this.files = files
    this.stdin = stdin
  }

  async readFile(path: string, maxBytes: number): Promise<Uint8Array> {
    const content = this.files[path]
    if (content === undefined)
      throw new Error(`Missing test file: ${path}`)
    const bytes = Buffer.from(content)
    if (bytes.byteLength > maxBytes)
      throw new Error('test input too large')
    return bytes
  }

  async readStdin(maxBytes: number): Promise<Uint8Array> {
    this.stdinReads += 1
    const bytes = Buffer.from(this.stdin)
    if (bytes.byteLength > maxBytes)
      throw new Error('test input too large')
    return bytes
  }
}

const spec: CommandInputSpec = {
  command: 'application.create',
  fields: [
    { name: 'name', required: true, schema: { type: 'string', minLength: 2 } },
    { name: 'enabled', schema: { type: 'boolean' } },
    { name: 'replicas', schema: { type: 'integer', minimum: 1 } },
    { name: 'tag', repeated: true, schema: { type: 'string' } },
    { name: 'secret', sensitive: true, schema: { type: 'string' } },
    { name: 'body', schema: { type: 'string' } },
  ],
  paramsSchema: {
    type: 'object',
    required: ['name'],
    additionalProperties: false,
    properties: {
      name: { type: 'string', minLength: 2 },
      enabled: { type: 'boolean' },
    },
  },
}

describe('command input parser', () => {
  it('parses typed and repeated key=value arguments', async () => {
    const result = await parseCommandInput(
      ['name=demo', 'enabled=true', 'replicas=2', 'tag=a', 'tag=b', 'body=@@literal'],
      spec,
      { reader: new MemoryReader() },
    )
    expect(result.values).toEqual({
      name: 'demo',
      enabled: true,
      replicas: 2,
      tag: ['a', 'b'],
      body: '@literal',
    })
  })

  it('loads params from JSON or YAML files and validates the object', async () => {
    const json = await parseCommandInput(['params=@params.json'], spec, {
      reader: new MemoryReader({ 'params.json': '{"name":"demo","enabled":true}' }),
    })
    expect(json.values.params).toEqual({ name: 'demo', enabled: true })

    const yaml = await parseCommandInput(['params=@params.yaml'], spec, {
      reader: new MemoryReader({ 'params.yaml': 'name: demo\nenabled: false\n' }),
    })
    expect(yaml.values.params).toEqual({ name: 'demo', enabled: false })
  })

  it('allows one stdin source and rejects a second one', async () => {
    await expect(parseCommandInput(['name=@-', 'body=@-'], spec, {
      reader: new MemoryReader({}, 'demo'),
    })).rejects.toMatchObject({
      code: 'invalid_arguments',
      fields: expect.arrayContaining([{ key: 'body', code: 'stdin_already_used' }]),
    })
  })

  it('rejects inline secrets without exposing their value', async () => {
    const secret = 'super-secret-value'
    await expect(parseCommandInput([`name=demo`, `secret=${secret}`], spec, {
      reader: new MemoryReader(),
    })).rejects.toSatisfy((error: unknown) => {
      expect(error).toBeInstanceOf(LunaError)
      const serialized = JSON.stringify(error)
      expect(serialized).not.toContain(secret)
      expect(serialized).toContain(REDACTED_VALUE)
      return true
    })
  })

  it('aggregates unknown, duplicate and missing argument errors', async () => {
    await expect(parseCommandInput(['unknown=x', 'enabled=true', 'enabled=false'], spec, {
      reader: new MemoryReader(),
    })).rejects.toMatchObject({
      code: 'invalid_arguments',
      fields: expect.arrayContaining([
        { key: 'unknown', code: 'unknown' },
        { key: 'enabled', code: 'duplicate' },
        { key: 'name', code: 'required' },
      ]),
    })
  })

  it('rejects params mixed with business arguments and multiline inline values', async () => {
    await expect(parseCommandInput(['name=demo', 'params=@params.json'], spec, {
      reader: new MemoryReader({ 'params.json': '{"name":"other"}' }),
    })).rejects.toMatchObject({ code: 'invalid_arguments' })

    await expect(parseCommandInput(['name=line1\nline2'], spec, {
      reader: new MemoryReader(),
    })).rejects.toMatchObject({
      fields: expect.arrayContaining([
        expect.objectContaining({ key: 'name', code: 'maxBytes' }),
      ]),
    })
  })
})
