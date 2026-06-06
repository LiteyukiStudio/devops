import i18next from 'i18next'
import { initReactI18next } from 'react-i18next'

i18next.use(initReactI18next).init({
  lng: 'zh-CN',
  fallbackLng: 'zh-CN',
  interpolation: {
    escapeValue: false,
  },
  resources: {
    'zh-CN': {
      translation: {
        appName: 'Liteyuki DevOps',
        projects: '项目',
        applications: '应用',
        accessTokens: 'Access Token',
        security: '账号安全',
        authProviders: '身份源',
        siteSettings: '站点设置',
        users: '用户',
        login: '登录',
        logout: '退出',
      },
    },
    'en-US': {
      translation: {
        appName: 'Liteyuki DevOps',
        projects: 'Projects',
        applications: 'Applications',
        accessTokens: 'Access Token',
        security: 'Security',
        authProviders: 'Auth Providers',
        siteSettings: 'Site Settings',
        users: 'Users',
        login: 'Login',
        logout: 'Logout',
      },
    },
  },
})

export default i18next
