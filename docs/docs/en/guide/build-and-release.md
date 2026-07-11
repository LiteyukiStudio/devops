# Platform Startup Problems

This page covers the Docker Compose problems most likely to block the platform from starting. For application builds or Kubernetes runtime failures, continue with the troubleshooting guide under Use.

## Run a specific image tag

The default `docker-compose.yaml` uses the `nightly` images. To verify an RC or stable release, set `DEVOPS_IMAGE_TAG` before starting:

```bash
DEVOPS_IMAGE_TAG=v0.1.0-rc.1 docker compose up -d
```

If you want to build images from the current source tree instead of pulling DockerHub images, use the source-build Compose file:

```bash
docker compose -f docker-compose-build.yaml up -d --build
```

## Port `8088` is occupied

Check the listener:

```bash
lsof -nP -iTCP:8088 -sTCP:LISTEN
```

You can stop the existing process or change the port mapping in `docker-compose.yaml`:

```yaml
ports:
  - "8089:8080"
```

Then visit `http://localhost:8089`.

## The page opens, but API calls fail

Check API logs:

```bash
docker compose logs -f api
```

Then confirm PostgreSQL and Redis are healthy:

```bash
docker compose ps
```

## Worker did not start

Check worker logs:

```bash
docker compose logs -f worker
```

The Worker handles builds, deployments, and status synchronization. The API alone is enough to browse the console, but the Worker must stay healthy before you can build or release an application.
