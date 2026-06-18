# Startup Troubleshooting

This page covers the most common Docker Compose startup issues. Deeper build and cluster issues are covered in the Use section.

## Verify a specific image tag

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

## Page opens but API calls fail

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

The worker handles builds, deployments, and status sync. You can browse the console with only the API running, but release workflows need the worker.
