# ADR-0002 — The umbrella chart is the product release unit

**Status:** accepted, reversing an earlier decision

## Context

Standard advice — including the advice this framework started with — is that a Helm umbrella
chart aggregating many services is a **local convenience only**, never a release unit, because
it recouples services that were deliberately decoupled and forces them onto one release
cadence.

That advice is correct **for a platform team deploying its own services**. It is wrong for a
vendor shipping a product into environments it does not control.

## Decision

Version at two levels:

- **Internally** — per-service versions, independent build cadence, independent CI.
- **Externally** — one product version pinning a *tested combination* of service versions.

The umbrella chart carries the external version and is the artifact a customer installs.

## Consequences

- A customer installs one thing, approves one number in a change ticket, and accredits one
  baseline. They never learn that `auth-svc` is at 1.4.3.
- The tested combination is what you support. Never let a site mix and match service versions;
  that is an unbounded support matrix wearing a disguise.
- Internal decoupling is preserved — services still build and version independently. Only the
  release artifact is unified.

## When this flips

If you are a platform team deploying your own services into your own clusters, go back to the
original advice: per-service charts, per-service releases, and an umbrella only for spinning up
a local stack.

The distinguishing question is **who installs it**. If that is someone outside your
organization, they need one version number.
