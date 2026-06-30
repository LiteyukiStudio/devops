# Deploy the Platform

This page does one thing: start Liteyuki DevOps with Docker Compose and open the console. If you want to deploy directly to Kubernetes, see [Helm Deployment](/en/guide/helm-deployment).

## Prepare

You need:

- A machine that can run Docker.
- Docker Compose.
- Network access to pull DockerHub images.
- Host port `8088` available.

## Choose an image tag

The full deployment path uses the repository root `docker-compose.yaml` and pulls these images by default:

```text
liteyukistudio/devops-api:nightly
liteyukistudio/devops-worker:nightly
```

To verify a specific release, set `DEVOPS_IMAGE_TAG` before starting:

```bash
DEVOPS_IMAGE_TAG=v0.1.0-rc.1 docker compose up -d
```

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

## Check services

```bash
docker compose ps
docker compose logs -f api
docker compose logs -f worker
```

When API is healthy, the console opens in the browser. When worker is healthy, builds, deployments, and status sync can run.

## Next

1. Open [Initialize Console](/en/guide/product) and create or sign in as an administrator.
2. Open [Connect Cluster and Registry](/en/guide/workspace) to prepare runtime and image storage.
3. Follow [Deploy a Web Project](/en/operations/deploy-web-project) to complete the first delivery path.

## Stop services

```bash
docker compose down
```

To remove data as well:

```bash
docker compose down -v
```

<div class="hint">
Start first, configure gradually. Do not configure every external system at once; the first goal is to enter the console.
</div>
