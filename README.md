# exclave

**A framework for delivering software into environments you cannot log into.**

Customer clouds, on-premise data centers, air-gapped networks, classified enclaves,
ships, edge devices. You ship the software; someone else runs the cluster, holds the
credentials, and decides when anything changes.

Four planes, and the constraint resolver that ties them together.

```
ENVIRONMENT      CURRENT  TARGET  STATUS
customer-a-prod  3.1.4    3.2.0   upgrade
customer-b-prod  3.0.1    3.1.4   upgrade (3.2.0 blocked: requires platform >=3.1, installed 3.0.1)
customer-c-prod  2.9.0    3.0.0   upgrade (3.2.0 blocked: schema 6 < 7)
customer-d-prod  2.8.0    —       no eligible release (kubernetes 1.26 not in >=1.28 <1.32)
internal-lab     3.3.0    3.3.0   current
```

**The parenthetical is the whole point.** "No eligible release" without a reason is the
failure mode every system of this kind eventually hits.

## The mental shift

> One environment is a pipeline. Many environments are a control loop.

Nearly every mistake in this space comes from scaling the first model instead of switching
to the second: a promotion job per environment, a values branch per customer, a release
checklist per enclave. Past a handful of targets, environments become **data**, releases
become a **catalog**, and deployment becomes a **solver plus agents**.

## The four planes

| Plane | Invariant | Skip it and |
|---|---|---|
| **[Artifact](docs/01-artifact.md)** | What you ship is immutable, signed, and self-describing — it carries its own rules about where it may run | You cannot answer "what is actually running in enclave 7?" |
| **[Contract](docs/02-contract.md)** | Configuration is a versioned, schema-validated API between you and the environment | Every environment becomes a snowflake and the support matrix is unbounded |
| **[Distribution](docs/03-distribution.md)** | Artifacts cross boundaries as verified copies; no runtime egress, ever | Installs fail at the worst possible moment because something reached for the internet |
| **[Reconciliation](docs/04-reconciliation.md)** | An agent *inside* the environment pulls desired state and converges | You need network access into customer environments, which is the thing you cannot have |

Full write-up: **[the documentation site](https://tineoc.github.io/exclave/)** · diagrams: **[Architecture](https://tineoc.github.io/exclave/architecture.html)**.

## Try it

```bash
just plan                       # resolve the example fleet, no cluster needed
just explain customer-c-prod    # every constraint, release by release
just demo                       # all four planes against a real kind cluster
just demo-clean
```

`just demo` builds two services, captures their **real digests from the registry**, validates
site values against a JSON schema, resolves which release each environment is eligible for,
publishes the product chart as an OCI artifact, and has Argo CD pull it into a namespace.
Nothing is pushed at the cluster.

## The piece you would otherwise build yourself

`exclave` is the resolver. Everything else in this repository is assembled from tools that
already exist — Helm, OCI registries, Cosign, Zarf, Argo CD or Flux. The resolver is the gap:
given a fleet and a catalog, decide what each environment is eligible for and **explain every
exclusion in terms an operator can act on**.

```
exclave plan                          newest eligible release per environment
exclave explain <env> [version]       every constraint, checked and reported
exclave validate                      structural check of catalog and fleet
```

Constraints are deliberately few — semver ranges on Kubernetes, a monotonic schema floor, an
upgrade-path floor, tier and classification lists, and a channel ladder. Maintenance windows
are *reported, never scheduled*: this is a resolver, not a cron system. See
[docs/constraints.md](docs/constraints.md).

## Prior art

The pattern is not new; the vendor-neutral assembly of it is. [Big Bang](https://github.com/DoD-Platform-One/bigbang)
is the DoD's instance, [Zarf](https://zarf.dev/) is the transport slice, and Palantir's Apollo
is a mature commercial one — it
[publishes declarative instructions that agents inside each environment pull, verify and apply](https://blog.palantir.com/palantir-apollo-orchestration-constraint-based-continuous-deployment-for-modern-architectures-cdf42da19ba4).
That three independent systems converged on the same shape is the reason to trust it.

None of them is portable across employers, and none publishes the resolver. This is that,
in about 400 lines of Go and a set of conventions you can carry anywhere.

## Requirements

Go 1.24+ for the resolver. `just`, `helm`, `kind`, `kubectl` and a container runtime for the
demo. `zarf` and `cosign` for the air-gap example.

## License

MIT — see [LICENSE](LICENSE).
