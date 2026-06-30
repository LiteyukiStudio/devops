---
description: Liteyuki DevOps documentation home for shipping applications from source code to reachable services.
pageType: home

hero:
  name: Liteyuki DevOps
  text: Code once, deploy anywhere.
  tagline: Start the platform first, then ship your project. Builds, releases, routes, and status stay in one clear path.
  actions:
    - theme: brand
      text: Get Started
      link: /guide/deployment/kubernetes-helm
    - theme: alt
      text: Deploy a Project
      link: /operations/deploy-web-project
  image:
    src: /brand/mascot-liteyuki-catgirl-alpha.webp
    alt: Liteyuki DevOps mascot
features:
  - title: Kubernetes (Helm)
    details: Use Helm to start API, worker, PostgreSQL, and Redis in Kubernetes or K3s.
    link: /guide/deployment/kubernetes-helm
  - title: Docker Compose
    details: Run docker compose up -d to start PostgreSQL, Redis, API, and worker on one machine.
    link: /guide/deployment/docker-compose
  - title: Deploy a web project
    details: Follow project space, application, deployment target, build, Release, and route to publish the example project.
    link: /operations/deploy-web-project
  - title: Connect external systems
    details: Configure runtime clusters, registries, Git providers, and secrets separately. Start small, then enable automation.
    link: /guide/workspace
  - title: Troubleshoot failed states
    details: Use build logs, Release logs, cluster events, and route checks to find where the delivery path failed.
    link: /operations/diagnostics
---
