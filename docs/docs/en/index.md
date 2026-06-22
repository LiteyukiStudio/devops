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
      link: /guide/getting-started
    - theme: alt
      text: Deploy a Project
      link: /operations/deploy-web-project
  image:
    src: /brand/mascot-liteyuki-catgirl-alpha.webp
    alt: Liteyuki DevOps mascot
features:
  - title: Deploy the platform
    details: Run docker compose up -d to start PostgreSQL, Redis, API, and worker, then open localhost:8088.
    link: /guide/getting-started
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
