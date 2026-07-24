import { describe, expect, it } from 'vitest'

import {
  logoutLocal,
  storeValidatedAccessToken,
} from '../../src/auth/index.js'
import { ContextService } from '../../src/config/index.js'
import { MemoryConfigStore } from '../config/memory-store.js'

describe('logoutLocal', () => {
  it('unbinds the current context and removes its orphaned credential', async () => {
    const store = new MemoryConfigStore()
    await storeValidatedAccessToken(store, {
      context: 'work',
      server: 'https://devops.example.com',
      token: 'secret',
    })

    const result = await logoutLocal(store)

    expect(result).toEqual({
      contexts: ['work'],
      removedCredentials: ['work-access-token'],
    })
    expect(store.value.contexts.work.credential).toBeUndefined()
    expect(store.value.credentials).toEqual({})
    expect(store.value.currentContext).toBe('work')
  })

  it('preserves credentials that remain referenced by another context', async () => {
    const store = new MemoryConfigStore()
    await storeValidatedAccessToken(store, {
      context: 'first',
      server: 'https://devops.example.com',
      token: 'secret',
    })
    const credential = store.value.contexts.first.credential!
    await new ContextService(store).set({
      name: 'second',
      server: 'https://devops.example.com',
      credential,
    })

    const result = await logoutLocal(store, { context: 'first' })

    expect(result.removedCredentials).toEqual([])
    expect(store.value.credentials[credential]).toBeDefined()
    expect(store.value.contexts.second.credential).toBe(credential)
  })

  it('logs out every context and removes all orphaned credentials', async () => {
    const store = new MemoryConfigStore()
    await storeValidatedAccessToken(store, {
      context: 'first',
      server: 'https://one.example.com',
      token: 'first-secret',
    })
    await storeValidatedAccessToken(store, {
      context: 'second',
      server: 'https://two.example.com',
      token: 'second-secret',
    })

    const result = await logoutLocal(store, { all: true })

    expect(result.contexts).toEqual(['first', 'second'])
    expect(result.removedCredentials).toEqual([
      'first-access-token',
      'second-access-token',
    ])
    expect(store.value.credentials).toEqual({})
  })
})
