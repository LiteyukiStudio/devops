import os from 'node:os'
import path from 'node:path'
import process from 'node:process'

export interface ConfigPathOptions {
  readonly configPath?: string
  readonly env?: Readonly<Record<string, string | undefined>>
  readonly homeDir?: string
}

export function resolveConfigPath(options: ConfigPathOptions = {}): string {
  const env = options.env ?? process.env
  const explicitPath = options.configPath ?? env.LUNA_CONFIG
  if (explicitPath?.trim()) {
    return path.resolve(expandHome(explicitPath.trim(), options.homeDir))
  }

  const home = options.homeDir ?? env.LUNA_HOME ?? os.homedir()
  return path.join(path.resolve(home), '.luna', 'auth.json')
}

function expandHome(value: string, homeDir?: string): string {
  if (value === '~')
    return homeDir ?? os.homedir()
  if (value.startsWith('~/') || value.startsWith('~\\')) {
    return path.join(homeDir ?? os.homedir(), value.slice(2))
  }
  return value
}
