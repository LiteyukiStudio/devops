# Quick Docker Compose Deploy

This page does one thing: get Liteyuki DevOps running and open the console.

## Choose an image tag

The current full deployment path uses the repository root `docker-compose.yaml` and pulls DockerHub images by default:

```text
liteyukistudio/devops-api:nightly
liteyukistudio/devops-worker:nightly
```

To verify a specific release, set `DEVOPS_IMAGE_TAG` before starting, for example `v0.1.0-rc.1`.

## Start the platform

Run this from the repository root:

```bash
docker compose up -d
```

This starts PostgreSQL, Redis, API, and worker. The API image already embeds the web console, so you do not need to start Vite separately.

If you want to build images from the current source tree instead of pulling DockerHub images, run:

```bash
docker compose -f docker-compose-build.yaml up -d --build
```

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
