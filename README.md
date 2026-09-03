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

## Running a portfolio

Delivering to many environments at once — a defense contractor's contracts, a vendor's customers,
a fleet of edge sites. Environments become data, releases become a catalog, and compliance becomes
constraints the resolver checks:

```console
$ just portfolio
ENVIRONMENT   CURRENT  TARGET      STATUS
army-abc-il5  4.0.3    4.2.1       upgrade (4.3.0 blocked: requires platform >=4.2, installed 4.0.3)
corp-lowside  4.3.0    4.3.0       current (4.4.0-rc.1 blocked: 2 critical CVEs, environment allows 0)
dla-ghi-il4   3.9.0    —           no eligible release (kubernetes 1.26 not in >=1.28 <1.32)
navy-xyz-il4  4.2.1    4.3.0       upgrade
usaf-def-il6  4.2.1    4.2.1       pinned (pinned, already installed)
```

Capabilities (`stigProfile`, `fips`, whatever your obligations are) match exactly; critical CVE
counts are a ceiling, and a release with **no scan result is refused rather than assumed clean**.
Adding an obligation is a YAML key, not a code change.

Holding a real fleet safely is its own problem: a directory of site descriptors maps which sites run
known-vulnerable versions and when each is unattended, which is more sensitive than any one file in
it. So the catalog is shareable and **the fleet is not** — the resolver runs inside each boundary, a
classification ceiling stops the low side ingesting what it must not hold, the catalog is signed, and
the roll-up crosses boundaries redacted, released by a human rather than a pipe.

See [the portfolio pattern](https://tineoc.github.io/exclave/portfolio.html),
[operating it for real](https://tineoc.github.io/exclave/operating.html),
[adding a contract](https://tineoc.github.io/exclave/onboarding.html),
[known gaps](https://tineoc.github.io/exclave/gaps.html), and
[`examples/contracts/`](examples/contracts/).

## Prior art

The pattern is not new; the vendor-neutral assembly of it is. [Big Bang](https://github.com/DoD-Platform-One/bigbang)
is the DoD's instance, [Zarf](https://zarf.dev/) is the transport slice, and Palantir's Apollo
is a mature commercial one — it
[publishes declarative instructions that agents inside each environment pull, verify and apply](https://blog.palantir.com/palantir-apollo-orchestration-constraint-based-continuous-deployment-for-modern-architectures-cdf42da19ba4).
That three independent systems converged on the same shape is the reason to trust it.

None of them is portable across employers, and none publishes the resolver. This is that,
in about 400 lines of Go and a set of conventions you can carry anywhere.

## Portability

The framework is **cloud-platform agnostic by construction**. `go.mod` carries two dependencies —
a semver parser and a YAML parser. No cloud SDKs, and nothing platform-specific is hardcoded in a
chart: the ingress class, gateway name, proxy and CA bundle are all values a site supplies.

| | |
|---|---|
| **Requires** | Kubernetes 1.28+ · any OCI registry · any GitOps controller (Argo CD or Flux) |
| **Does not require** | any cloud service · a managed control plane · a specific CI system · a service mesh · network egress at install time |
| **Runs on** | EKS · GKE · AKS · OpenShift · RKE2 · k3s at the edge · bare metal · air-gapped enclaves |

Everything that genuinely differs between platforms is a **site value**, not a chart edit — that
is what the [contract plane](docs/02-contract.md) is for. Routing is the sharpest example: set
`global.routing.mode` to `ingress` for a cluster running NGINX or Traefik, or to `gateway` to emit
a Gateway API `HTTPRoute` and keep platform differences in a `Gateway` object the platform team
owns. The service charts are identical either way.

### Tooling

Go 1.24+ for the resolver. `just`, `helm`, `kind`, `kubectl` and a container runtime for the
demo. `zarf` and `cosign` for the air-gap example.

## License

MIT — see [LICENSE](LICENSE).
