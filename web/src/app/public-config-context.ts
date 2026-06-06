import { createContext, use } from 'react'

export const defaultPublicConfigs: Record<string, string> = {
  'site.title': 'Liteyuki DevOps',
  'site.logoUrl': '',
  'site.faviconUrl': '',
  'site.loginSubtitle': '使用本地账号登录控制台',
}

export const PublicConfigContext = createContext(defaultPublicConfigs)

export function usePublicConfig() {
  return use(PublicConfigContext)
}
