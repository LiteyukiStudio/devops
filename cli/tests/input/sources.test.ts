import type { InputSourceReader } from '../../src/input/index.js'
import { Buffer } from 'node:buffer'
import { Readable } from 'node:stream'
import { describe, expect, it } from 'vitest'
import {
  NodeInputSourceReader,
  resolveFieldValue,
} from '../../src/input/index.js'

class MemoryReader implements InputSourceReader {
  async readFile(): Promise<Uint8Array> {
    return Buffer.from([0xDE, 0xAD, 0xBE, 0xEF])
  }

  async readStdin(): Promise<Uint8Array> {
    return Buffer.from([0xDE, 0xAD, 0xBE, 0xEF])
  }
}

describe('input sources', () => {
  it('enforces stdin limits before returning input', async () => {
    const reader = new NodeInputSourceReader(Readable.from(['123', '45']))

    await expect(reader.readStdin(4)).rejects.toMatchObject({
      code: 'input_source_too_large',
      exitCode: 2,
    })
  })

  it('normalizes file read failures', async () => {
    const reader = new NodeInputSourceReader(Readable.from([]))

    await expect(reader.readFile('/path/that/does/not/exist/luna-input', 1024))
      .rejects
      .toMatchObject({
        code: 'input_source_read_failed',
        exitCode: 2,
      })
  })

  it('accepts binary values only from files', async () => {
    const field = {
      name: 'archive',
      schema: { type: 'binary' as const },
    }
    const reader = new MemoryReader()

    await expect(resolveFieldValue('@-', field, reader))
      .rejects
      .toMatchObject({ code: 'binary_inline_forbidden' })

    await expect(resolveFieldValue('@archive.bin', field, reader))
      .resolves
      .toEqual({
        value: Buffer.from([0xDE, 0xAD, 0xBE, 0xEF]),
        source: 'file',
      })
  })
})
