# Plane 2 — Contract

**Invariant:** configuration is a versioned, schema-validated API between you and the
environment. Everything site-specific is injected; nothing is baked in.

This is the highest-leverage decision in the whole framework. Your values file will outlive
your architecture, your CI system and probably your employer.

## The surface should fit on one page

```
registry prefix · pull secrets · trust bundle / PKI · OIDC issuer
ingress hostnames · storage class · proxy settings · resource tier
```

Everything else gets a working default. **If a customer needs to understand your internals to
install the product, the contract has failed.**

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
