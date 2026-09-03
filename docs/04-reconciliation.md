# Plane 4 — Reconciliation

**Invariant:** an agent *inside* the environment pulls desired state and converges. You publish
intent. You never push.

You have no network path into a customer enclave, and they do not want to give you one. That
constraint is not an obstacle to work around — it is the design.

## Environments are data

```yaml
name: customer-a-prod
tier: production
classification: il6
channel: stable
kubernetes: "1.29"
schema: 9
current: "3.1.4"
maintenanceWindow: "Sun 02:00-06:00 UTC"
```

One file per place the product runs. This file is *everything the resolver knows* — there is no
API call to make, because there is nothing to call.

Working example:
[`examples/quickstart/fleet/environments/`](https://github.com/TineoC/exclave/tree/main/examples/quickstart/fleet/environments)

## Do not assume the agent

Some customers run Argo CD. Some run Flux. Some run neither and will pick one this quarter. The
**chart is the product**; the controller wiring is reference material you provide for both:

```yaml
# Flux
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
spec:
  chart:
    spec:
      chart: acme-platform
      version: "3.2.0"
  valuesFrom:
    - kind: ConfigMap
      name: acme-site-values      # theirs, never yours
```

```yaml
# Argo CD
apiVersion: argoproj.io/v1alpha1
kind: Application
spec:
  source:
    repoURL: registry.customer.internal/charts
    chart: acme-platform
    targetRevision: "3.2.0"
```

Both wirings ship in
[`examples/quickstart/reference/`](https://github.com/TineoC/exclave/tree/main/examples/quickstart/reference).

## Their promotion, not yours

You deliver a version. Their change control decides when it reaches production. A GitOps
repository seeded from your template gives them dev/staging/prod structure, but **they own the
promotion commits** — which is also what makes the audit trail theirs to show an assessor.

## One environment is a pipeline; many are a control loop

At one target, a promotion job works. At forty it does not: you cannot run a job per
environment per service and keep it comprehensible. Publish desired state, let agents converge,
and let the [resolver](constraints.md) decide what each one is eligible for.

## Skip this plane and

You need inbound network access to customer environments — the one thing you are never getting.
