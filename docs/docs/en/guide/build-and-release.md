# Startup Troubleshooting

This page covers the most common Docker Compose startup issues. Deeper build and cluster issues are covered in the Use section.

## `.env.worker` is missing

If startup reports `env file .env.worker not found`, run:

```bash
cp .env.worker.example .env.worker
docker compose up -d --build
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
