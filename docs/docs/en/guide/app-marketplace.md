# App Marketplace

The app marketplace installs common infrastructure apps into project spaces from templates. The MVP includes Redis, PostgreSQL, MySQL, and RabbitMQ for quick cache, database, and queue setup.

Installing a template creates:

- An application.
- An image-based deployment target.
- Template-defined environment variables, secret variables, and runtime data volumes.
- An optional first Release; deployment is enabled by default.

Secret parameters are written to the platform secret store. Deployment targets keep secret references only, and plaintext values are not echoed back to the frontend.

## Install Flow

1. Open “App Marketplace”.
2. Pick a template and click “Install”.
3. Select a project space and confirm the application name, slug, runtime cluster, CPU, memory, replicas, and data capacity.
4. Fill in template parameters. Auto-generated passwords can be left empty; the backend generates them.
5. Keep “Deploy after install” enabled, or disable it and release manually from the application deployment page later.

After a successful install, the page navigates to the new application's deployment tab.

## Current Limits

The MVP only enables templates whose images can run with their default command. Images that require custom command or args are not installable yet; MinIO will be enabled after deployment targets support startup commands.

Marketplace templates are loaded from backend-embedded JSON. Future third-party marketplaces can reuse the same schema, with backend-side fetching, validation, and caching.
