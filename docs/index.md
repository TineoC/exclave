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

## Portability

Cloud-platform agnostic by construction. The resolver depends on a semver parser and a YAML
parser; the charts contain no cloud SDKs, LoadBalancer annotations, storage classes or ingress
classes.

- **Requires:** Kubernetes 1.28+, any OCI registry, any GitOps controller (Argo CD or Flux).
- **Does not require:** any cloud service, a managed control plane, a specific CI system, a
  service mesh, or network egress at install time.
- **Runs on:** EKS, GKE, AKS, OpenShift, RKE2, k3s at the edge, bare metal, and air-gapped
  enclaves.

What differs between platforms is a site value, never a chart edit — set `global.routing.mode` to
`ingress` or `gateway` and the same service charts land on either. See
[the contract plane](02-contract.md).

## Using it for real

Holding a real fleet safely — what is sensitive, why the aggregate is worse than any one
descriptor, the classification ceiling, signed catalogs and redacted roll-ups.
**[Operating →](operating.md)** · **[Adding a contract →](onboarding.md)** ·
**[Known gaps →](gaps.md)**

## Running a portfolio

Delivering one product to many environments at once — contracts, impact levels, compliance as
constraints, and inherited accreditation. **[The portfolio pattern →](portfolio.md)**

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
