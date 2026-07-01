# Status and Troubleshooting

Liteyuki DevOps keeps build, release, route, and runtime state close together. When something fails, first locate the stage.

## Build did not succeed

Check the build record:

- Dockerfile path.
- Build context.
- Dependency download failures.
- Registry push credentials.

When a build record shows `kubernetes build job failed`, the platform enriches the message with the build Pod status and recent Kubernetes Events. Common fields include `executor terminated`, `exitCode=137`, `OOMKilled`, `Evicted`, and `BackOff`. `exitCode=137` / `OOMKilled` usually means the build environment ran out of memory; increase the build environment size in the deployment target and retry.

If Git and registry connections are not ready yet, deploy an existing image first to verify the second half of the delivery path.

## Release did not succeed

Check Release status and deployment logs:

- Image exists.
- Runtime cluster is reachable.
- Image pull credential is correct.
- Service port matches the real application port.
- Environment variables, secrets, and config files match application expectations.

## Route is not reachable

Check:

- Domain resolves to the right entrypoint.
- Gateway API CRDs are installed, and GatewayClass/Gateway exist.
- HTTPRoute is Accepted, ResolvedRefs, and Programmed.
- Service points to the correct port.
- TLS settings match the gateway.
- Service endpoints have ready Pods.

For local test domains, start with hosts or `curl --resolve` before changing public DNS.

## Recovery suggestions

- Wrong config: edit the deployment target and release again.
- Wrong image: select the right image and create a new Release.
- App issue: inspect runtime logs before restart or rollback.
- Route issue: check route status first, then the cluster gateway.
