import YAML from 'yaml'

type KubeconfigObject = Record<string, unknown>

export interface KubeconfigContextOption {
  cluster: string
  current: boolean
  name: string
  namespace: string
  server: string
  user: string
}

export interface KubeconfigInspection {
  contexts: KubeconfigContextOption[]
  currentContext: string
  error: boolean
}

export function inspectKubeconfig(value: string): KubeconfigInspection {
  const source = value.trim()
  if (!source)
    return { contexts: [], currentContext: '', error: false }
  try {
    const config = parseKubeconfigObject(source)
    const currentContext = stringValue(config['current-context'])
    const clusterServers = new Map(
      arrayValue(config.clusters)
        .map(entry => [stringValue(entry.name), stringValue(objectValue(entry.cluster).server)] as const)
        .filter(([name]) => name),
    )
    const contexts = arrayValue(config.contexts)
      .map((entry) => {
        const context = objectValue(entry.context)
        const name = stringValue(entry.name)
        const cluster = stringValue(context.cluster)
        return {
          cluster,
          current: name === currentContext,
          name,
          namespace: stringValue(context.namespace),
          server: clusterServers.get(cluster) ?? '',
          user: stringValue(context.user),
        }
      })
      .filter(context => context.name)
    return { contexts, currentContext, error: false }
  }
  catch {
    return { contexts: [], currentContext: '', error: true }
  }
}

export function selectSingleKubeconfigContext(value: string, contextName: string): string {
  const config = parseKubeconfigObject(value)
  const contexts = arrayValue(config.contexts)
  if (contexts.length <= 1) {
    const onlyContextName = stringValue(contexts[0]?.name)
    if (onlyContextName)
      config['current-context'] = onlyContextName
    return YAML.stringify(config)
  }

  const selectedContext = contexts.find(context => stringValue(context.name) === contextName)
  if (!selectedContext)
    throw new Error('selected kubeconfig context not found')

  const selectedContextBody = objectValue(selectedContext.context)
  const selectedClusterName = stringValue(selectedContextBody.cluster)
  const selectedUserName = stringValue(selectedContextBody.user)
  config.contexts = [selectedContext]
  config.clusters = arrayValue(config.clusters).filter(cluster => stringValue(cluster.name) === selectedClusterName)
  config.users = arrayValue(config.users).filter(user => stringValue(user.name) === selectedUserName)
  config['current-context'] = contextName
  return YAML.stringify(config)
}

function parseKubeconfigObject(value: string): KubeconfigObject {
  const parsed = YAML.parse(value)
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed))
    throw new Error('kubeconfig must be an object')
  return parsed as KubeconfigObject
}

function arrayValue(value: unknown): KubeconfigObject[] {
  return Array.isArray(value) ? value.filter(item => item && typeof item === 'object' && !Array.isArray(item)) as KubeconfigObject[] : []
}

function objectValue(value: unknown): KubeconfigObject {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as KubeconfigObject : {}
}

function stringValue(value: unknown): string {
  return typeof value === 'string' ? value : ''
}
