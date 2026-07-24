import type { ProjectContextSnapshot } from '../commands/types.js'
import type { LunaCredential } from '../config/schema.js'

export interface AuthUserSnapshot {
  readonly id: string
  readonly name?: string
  readonly [key: string]: unknown
}

export interface StoreAccessTokenInput {
  readonly context: string
  readonly server: string
  readonly token: string
  readonly credential?: string
  readonly scopes?: readonly string[]
  readonly user?: AuthUserSnapshot
  readonly expiresAt?: string
  readonly project?: ProjectContextSnapshot | null
  readonly makeCurrent?: boolean
}

export interface StoreOAuthCredentialInput {
  readonly context: string
  readonly server: string
  readonly accessToken: string
  readonly refreshToken?: string
  readonly tokenType?: string
  readonly credential?: string
  readonly scopes?: readonly string[]
  readonly user?: AuthUserSnapshot
  readonly expiresAt?: string
  readonly project?: ProjectContextSnapshot | null
  readonly makeCurrent?: boolean
}

export interface AuthStatusEntry {
  readonly context: string
  readonly current: boolean
  readonly server: string
  readonly authenticated: boolean
  readonly credential?: {
    readonly name: string
    readonly type: LunaCredential['type']
    readonly scopes: readonly string[]
    readonly user?: AuthUserSnapshot
    readonly expiresAt?: string
    readonly expired: boolean
    readonly source: 'stored' | 'environment'
  }
}

export interface LogoutLocalOptions {
  readonly context?: string
  readonly all?: boolean
}

export interface LogoutLocalResult {
  readonly contexts: readonly string[]
  readonly removedCredentials: readonly string[]
}
