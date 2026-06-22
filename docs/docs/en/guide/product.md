# Initialize Console

After the platform starts, complete the smallest useful setup. The goal is not to configure every integration at once. The goal is to sign in, create a project space, and continue to runtime setup.

## Sign in or bootstrap

Compose starts the API in development mode by default. Open the sign-in page and follow the account hint shown there.

If you switch to production mode, visit this page the first time:

```text
http://localhost:8088/bootstrap
```

Use it to create the first administrator account.

## Create the first project space

A project space is a workspace for teams, applications, deployment targets, and runtime resources.

Suggested first values:

| Field | Suggestion |
| --- | --- |
| Name | Product or team name |
| Slug | Lowercase English with hyphens |
| Members | Start with yourself, invite others later |

The project space list defaults to spaces related to the current user. Platform administrators can switch the scope to all project spaces when they need global maintenance.

## Create the first application

An application represents a deployable service. For the first run, create an empty application:

- Fill in name.
- Fill in a short lowercase slug.
- Leave runtime details for later.

Service port, image, Dockerfile, environment variables, and data volumes are maintained in deployment targets, not in the application metadata form.

## Next

Continue to [Connect Cluster and Registry](/en/guide/workspace).

If you already have an image, start with an existing-image deployment. It is the shortest path to verify the platform, cluster, and route.

If you want repository-based builds, configure Git providers, registries, and build settings afterward.
