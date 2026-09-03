# ADR-0005 — Environments are data, not pipelines

**Status:** accepted, reversing an earlier decision

## Context

The first design gave each environment its own promotion jobs, wired with explicit
dependencies: deploy to dev, verify, deploy to staging, verify, gate, deploy to prod. It is
readable and it works — for one environment.

At forty environments it collapses. You cannot maintain a job per environment per service, you
cannot read the resulting pipeline, and every new customer edits CI configuration.

## Decision

Environments are **descriptor files**. Releases are a **catalog**. Deployment is a **resolver
plus agents**. Adding an environment means adding a YAML file; no pipeline changes.

## Consequences

- Onboarding an environment is data entry, not CI surgery.
- The fleet's state is greppable and diffable. "Which sites are still on 3.0?" is a query, not
  an investigation.
- You need a resolver, and you will write it yourself — no open source project provides one.
  See [the constraint resolver](../constraints.md).
- Per-environment special cases have to be expressed as constraints rather than as bespoke
  pipeline steps. This is a feature: it forces the special case to be named.

## When this flips

Below roughly five stable environments the explicit pipeline is genuinely simpler and you
should keep it. The switch is worth making when you start copy-pasting a promotion job, or the
first time someone asks a fleet-wide question you cannot answer quickly.
