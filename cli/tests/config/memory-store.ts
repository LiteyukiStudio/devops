import type {
  ConfigPort,
  LunaConfigDocument,
} from '../../src/commands/types.js'
import type { StoredLunaConfig } from '../../src/config/schema.js'
import {
  emptyConfigDocument,
  parseConfigDocument,

} from '../../src/config/schema.js'

export class MemoryConfigStore implements ConfigPort {
  value: StoredLunaConfig

  constructor(initial: LunaConfigDocument = emptyConfigDocument()) {
    this.value = parseConfigDocument(initial)
  }

  async read(): Promise<StoredLunaConfig> {
    return structuredClone(this.value)
  }

  async write(config: LunaConfigDocument): Promise<void> {
    this.value = parseConfigDocument(structuredClone(config))
  }
}
