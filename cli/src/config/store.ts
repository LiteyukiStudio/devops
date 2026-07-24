import type { ConfigPort, LunaConfigDocument } from '../commands/types.js'
import type { ConfigPathOptions } from './paths.js'
import type { StoredLunaConfig } from './schema.js'
import { randomUUID } from 'node:crypto'

import { constants as fsConstants } from 'node:fs'
import {
  chmod,
  lstat,
  mkdir,
  open,
  rename,
  rm,
} from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { CliCommandError } from '../commands/errors.js'
import { resolveConfigPath } from './paths.js'
import {
  emptyConfigDocument,
  parseConfigDocument,

} from './schema.js'

export interface FileConfigStoreOptions extends ConfigPathOptions {
  readonly lockTimeoutMs?: number
  readonly lockRetryMs?: number
  readonly staleLockMs?: number
  readonly platform?: NodeJS.Platform
  readonly now?: () => number
  readonly randomId?: () => string
}

export interface MutableConfigStore extends ConfigPort {
  update: (
    mutator: (config: StoredLunaConfig) => LunaConfigDocument | void,
  ) => Promise<StoredLunaConfig>
}

export class FileConfigStore implements MutableConfigStore {
  readonly path: string
  readonly #lockTimeoutMs: number
  readonly #lockRetryMs: number
  readonly #staleLockMs: number
  readonly #platform: NodeJS.Platform
  readonly #now: () => number
  readonly #randomId: () => string

  constructor(options: FileConfigStoreOptions = {}) {
    this.path = resolveConfigPath(options)
    this.#lockTimeoutMs = options.lockTimeoutMs ?? 5_000
    this.#lockRetryMs = options.lockRetryMs ?? 25
    this.#staleLockMs = options.staleLockMs ?? 30_000
    this.#platform = options.platform ?? process.platform
    this.#now = options.now ?? Date.now
    this.#randomId = options.randomId ?? randomUUID
  }

  async read(): Promise<StoredLunaConfig> {
    await this.#prepareDirectory(false)
    await this.#assertSafeFile(this.path, 0o600)

    try {
      const content = await readFileWithoutFollowingLinks(this.path)
      return parseConfigDocument(JSON.parse(content) as unknown)
    }
    catch (error) {
      if (isNodeError(error, 'ENOENT'))
        return emptyConfigDocument()
      if (error instanceof SyntaxError) {
        throw new CliCommandError(
          'config_invalid_json',
          `Configuration file "${this.path}" is not valid JSON.`,
          { status: 422, cause: error },
        )
      }
      if (error instanceof CliCommandError)
        throw error
      if (isZodError(error)) {
        throw new CliCommandError(
          'config_schema_invalid',
          `Configuration file "${this.path}" does not match the supported schema.`,
          { status: 422, details: { issues: error.issues }, cause: error },
        )
      }
      throw configIoError('read', this.path, error)
    }
  }

  async write(config: LunaConfigDocument): Promise<void> {
    await this.#withLock(async () => {
      await this.#writeUnlocked(parseConfigDocument(config))
    })
  }

  async update(
    mutator: (config: StoredLunaConfig) => LunaConfigDocument | void,
  ): Promise<StoredLunaConfig> {
    return this.#withLock(async () => {
      const current = await this.#readUnlocked()
      const working = structuredClone(current)
      const replacement = mutator(working)
      const next = parseConfigDocument(replacement ?? working)
      await this.#writeUnlocked(next)
      return next
    })
  }

  async #readUnlocked(): Promise<StoredLunaConfig> {
    await this.#assertSafeFile(this.path, 0o600)
    try {
      const content = await readFileWithoutFollowingLinks(this.path)
      return parseConfigDocument(JSON.parse(content) as unknown)
    }
    catch (error) {
      if (isNodeError(error, 'ENOENT'))
        return emptyConfigDocument()
      if (error instanceof SyntaxError) {
        throw new CliCommandError(
          'config_invalid_json',
          `Configuration file "${this.path}" is not valid JSON.`,
          { status: 422, cause: error },
        )
      }
      if (isZodError(error)) {
        throw new CliCommandError(
          'config_schema_invalid',
          `Configuration file "${this.path}" does not match the supported schema.`,
          { status: 422, details: { issues: error.issues }, cause: error },
        )
      }
      if (error instanceof CliCommandError)
        throw error
      throw configIoError('read', this.path, error)
    }
  }

  async #writeUnlocked(config: StoredLunaConfig): Promise<void> {
    await this.#prepareDirectory(true)
    await this.#assertSafeFile(this.path, 0o600)
    if (this.#platform === 'win32') {
      throw new CliCommandError(
        'secure_storage_unavailable',
        'Secure credential persistence requires a Windows DACL backend.',
        {
          status: 501,
          details: {
            path: this.path,
            remediation: 'Use LUNA_TOKEN without store=true until the DACL backend is available.',
          },
        },
      )
    }

    const directory = path.dirname(this.path)
    const temporaryPath = path.join(
      directory,
      `.${path.basename(this.path)}.${process.pid}.${this.#randomId()}.tmp`,
    )
    const flags = fsConstants.O_CREAT
      | fsConstants.O_EXCL
      | fsConstants.O_WRONLY
      | (fsConstants.O_NOFOLLOW ?? 0)
    let handle

    try {
      handle = await open(temporaryPath, flags, 0o600)
      await handle.writeFile(`${JSON.stringify(config, null, 2)}\n`, 'utf8')
      await handle.sync()
      await handle.close()
      handle = undefined
      await rename(temporaryPath, this.path)
      await chmod(this.path, 0o600)
      await syncDirectory(directory)
    }
    catch (error) {
      await handle?.close().catch(() => undefined)
      await rm(temporaryPath, { force: true }).catch(() => undefined)
      throw configIoError('write', this.path, error)
    }
  }

  async #withLock<T>(operation: () => Promise<T>): Promise<T> {
    await this.#prepareDirectory(true)
    const lockPath = `${this.path}.lock`
    const deadline = this.#now() + this.#lockTimeoutMs
    let handle

    while (!handle) {
      await this.#assertLockPathSafe(lockPath)
      try {
        const flags = fsConstants.O_CREAT
          | fsConstants.O_EXCL
          | fsConstants.O_WRONLY
          | (fsConstants.O_NOFOLLOW ?? 0)
        handle = await open(lockPath, flags, 0o600)
        await handle.writeFile(
          JSON.stringify({ pid: process.pid, createdAt: new Date(this.#now()).toISOString() }),
          'utf8',
        )
        await handle.sync()
      }
      catch (error) {
        if (!isNodeError(error, 'EEXIST')) {
          throw configIoError('lock', this.path, error)
        }
        if (await this.#removeStaleLock(lockPath))
          continue
        if (this.#now() >= deadline) {
          throw new CliCommandError(
            'config_lock_timeout',
            `Timed out waiting for the configuration lock "${lockPath}".`,
            { status: 409, retryable: true },
          )
        }
        await delay(this.#lockRetryMs)
      }
    }

    try {
      return await operation()
    }
    finally {
      await handle.close().catch(() => undefined)
      await rm(lockPath, { force: true }).catch(() => undefined)
    }
  }

  async #prepareDirectory(create: boolean): Promise<void> {
    const directory = path.dirname(this.path)
    if (create)
      await mkdir(directory, { recursive: true, mode: 0o700 })

    try {
      const stats = await lstat(directory)
      if (stats.isSymbolicLink() || !stats.isDirectory()) {
        throw unsafePathError(directory)
      }
      await this.#tightenPermissions(directory, stats.mode, stats.uid, 0o700)
    }
    catch (error) {
      if (!create && isNodeError(error, 'ENOENT'))
        return
      if (error instanceof CliCommandError)
        throw error
      throw configIoError('inspect', directory, error)
    }
  }

  async #assertSafeFile(filePath: string, mode: number): Promise<void> {
    try {
      const stats = await lstat(filePath)
      if (stats.isSymbolicLink() || !stats.isFile())
        throw unsafePathError(filePath)
      if (this.#platform === 'win32') {
        throw new CliCommandError(
          'secure_storage_unavailable',
          'Secure credential persistence requires a Windows DACL backend.',
          {
            status: 501,
            details: {
              path: filePath,
              remediation: 'Use LUNA_TOKEN until the DACL backend is available.',
            },
          },
        )
      }
      await this.#tightenPermissions(filePath, stats.mode, stats.uid, mode)
    }
    catch (error) {
      if (isNodeError(error, 'ENOENT'))
        return
      if (error instanceof CliCommandError)
        throw error
      throw configIoError('inspect', filePath, error)
    }
  }

  async #assertLockPathSafe(lockPath: string): Promise<void> {
    try {
      const stats = await lstat(lockPath)
      if (stats.isSymbolicLink() || !stats.isFile())
        throw unsafePathError(lockPath)
      await this.#tightenPermissions(lockPath, stats.mode, stats.uid, 0o600)
    }
    catch (error) {
      if (isNodeError(error, 'ENOENT'))
        return
      if (error instanceof CliCommandError)
        throw error
      throw configIoError('inspect', lockPath, error)
    }
  }

  async #removeStaleLock(lockPath: string): Promise<boolean> {
    try {
      const stats = await lstat(lockPath)
      if (stats.isSymbolicLink() || !stats.isFile())
        throw unsafePathError(lockPath)
      await this.#tightenPermissions(lockPath, stats.mode, stats.uid, 0o600)
      if (this.#now() - stats.mtimeMs <= this.#staleLockMs)
        return false
      await rm(lockPath)
      return true
    }
    catch (error) {
      if (isNodeError(error, 'ENOENT'))
        return true
      if (error instanceof CliCommandError)
        throw error
      throw configIoError('inspect', lockPath, error)
    }
  }

  async #tightenPermissions(
    target: string,
    currentMode: number,
    owner: number,
    requiredMode: number,
  ): Promise<void> {
    if (this.#platform === 'win32')
      return
    const currentUser = typeof process.getuid === 'function' ? process.getuid() : undefined
    if (currentUser !== undefined && owner !== currentUser) {
      throw new CliCommandError(
        'config_owner_mismatch',
        `Configuration path "${target}" is not owned by the current user.`,
        { status: 403 },
      )
    }
    if ((currentMode & 0o777) !== requiredMode) {
      try {
        await chmod(target, requiredMode)
      }
      catch (error) {
        throw new CliCommandError(
          'config_permissions_insecure',
          `Unable to restrict permissions for "${target}".`,
          { status: 403, cause: error },
        )
      }
    }
  }
}

export async function updateConfig(
  store: ConfigPort,
  mutator: (config: StoredLunaConfig) => LunaConfigDocument | void,
): Promise<StoredLunaConfig> {
  if ('update' in store && typeof store.update === 'function') {
    return (store as MutableConfigStore).update(mutator)
  }
  const current = parseConfigDocument(await store.read())
  const working = structuredClone(current)
  const replacement = mutator(working)
  const next = parseConfigDocument(replacement ?? working)
  await store.write(next)
  return next
}

function configIoError(
  operation: string,
  target: string,
  cause: unknown,
): CliCommandError {
  return new CliCommandError(
    'config_io_error',
    `Unable to ${operation} configuration path "${target}".`,
    { status: 500, cause },
  )
}

function unsafePathError(target: string): CliCommandError {
  return new CliCommandError(
    'config_path_unsafe',
    `Refusing to use unsafe configuration path "${target}".`,
    { status: 403 },
  )
}

function isNodeError(error: unknown, code: string): error is NodeJS.ErrnoException {
  return error instanceof Error && 'code' in error && error.code === code
}

function isZodError(error: unknown): error is { issues: readonly unknown[] } {
  return typeof error === 'object'
    && error !== null
    && 'issues' in error
    && Array.isArray(error.issues)
}

async function syncDirectory(directory: string): Promise<void> {
  try {
    const handle = await open(directory, 'r')
    await handle.sync()
    await handle.close()
  }
  catch (error) {
    if (
      isNodeError(error, 'EINVAL')
      || isNodeError(error, 'ENOTSUP')
      || isNodeError(error, 'EISDIR')
    ) {
      return
    }
    throw error
  }
}

async function readFileWithoutFollowingLinks(filePath: string): Promise<string> {
  const flags = fsConstants.O_RDONLY | (fsConstants.O_NOFOLLOW ?? 0)
  const handle = await open(filePath, flags)
  try {
    return await handle.readFile('utf8')
  }
  finally {
    await handle.close()
  }
}

function delay(milliseconds: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, milliseconds))
}
