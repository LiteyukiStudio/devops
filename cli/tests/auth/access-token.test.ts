import { describe, expect, it } from 'vitest'

import {
  accessTokenFromEnvironment,
  getAuthStatus,
  storeValidatedAccessToken,
} from '../../src/auth/index.js'
import { MemoryConfigStore } from '../config/memory-store.js'

describe('access-token authentication', () => {
  it('stores validated access tokens and normalizes context metadata', async () => {
    const store = new MemoryConfigStore()

    await storeValidatedAccessToken(store, {
      context: ' work ',
      server: 'https://devops.example.com/',
      token: ' secret-token ',
      scopes: ['project:read', ' project:read ', 'build:write'],
      user: { id: 'usr_1', name: 'Luna' },
      project: { id: 'prj_1', identifier: 'platform' },
      makeCurrent: true,
    })

    expect(store.value.currentContext).toBe('work')
    expect(store.value.contexts.work.project).toEqual({
      id: 'prj_1',
      identifier: 'platform',
    })
    const credentialName = store.value.contexts.work.credential
    expect(credentialName).toBe('work-access-token')
    expect(store.value.credentials[credentialName!]).toMatchObject({
      type: 'access_token',
      token: 'secret-token',
      scopes: ['build:write', 'project:read'],
      user: { id: 'usr_1', name: 'Luna' },
    })
  })

  it('keeps LUNA_TOKEN process-local and reports it as an environment override', async () => {
    const store = new MemoryConfigStore()
    await storeValidatedAccessToken(store, {
      context: 'work',
      server: 'https://devops.example.com',
      token: 'stored-token',
    })
    const before = structuredClone(store.value)

    expect(accessTokenFromEnvironment({ LUNA_TOKEN: 'temporary-token' })).toEqual({
      type: 'access_token',
      token: 'temporary-token',
      scopes: [],
    })
    const [status] = await getAuthStatus(store, {
      env: { LUNA_TOKEN: 'temporary-token' },
    })

    expect(status.credential).toMatchObject({
      name: 'LUNA_TOKEN',
      source: 'environment',
      type: 'access_token',
    })
    expect(store.value).toEqual(before)
    expect(JSON.stringify(status)).not.toContain('temporary-token')
  })

  it('does not expose stored token values through auth status', async () => {
    const store = new MemoryConfigStore()
    await storeValidatedAccessToken(store, {
      context: 'work',
      server: 'https://devops.example.com',
      token: 'stored-secret',
      scopes: ['project:read'],
    })

    const [status] = await getAuthStatus(store, { env: {} })

    expect(status.authenticated).toBe(true)
    expect(status.credential).toMatchObject({
      source: 'stored',
      scopes: ['project:read'],
    })
    expect(JSON.stringify(status)).not.toContain('stored-secret')
  })
})
