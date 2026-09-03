# ADR-0004 — CI never writes to the repository it builds

**Status:** accepted

## Context

A promotion is a commit. The convenient place to put that commit is the repository the pipeline
already has checked out — which quietly grants CI write access to your source code, and makes
every deploy rewrite application history.

## Decision

Two repositories:

- **Source** — application code and charts. CI reads only.
- **GitOps** — deploy state: one small file per service per environment, holding a pinned
  version. CI writes here, and nowhere else.

## Consequences

- The pipeline's single long-lived write credential reaches a repository that contains no code.
  Scope it to that repository, mask it, protect it.
- Application history stays free of deploy noise, and the deploy history becomes a clean,
  attributable audit trail — which is exactly the artifact an assessor asks for
  (NIST 800-53 AU-2, AU-12).
- Promotion commits need `[skip ci]`, or the GitOps repository will build itself in a loop the
  first time it gets a pipeline. This is not optional.
- Permissions become git-native: `CODEOWNERS` on `envs/prod/` plus protected branches means
  production promotion requires review by the right people, on any GitLab or GitHub tier.

## When this flips

Never, but note the layer it does not cover: if engineers have UI access to the reconciliation
agent with sync rights, they can bypass all of this. Argo CD RBAC or its Flux equivalent is a
required third layer, not an optional one.
