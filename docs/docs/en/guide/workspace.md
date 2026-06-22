# Connect Cluster and Registry

After the platform is running, you do not need every external integration immediately. Follow this order to avoid loops.

```text
Runtime cluster -> Registry -> Deployment target -> Route
```

## 1. Configure a runtime cluster

The runtime cluster is where applications are deployed. It can be Kubernetes or a lightweight K3s cluster.

For the first integration, prepare a test cluster and make sure its kubeconfig can be reached from the API and worker containers.

Before deleting a runtime cluster, migrate or delete deployment targets that reference it. The platform does not automatically delete deployment targets when a cluster is deleted.

## 2. Configure a registry

The registry stores build outputs and can also provide existing images for deployment.

If you only want to explore deployment, start with an existing image. Add push credentials when you are ready for the full build path.

## 3. Create a deployment target

A deployment target answers how this application should ship:

- Existing image or repository build.
- Runtime cluster.
- Stage, such as development, test, staging, or production.
- Runtime replicas, CPU, and memory.
- Build Job CPU and memory.
- Service port.
- Environment variables, secrets, config files, and data volumes.

If you only want to verify the platform first, use an existing image. After the cluster and route work, connect Git providers and automated builds.

## 4. Create a route

A route connects domain, path, TLS, and backend service. After creating one, check its status in the console, then verify it with a browser or `curl`.

## 5. Enable automation later

After the first path works, enable these gradually:

- Git providers and Git account authorization.
- Repository binding and webhooks.
- Automatic builds.
- Auto deploy after successful builds.
- Custom domains and HTTPS.
