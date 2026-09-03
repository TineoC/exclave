# Plane 2 — Contract

**Invariant:** configuration is a versioned, schema-validated API between you and the
environment. Everything site-specific is injected; nothing is baked in.

This is the highest-leverage decision in the whole framework. Your values file will outlive
your architecture, your CI system and probably your employer.

## The surface should fit on one page

Six values, and every one of them is a place where one platform differs from another:

```
registry prefix · pull secrets · routing (mode + class/gateway)
proxy settings · trust bundle / PKI · per-service resources
```

Everything else gets a working default. **If a customer needs to understand your internals to
install the product, the contract has failed.**

Two values a real product would add and this example deliberately does not: a **storage class**,
which needs a stateful component to be worth demonstrating, and an **OIDC issuer**, which needs an
auth flow. Shipping either here would mean a value nothing consumes — speculative configuration,
which is the thing this framework argues against. Add them when something actually reads them.

## Routing is the sharpest seam

Where traffic enters is the single most platform-specific thing about a Kubernetes workload, so it
gets a mode switch rather than a hardcoded answer:

```yaml
global:
  routing:
    mode: gateway              # none | ingress | gateway
    gateway:
      name: platform-gw
      namespace: gateways
```

- **`ingress`** — ubiquitous, works on every cluster in service today, but pushes platform
  differences into vendor-specific annotations. Fine, and often the only option on an older
  cluster.
- **`gateway`** — emits a Gateway API `HTTPRoute`. The platform team owns a `Gateway` object; the
  workload chart references it by name and carries **no cloud-specific strings at all**. This is
  the portable choice wherever the cluster has it: GKE Gateway, Istio, Envoy Gateway, Cilium,
  NGINX Gateway Fabric and on-prem implementations all consume the same `HTTPRoute`.
- **`none`** — the default. Nothing is rendered.

The same per-service values (`route.enabled`, `route.host`) drive both. A site moves from an
on-prem NGINX Ingress to a cloud Gateway by changing one word, and no service chart is touched.

## Validate it, so failures land early

A schema turns a 2am outage into a `helm template` error:

```console
$ helm template acme charts/product --set global.registry=""
Error: values don't meet the specifications of the schema(s) in the following chart(s):
acme-platform:
- at '/global/registry': minLength: got 0, want 1

$ helm template acme charts/product --set auth-svc.image.digest="sha256:nothex"
Error: values don't meet the specifications of the schema(s):
- at '/auth-svc/image/digest': 'sha256:nothex' does not match pattern '^(sha256:[a-f0-9]{64})?$'
```

Working example:
[`charts/product/values.schema.json`](https://github.com/TineoC/exclave/blob/main/examples/quickstart/charts/product/values.schema.json)

`global.registry` is **required with a minimum length of one**. An install must state where its
images came from; that is not a field worth defaulting.

## Registry indirection on every image

The single most-copied pattern from Big Bang, and non-negotiable for air-gapped delivery. Every
image reference resolves through one prefix so customers mirror into whatever they run:

```
{{- define "common.image" -}}
{{- $reg := .Values.global.registry | default "" -}}
{{- if .Values.image.digest -}}
{{- $ref = printf "%s@%s" $repo .Values.image.digest -}}
{{- else if .Values.image.tag -}}
{{- $ref = printf "%s:%s" $repo .Values.image.tag -}}
{{- else -}}
{{- fail "set either image.digest (preferred) or image.tag" -}}
{{- end -}}
```

The chart refuses to render without an image reference. Failing loudly at template time beats
a pod stuck in `ImagePullBackOff` in a facility you cannot reach.

## Treat it like the API it is

Semver the contract. Never break it silently. A removed value is a breaking change even though
nothing failed to compile.

## Skip this plane and

Every environment becomes a snowflake, and your support matrix is unbounded across N enclaves
at M versions. That, rather than architecture, is what actually kills vendors in this space.
