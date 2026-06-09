import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'

import enUS from './locales/en-US'
import zhCN from './locales/zh-CN'

const resources = {
  'zh-CN': { translation: zhCN },
  'en-US': { translation: enUS },
}

function detectBrowserLanguage() {
  const languages = [localStorage.getItem('liteyuki-language'), ...navigator.languages, navigator.language].filter(Boolean)
  return languages.some(language => language?.toLowerCase().startsWith('en')) ? 'en-US' : 'zh-CN'
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
