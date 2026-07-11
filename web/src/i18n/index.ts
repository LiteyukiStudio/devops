import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'

import enUS from './locales/en-US'
import zhCN from './locales/zh-CN'

const resources = {
  'zh-CN': { translation: zhCN },
  'en-US': { translation: enUS },
}

type SupportedLanguage = 'zh-CN' | 'en-US'

function detectBrowserLanguage() {
  const storedLanguage = normalizeLanguage(localStorage.getItem('luna-devops-language'))
  if (storedLanguage)
    return storedLanguage

  const browserLanguages = [...navigator.languages, navigator.language].filter(Boolean)
  return browserLanguages.map(normalizeLanguage).find(Boolean) ?? 'zh-CN'
}

function normalizeLanguage(language?: string | null): SupportedLanguage | undefined {
  const normalized = language?.trim().toLowerCase()
  if (!normalized)
    return undefined
  if (normalized === 'zh-cn' || normalized === 'zh' || normalized.startsWith('zh-'))
    return 'zh-CN'
  if (normalized === 'en-us' || normalized === 'en' || normalized.startsWith('en-'))
    return 'en-US'
  return undefined
}

i18next.use(initReactI18next).init({
  lng: detectBrowserLanguage(),
  fallbackLng: 'zh-CN',
  interpolation: {
    escapeValue: false,
  },
  resources,
})

export default i18next
