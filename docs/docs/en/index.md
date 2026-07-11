---
description: Luna DevOps documentation home for shipping applications from source code to reachable services.
pageType: home

hero:
  name: Luna DevOps
  text: Code once, deploy anywhere.
  tagline: Deploy your projects in a few clear steps with a DevOps platform built for small teams and businesses.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/deployment/kubernetes-helm
    - theme: alt
      text: Deploy a Project
      link: /operations/deploy-web-project
  image:
    src: /brand/mascot-luna-catgirl-alpha.webp
    alt: Luna DevOps mascot
features:
  - title: Kubernetes (Helm)
    details: Already running Kubernetes or K3s? Install API, Worker, PostgreSQL, and Redis together with Helm.
    link: /guide/deployment/kubernetes-helm
  - title: Docker Compose
    details: Want to try it on one machine first? Start the complete platform with one docker compose up -d.
    link: /guide/deployment/docker-compose
  - title: Deploy a web project
    details: Follow one complete example from project space and image build to release and public domain.
    link: /operations/deploy-web-project
  - title: Connect external systems
    details: Connect a runtime cluster and registry first, verify deployment, and add Git automation when you are ready.
    link: /guide/workspace
  - title: Troubleshoot failed states
    details: Work through build, release, cluster, and route checks to find where a failure started.
    link: /operations/diagnostics
---
