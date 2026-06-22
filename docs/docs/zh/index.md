---
description: Liteyuki DevOps 文档首页，帮助个人开发者和小团队完成从代码到上线的应用交付。
pageType: home

hero:
  name: Liteyuki DevOps
  text: Code once, deploy anywhere.
  tagline: 先把平台跑起来，再把项目部署出去。构建、发布、访问入口和状态都在一条清晰路径里。
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: 部署项目
      link: /operations/deploy-web-project
  image:
    src: /brand/mascot-liteyuki-catgirl-alpha.webp
    alt: Liteyuki DevOps mascot
features:
  - title: 部署平台
    details: 用 docker compose up -d 启动 PostgreSQL、Redis、API 和 worker，打开 localhost:8088 进入控制台。
    link: /guide/getting-started
  - title: 部署一个 Web 项目
    details: 按项目空间、应用、部署配置、构建、Release 和访问入口的顺序，把示例项目发布上线。
    link: /operations/deploy-web-project
  - title: 连接外部系统
    details: 运行集群、镜像站、Git Provider 和密钥分开配置，先连最少依赖，再逐步打开自动化。
    link: /guide/workspace
  - title: 排查失败状态
    details: 从构建日志、Release 日志、集群事件和访问入口检查入手，快速定位问题发生在哪一段。
    link: /operations/diagnostics
---
