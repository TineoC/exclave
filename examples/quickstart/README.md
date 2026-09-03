# Quickstart — all four planes against a real cluster

```bash
just demo        # or ./demo.sh
just demo-clean
```

Needs `docker` (or OrbStack/Podman), `kind`, `kubectl`, `helm` and Go.

## What it actually does

| Step | Plane | What is proven |
|---|---|---|
| Builds two services, pushes them, reads digests **back from the registry** | artifact | Digests are observed, not asserted |
| Renders site values against `values.schema.json` | contract | Bad config fails at template time |
| Runs `exclave plan` over five environments | — | Each gets a different answer, with reasons |
| Packages the product chart and pushes it as an OCI artifact | distribution | The chart is a registry artifact like any other |
| Argo CD pulls that version into the cluster | reconciliation | Nothing is pushed at the cluster |

The five environments are contrived to exercise every outcome: a clean upgrade, an upgrade held
back by the upgrade path, one held back by a schema floor, one with nothing eligible at all, and
one already current.

## Layout

```
catalog/                 four releases, each declaring its own constraints
fleet/environments/      five environments, each a plain YAML file
charts/common/           library chart — the shape of a Deployment, defined once
charts/svc/              one service chart, aliased twice by the umbrella
charts/product/          the umbrella: what a customer installs, plus values.schema.json
services/svc/            ~30 lines of Go. The services are not the point.
reference/argocd/        wiring for customers running Argo CD
reference/flux/          wiring for customers running Flux
reference/gateway/       a sample Gateway for sites using routing.mode=gateway
```

## Routing

Nothing is routed by default. A site picks a mode and supplies its own hostname:

```bash
# a cluster running NGINX or Traefik
helm template acme charts/product \
  --set global.routing.mode=ingress \
  --set global.routing.ingress.className=nginx \
  --set auth-svc.route.enabled=true \
  --set auth-svc.route.host=auth.example.com

# a cluster with Gateway API — no vendor annotations reach this chart at all
helm template acme charts/product \
  --set global.routing.mode=gateway \
  --set global.routing.gateway.name=platform-gw \
  --set auth-svc.route.enabled=true \
  --set auth-svc.route.host=auth.example.com
```

`just contract` asserts both modes render and that the schema rejects a typo in either.

## If Argo CD does not converge

The demo says so and falls back to installing the same chart at the same resolved version
directly, rather than reporting success it did not achieve. Argo CD's internal gRPC is
unreliable on some local Docker backends; that is a property of the laptop, not of the design,
and planes 1–3 are unaffected either way.
