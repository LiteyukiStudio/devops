# Quick Docker Compose Deploy

This page does one thing: get Liteyuki DevOps running and open the console.

## Prepare environment files

The current full deployment path uses the repository root `docker-compose.yaml`. Prepare the worker environment file first:

```bash
cp .env.worker.example .env.worker
```

If you already have Kubernetes, registry, or build settings, you can edit `.env.worker` later. For a first run, keep the defaults.

## Start the platform

Run this from the repository root:

```bash
docker compose up -d --build
```

This starts PostgreSQL, Redis, API, and worker. The API image already embeds the web console, so you do not need to start Vite separately.

## Open the console

Visit:

```text
http://localhost:8088
```

The default Compose stack exposes the API on host port `8088`. PostgreSQL and Redis stay inside the Compose network, so they do not occupy host ports `5432` and `6379`.

## Stop services

```bash
docker compose down
```

To remove data as well:

```bash
docker compose down -v
```

<div class="hint">
Start first, configure gradually. The docs should reduce effort, not ask you to memorize platform terms upfront.
</div>
