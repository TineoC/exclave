# ADR-0001 — Pull, never push

**Status:** accepted

## Context

The obvious way to deploy is to connect to the target and apply. It is how nearly every CI
pipeline starts life, and it works right up until the target is a customer's cluster in a
facility you have no route to.

## Decision

Publish declarative intent. An agent running *inside* the environment pulls it, verifies it,
and converges. Nothing is ever pushed at a cluster from outside.

## Consequences

- No inbound network path is required, which is the only way this works in an enclave.
- Credentials to the target live in the target, not in your CI system. A compromise of your
  pipeline does not hand over customer clusters.
- Reconciliation is continuous, so drift is corrected rather than discovered.
- You give up synchronous deploy feedback. The pipeline knows a version was *published*, not
  that it was *applied* — which is a truthful reflection of who is actually in control.

## When this flips

Never, for environments you do not own. For your own low-side dev clusters a direct apply is
fine, and pretending otherwise is ceremony.
