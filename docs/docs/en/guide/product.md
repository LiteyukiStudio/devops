# First Time in the Console

After the platform starts, complete the few settings you actually need. There is no need to connect every external system at once. If you can sign in and create a project space, you are ready to prepare a runtime.

## Sign in or bootstrap

Compose starts the API in development mode by default. Open the sign-in page and follow the account hint shown there.

If you switch to production mode, visit this page the first time:

```text
http://localhost:8088/bootstrap
```

Create the first administrator account on this page.

## Create the first project space

A project space keeps the applications, members, and runtime settings for one product or team together. Think of it as that product's workspace inside the platform.

Suggested first values:

| Field | Suggestion |
| --- | --- |
| Name | Product or team name |
| Slug | Lowercase English with hyphens |
| Members | Start with yourself, invite others later |

The project space list defaults to spaces related to the current user. Platform administrators can switch the scope to all project spaces when they need global maintenance.

## Create the first application

An application represents one independently deployable service. For the first run, create a basic application:

- Fill in name.
- Fill in a short lowercase slug.
- Leave runtime details for later.

Service ports, image settings, Dockerfile paths, environment variables, and data volumes belong to deployment targets. The application profile only keeps the name, slug, and icon.

## Next

Continue to [Connect Cluster and Registry](/en/guide/workspace).

If you already have an image, start with an existing-image deployment. It is the shortest path to verify the platform, cluster, and route.

If you want repository-based builds, configure Git providers, registries, and build settings afterward.
