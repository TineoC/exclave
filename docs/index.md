# exclave

**A framework for delivering software into environments you cannot log into.**

Customer clouds, on-premise data centers, air-gapped networks, classified enclaves, ships,
edge devices. You ship the software; someone else runs the cluster, holds the credentials,
and decides when anything changes.

> One environment is a pipeline. Many environments are a control loop.

Nearly every mistake in this space comes from scaling the first model instead of switching to
the second: a promotion job per environment, a values branch per customer, a release checklist
per enclave. Past a handful of targets, environments become **data**, releases become a
**catalog**, and deployment becomes a **solver plus agents**.

## The four planes

1. **[Artifact](01-artifact.md)** — what you ship is immutable, signed and self-describing
2. **[Contract](02-contract.md)** — configuration is a versioned, schema-validated API
3. **[Distribution](03-distribution.md)** — artifacts cross boundaries as verified copies
4. **[Reconciliation](04-reconciliation.md)** — an agent inside the environment pulls and converges

Plus **[the constraint resolver](constraints.md)**, which is the piece no existing open source
project provides.

## Architecture

Three figures — the boundary, the four planes, and how the resolver decides.
**[Architecture →](architecture.md)**

## Decisions

The [architecture decision records](adr/) capture the calls that were reversed while this was
being designed, and why. Those are the pages worth reading if you are adapting the framework
rather than adopting it:

- [ADR-0001 — Pull, never push](adr/0001-pull-never-push.md)
- [ADR-0002 — The umbrella chart is the product release unit](adr/0002-umbrella-as-release-unit.md)
- [ADR-0003 — The registry is the version source of truth](adr/0003-registry-not-tags.md)
- [ADR-0004 — CI never writes to the repository it builds](adr/0004-ci-never-writes-source.md)
- [ADR-0005 — Environments are data, not pipelines](adr/0005-environments-are-data.md)

## Try it

```bash
git clone https://github.com/TineoC/exclave && cd exclave
just plan     # resolve the example fleet, no cluster needed
just demo     # all four planes against a real kind cluster
```

[Source on GitHub](https://github.com/TineoC/exclave) · MIT licensed
