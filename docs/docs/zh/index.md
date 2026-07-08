---
description: Liteyuki DevOps 文档首页，帮助个人开发者和小团队完成从代码到上线的应用交付。
pageType: home

hero:
  name: Liteyuki DevOps
  text: Code once, deploy anywhere.
  tagline: 仅需要轻松几步即可简单部署自己的项目（面向小型团队和企业的DevOps解决方案）
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/deployment/kubernetes-helm
    - theme: alt
      text: 部署项目
      link: /operations/deploy-web-project
  image:
    src: /brand/mascot-liteyuki-catgirl-alpha.webp
    alt: Liteyuki DevOps mascot
features:
  - title: Kubernetes (Helm) 部署
    details: 用 Helm 在 Kubernetes 或 K3s 中启动 API、worker、PostgreSQL 和 Redis。
    link: /guide/deployment/kubernetes-helm
  - title: Docker Compose 部署
    details: 用 docker compose up -d 在单机上启动 PostgreSQL、Redis、API 和 worker。
    link: /guide/deployment/docker-compose
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
