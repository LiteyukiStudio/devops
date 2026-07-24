import type { ConfigPort } from '../commands/types.js'
import type { LogoutLocalOptions, LogoutLocalResult } from './types.js'
import { CliCommandError } from '../commands/errors.js'
import { updateConfig } from '../config/store.js'

export async function logoutLocal(
  store: ConfigPort,
  options: LogoutLocalOptions = {},
): Promise<LogoutLocalResult> {
  let result: LogoutLocalResult = { contexts: [], removedCredentials: [] }

  await updateConfig(store, (config) => {
    const contexts = options.all
      ? Object.keys(config.contexts).sort()
      : [options.context ?? config.currentContext].filter(
          (name): name is string => Boolean(name),
        )
    if (contexts.length === 0) {
      result = { contexts: [], removedCredentials: [] }
      return
    }

    const credentials = new Set<string>()
    for (const name of contexts) {
      const context = config.contexts[name]
      if (!context) {
        throw new CliCommandError(
          'context_not_found',
          `Context "${name}" does not exist.`,
          { status: 404 },
        )
      }
      if (context.credential)
        credentials.add(context.credential)
      delete context.credential
    }

    const removed: string[] = []
    for (const credential of credentials) {
      const stillReferenced = Object.values(config.contexts).some(
        context => context.credential === credential,
      )
      if (!stillReferenced) {
        delete config.credentials[credential]
        removed.push(credential)
      }
    }
    result = { contexts, removedCredentials: removed.sort() }
  })

  return result
}
