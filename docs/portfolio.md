# Running a portfolio

A contractor with twenty contracts does not have twenty platforms. It has **one platform product,
installed twenty times** — and almost every symptom of the alternative follows from not saying so
out loud: security arriving at the end, ATO firefighting in month nine, custom pipeline code per
contract, and GitLab, Keycloak and Istio deployed from scratch again.

This page is the portfolio pattern. The [four planes](index.md) describe delivering one product to
one environment; this describes running the fleet.

## The reframe

The reversal in [ADR-0002](adr/0002-umbrella-as-release-unit.md) — internally components version
independently, externally the customer installs one number — applies with the contracts as the
customers.

- **Baseline `4.3.0`** is one number a change board approves. Not "Istio 1.24.3 plus Keycloak 26.1.0
  plus Kyverno 1.13.2".
- **Every contract runs an instance of it**, differing only in site values.
- **A contract that needs a chart edit is a bug in the contract plane.** Absorb the difference as a
  value, or decline it. The first fork is the end of the portfolio.

## The fleet is contracts

```yaml
# fleet/contracts/army-abc-il5.yaml
name: army-abc-il5
tier: production
classification: il5
channel: stable
kubernetes: "1.29"
current: "4.0.3"

maxCriticalCves: 0
requiresCapabilities:
  stigProfile: rhel9-v1r3
  fips: true

# Record fields — documentation for the contract, not read by the resolver.
cloud: gcp
orgTopology: contractor-owned
gcpFolder: folders/100000000003
billingAccount: 03CDEF-GH4567-89IJKL
gitlab: enclave
registry: harbor.army-abc.mil/baseline
```

The resolver reads the subset it gates on and ignores the rest. That is deliberate: the descriptor
is the contract's record, and the fields it carries for humans — which billing account, whose org,
which Harbor — belong next to the ones the machine reads.

## Compliance as constraints

Two mechanisms, so the resolver does not grow a bespoke check for every new obligation.

**Capabilities** are exact matches. A release declares what it delivers; a contract declares what it
demands.

```yaml
# in the release
provides:
  stigProfile: rhel9-v1r3
  fips: true

# in the contract
requiresCapabilities:
  stigProfile: rhel9-v1r3
  fips: true
```

Add `fedrampHigh`, `cmmcLevel`, whatever your obligations are. No code change.

**Critical CVEs** are a threshold, because a count is not a promise:

```yaml
# release — measured by the pipeline that built it
security:
  criticalCves: 0

# contract
maxCriticalCves: 0
```

A release with **no scan result is refused, not assumed clean.** An unscanned artifact and a clean
one are different facts, and a contract gating on the count is entitled to say so.

## What the portfolio plan gives you

```console
$ exclave plan -catalog catalog -fleet fleet/contracts
ENVIRONMENT   CURRENT  TARGET      STATUS
army-abc-il5  4.0.3    4.2.1       upgrade (4.3.0 blocked: requires platform >=4.2, installed 4.0.3)
corp-lowside  4.3.0    4.3.0       current (4.4.0-rc.1 blocked: 2 critical CVEs, environment allows 0)
dla-ghi-il4   3.9.0    —           no eligible release (kubernetes 1.26 not in >=1.28 <1.32)
navy-xyz-il4  4.2.1    4.3.0       upgrade
platform-lab  4.3.0    4.4.0-rc.1  upgrade
usaf-def-il6  4.2.1    4.2.1       pinned (pinned, already installed)
```

Six contracts, six different answers, every one of them explained. This is the status report a
program manager currently assembles by asking around, and the evidence an assessor asks for when
they want to know why a site is behind.

Drill into any row:

```console
$ exclave explain army-abc-il5 4.3.0
army-abc-il5 — tier production, classification il5, channel stable, kubernetes 1.29, max 0 critical CVEs
installed: 4.0.3   window: Sun 03:00-05:00 UTC
requires:  fips, stigProfile

BLOCKED  4.3.0  (channel stable)
    ok    channel        channel stable accepts stable
    ok    kubernetes     kubernetes 1.29 satisfies >=1.28 <1.32
    FAIL  platform       requires platform >=4.2, installed 4.0.3
    ok    classification classification il5 is permitted
    ok    capability     fips=true satisfied
    ok    capability     stigProfile=rhel9-v1r3 satisfied
    ok    cve            0 critical CVEs within limit of 0
```

One failing check, named, with the remedy implied: step through 4.2.1 first.

Runnable: [`examples/contracts/`](https://github.com/TineoC/exclave/tree/main/examples/contracts).

## Impact levels

The DoD Cloud Computing SRG defines **IL2, IL4, IL5 and IL6**. IL1 and IL3 were consolidated —
IL1 into IL2, IL3 into IL4 — so a proposal that scopes "IL3" tells an assessor the SRG was not
read. Model them as classifications:

| Level | Data | What changes in the portfolio |
|---|---|---|
| **IL2** | Public / non-critical unclassified | Corp low side, labs. Connected pipeline. |
| **IL4** | CUI | Connected, but customer-controlled accounts and their GitLab. |
| **IL5** | Higher-sensitivity CUI, unclassified NSS | As IL4 plus stricter separation and personnel controls. |
| **IL6** | Classified up to SECRET | Air-gapped. Delivery becomes a signed bundle crossing a boundary — see [Distribution](03-distribution.md). |

A release declares the levels it is accredited for; contracts above that line cannot take it:

```yaml
allowedClassifications: [il2, il4, il5, il6]
```

Only IL6 changes the delivery *mechanism*. IL2 through IL5 differ in controls and isolation, not in
shape — which is why one baseline can serve all four and only the air gap forks the pipeline.

## Two GitLabs

Corp GitLab on the low side is the source of truth. Each enclave runs its own GitLab as a
**delivery target, never a fork**.

```
CORP GITLAB (low side)                    ENCLAVE GITLAB (per contract)
  baseline/            builds 4.3.0         gitops/       pinned versions
  golden-pipelines/    CI templates         apps/         contract app code
  portfolio/           fleet + catalog      runners        inside the boundary
  landing-zone/        Terraform
        │
        └── Harbor replication ──────────►  Harbor (in-enclave)
```

The rule that keeps this honest: **nothing is authored in an enclave GitLab that does not also exist
on the corp side.** The moment a contract patches the baseline locally, you have a fork you will
discover during an audit.

## Harbor is the seam

One registry that runs identically on GKE, on-prem and air-gapped, and it collapses four
requirements into one component:

| Harbor feature | What it replaces |
|---|---|
| **Replication rules** | Your low-side → high-side mirror, and pull-through caching |
| **Trivy scanning** | The CVE gate — and the source of `security.criticalCves` |
| **Cosign verification policy** | "We sign images" becomes "unsigned images cannot be pulled" |
| **Immutable tag rules** | Immutable artifacts, enforced rather than agreed |
| **Projects, quotas, robot accounts** | Per-contract isolation and scoped CI credentials |

Pair it with **Binary Authorization** on GKE and unsigned images cannot run either. Two enforcement
points, both declarative, both evidence an assessor can inspect.

## What this does not do

Honest scoping. The portfolio control plane is one layer of five:

| Layer | Filled by |
|---|---|
| Landing zone — GCP orgs, folders, projects, billing, IAM | Terraform ([Cloud Foundation Fabric](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric)) |
| Cluster baseline — Istio, Keycloak, Harbor, Kyverno, runners | Your umbrella chart, versioned as one number |
| Guardrails — policy, admission control, signing | Kyverno, Binary Authorization, Cosign |
| Evidence — SBOM, SLSA, control mappings | Syft, Trivy, in-toto, [OSCAL](https://pages.nist.gov/OSCAL/) |
| **Portfolio delivery** | **this** |

The landing zone is Terraform and always will be. Do not try to make a resolver provision a GCP
organisation.

### The landing zone can still be provider-agnostic

Not by sharing an implementation — org hierarchies, IAM models and billing structures genuinely
differ — but by sharing an **interface**. Identical inputs, identical outputs, one body per
provider:

```
isolation_scope_id      project / account / subscription / cluster
cluster_endpoint        oidc_issuer_url        workload_identity_ref
registry_url            audit_sink_destination classification
```

Everything above the landing zone consumes those outputs and never learns which cloud produced
them. That is what makes the baseline portable — not the Terraform, which is not.

`topology` is the only branch that matters: `customer-owned` when they granted you a scope inside
their organisation, `contractor-owned` when you bootstrapped the tree. **Nothing below the parent
scope differs**, and the on-prem module — no org, no folders, no billing — is what proves the
interface is real rather than GCP with the names filed off.

Interfaces, the concept mapping per provider, and a conformance check that fails on drift:
[`examples/landing-zone/`](https://github.com/TineoC/exclave/tree/main/examples/landing-zone).

## Inherited accreditation is the whole game

Everything above is tractable engineering. This is the part that changes the economics.

Today each contract assembles its control set from nothing, which is *why* security lands at the
end — there is nothing to inherit, so it cannot start earlier. Accredit the baseline once and each
contract inherits it:

- **OSCAL component definitions** ship with the baseline. A contract's System Security Plan inherits
  and documents only its deltas, turning a 400-page per-contract document into a diff.
- **Evidence is continuous, not assembled.** The pipeline already emits SBOMs, scan results and
  signatures; attaching them to the release makes them ATO artifacts by default.
- **cATO is the destination**, and the argument for it is a git-native audit trail plus automated
  evidence — which the four planes already produce.

The hardest part is not technical. Every contract team will have a reason theirs must be special,
and most of those reasons will be real but small. The baseline has to absorb the small ones as site
values; the organisation has to be willing to decline the rest. That is a leadership problem wearing
an architecture costume, and no resolver fixes it.
