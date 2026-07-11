import * as path from 'node:path';
import { defineConfig } from '@rspress/core';

export default defineConfig({
  root: path.join(__dirname, 'docs'),
  title: 'Liteyuki DevOps',
  description: 'Deploy and operate applications in a few clear steps, built for small teams and businesses.',
  lang: 'zh',
  icon: '/liteyuki-logo.svg',
  logo: {
    light: '/liteyuki-logo.svg',
    dark: '/liteyuki-logo.svg',
  },
  locales: [
    {
      lang: 'zh',
      label: '简体中文',
      title: 'Liteyuki DevOps',
      description: '轻松几步部署和管理应用，面向小型团队和企业的 DevOps 解决方案。',
    },
    {
      lang: 'en',
      label: 'English',
      title: 'Liteyuki DevOps',
      description: 'Deploy and operate applications in a few clear steps, built for small teams and businesses.',
    },
  ],
  themeConfig: {
    socialLinks: [
      {
        icon: 'github',
        mode: 'link',
        content: 'https://github.com/LiteyukiStudio/devops',
      },
    ],
  },
});
