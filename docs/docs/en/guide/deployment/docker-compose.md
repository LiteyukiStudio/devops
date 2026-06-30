# Docker Compose Deployment

Docker Compose is the quickest way to start Liteyuki DevOps. Use it for personal servers, test environments, and small-team trials.

If you want the platform itself to run in Kubernetes, start with [Kubernetes (Helm)](/en/guide/deployment/kubernetes-helm).

## Prepare

You need:

- A machine that can run Docker.
- Docker Compose.
- Network access to pull DockerHub images.
- Host port `8088` available.

## Choose A Version

The repository root `docker-compose.yaml` pulls these images by default:

```text
liteyukistudio/devops-api:nightly
liteyukistudio/devops-worker:nightly
```

To verify a specific release, set the image tag before starting:

```bash
DEVOPS_IMAGE_TAG=v0.1.0-rc.1 docker compose up -d
```

## Start

Run this from the repository root:

```bash
docker compose up -d
```

This starts PostgreSQL, Redis, API, and worker. The API image already embeds the web console, so you do not need to start Vite separately.

To build images from the current source tree:

```bash
docker compose -f docker-compose-build.yaml up -d --build
```

## Open The Console

Visit:

```text
http://localhost:8088
```

The default Compose stack only exposes API on host port `8088`. PostgreSQL and Redis stay inside the container network and do not occupy host ports `5432` and `6379`.

## Check Services

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

## Stop

```bash
docker compose down
```

To remove data as well:

```bash
docker compose down -v
```

<div class="hint">
Start first, configure gradually. The first goal is to enter the console, not to connect every external system at once.
</div>
