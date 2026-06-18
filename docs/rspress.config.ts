import * as path from 'node:path';
import { defineConfig } from '@rspress/core';

export default defineConfig({
  root: path.join(__dirname, 'docs'),
  title: 'Liteyuki DevOps',
  description: 'A friendly DevOps delivery platform for individual developers and small teams.',
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
      description: '面向个人开发者和小团队的 DevOps 应用交付平台。',
    },
    {
      lang: 'en',
      label: 'English',
      title: 'Liteyuki DevOps',
      description: 'A friendly DevOps delivery platform for individual developers and small teams.',
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
