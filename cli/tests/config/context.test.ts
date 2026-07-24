import { describe, expect, it } from 'vitest'

import {
  ContextService,
  emptyConfigDocument,
  normalizeServerOrigin,
} from '../../src/config/index.js'
import { MemoryConfigStore } from './memory-store.js'

describe('contextService', () => {
  it('reuses instances by normalized origin and manages current context', async () => {
    const store = new MemoryConfigStore()
    const service = new ContextService(store)

    await service.set({
      name: 'work-admin',
      server: 'https://devops.example.com/',
      makeCurrent: true,
    })
    await service.set({
      name: 'work-owner',
      server: 'https://devops.example.com',
    })

    expect(Object.keys(store.value.instances)).toHaveLength(1)
    expect(store.value.currentContext).toBe('work-admin')
    await service.use('work-owner')
    expect(store.value.currentContext).toBe('work-owner')
  })

  it('clears credentials and projects when the origin changes', async () => {
    const config = emptyConfigDocument()
    config.instances.old = {
      server: 'https://old.example.com',
      tls: { caFile: '', insecureSkipVerify: false },
      network: { proxy: '', noProxy: '' },
    }
    config.credentials.token = {
      type: 'access_token',
      token: 'secret',
      scopes: [],
    }
    config.contexts.work = {
      instance: 'old',
      credential: 'token',
      project: { id: 'prj_old' },
      language: '',
      output: '',
    }
    config.currentContext = 'work'
    const store = new MemoryConfigStore(config)

    await new ContextService(store).set({
      name: 'work',
      server: 'https://new.example.com',
    })

    expect(store.value.contexts.work.credential).toBeUndefined()
    expect(store.value.contexts.work.project).toBeNull()
    expect(store.value.instances[store.value.contexts.work.instance].server).toBe(
      'https://new.example.com',
    )
    expect(store.value.credentials).toEqual({})
    expect(store.value.instances.old).toBeUndefined()
  })

  it('requires confirmation before deleting the current context', async () => {
    const store = new MemoryConfigStore()
    const service = new ContextService(store)
    await service.set({
      name: 'work',
      server: 'https://devops.example.com',
      makeCurrent: true,
    })

    await expect(service.delete('work')).rejects.toMatchObject({
      code: 'current_context_delete_requires_confirmation',
    })
    await service.delete('work', { allowCurrent: true })
    expect(store.value.currentContext).toBeNull()
    expect(store.value.contexts).toEqual({})
    expect(store.value.instances).toEqual({})
  })

  it('renames the current context without changing its instance', async () => {
    const store = new MemoryConfigStore()
    const service = new ContextService(store)
    await service.set({
      name: 'before',
      server: 'https://devops.example.com',
      makeCurrent: true,
    })
    const instance = store.value.contexts.before.instance

    await service.rename('before', 'after')

    expect(store.value.currentContext).toBe('after')
    expect(store.value.contexts.after.instance).toBe(instance)
    expect(store.value.contexts.before).toBeUndefined()
  })

  it('views credential metadata without exposing secret material', async () => {
    const config = emptyConfigDocument()
    config.instances.work = {
      server: 'https://devops.example.com',
      tls: { caFile: '', insecureSkipVerify: false },
      network: { proxy: '', noProxy: '' },
    }
    config.credentials.oauth = {
      type: 'oauth',
      accessToken: 'access-secret',
      refreshToken: 'refresh-secret',
      scopes: ['project:read'],
      user: { id: 'usr_1', name: 'Luna' },
    }
    config.contexts.work = {
      instance: 'work',
      credential: 'oauth',
      output: '',
      language: '',
    }
    config.currentContext = 'work'

    const view = await new ContextService(new MemoryConfigStore(config)).view('work')

    expect(view.credential).toEqual({
      name: 'oauth',
      type: 'oauth',
      scopes: ['project:read'],
      expiresAt: undefined,
      user: { id: 'usr_1', name: 'Luna' },
    })
    expect(JSON.stringify(view)).not.toContain('secret')
  })
})

describe('normalizeServerOrigin', () => {
  it('normalizes root URLs and rejects unsafe URL components', () => {
    expect(normalizeServerOrigin('https://example.com:443/')).toBe('https://example.com')
    expect(() => normalizeServerOrigin('https://user@example.com')).toThrowError(
      expect.objectContaining({ code: 'server_url_invalid' }),
    )
    expect(() => normalizeServerOrigin('https://example.com/api')).toThrowError(
      expect.objectContaining({ code: 'server_url_subpath_unsupported' }),
    )
  })
})
