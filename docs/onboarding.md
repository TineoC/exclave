# Adding a contract

A walkthrough for onboarding contract number thirteen. Every command here runs against the shipped
fixtures, so you can follow it end to end before touching anything real.

The test of whether this is working: **onboarding should be data entry, not engineering.** If a new
contract needs code, the [contract plane](02-contract.md) failed.

---

## Before you start

Decide two things, because they are expensive to change later.

**Impact level** — `il2`, `il4`, `il5` or `il6`. IL1 and IL3 were consolidated by the DoD Cloud
Computing SRG and are not levels. This decides where the descriptor is allowed to live.

**Topology** — `contractor-owned` if you bootstrap the cloud hierarchy, `customer-owned` if they
grant you a scope inside theirs. Nothing below the parent scope differs between them, but the
landing zone asserts different permissions.

---

## 1. Provision the landing zone

```hcl
# contracts/navy-xyz/main.tf
module "contract" {
  source = "../../modules/gcp"        # or aws | azure | onprem

  name           = "navy-xyz"
  classification = "il4"
  topology       = "customer-owned"
  parent_scope   = "folders/447120983001"   # the folder they granted you
  billing_ref    = "02BCDE-FG3456-789HIJ"

  region          = "us-central1"
  network_cidr    = "10.42.0.0/16"
  egress          = "restricted"
  cluster_version = "1.29"
  cluster_node_pools = [
    { name = "system", size = "medium", min = 3, max = 6 },
  ]
}
```

The module returns [the ten outputs](../examples/landing-zone/) everything downstream consumes —
cluster endpoint, OIDC issuer, registry URL, audit sink. Swap `source` to `modules/aws` and the
block is otherwise unchanged; that is the whole point of the interface.

## 2. Stand up the enclave

Inside the new cluster: Harbor, the enclave GitLab, and a GitOps controller. This is the baseline
installing itself, so it is one `helm install` of the product chart at whatever version the
portfolio says this classification is entitled to.

Then wire the one-directional replication: **corp Harbor → enclave Harbor, never the reverse.**

## 3. Write the descriptor

In the **enclave's** repository, not corp's:

```yaml
# fleet/contracts/navy-xyz-il4.yaml
name: navy-xyz-il4
tier: production
classification: il4
channel: stable
kubernetes: "1.29"
current: ""                    # nothing installed yet

maxCriticalCves: 0
requiresCapabilities:
  stigProfile: rhel9-v1r3

# Record fields: for the humans, ignored by the resolver.
cloud: gcp
orgTopology: customer-owned
gcpFolder: folders/447120983001
billingAccount: 02BCDE-FG3456-789HIJ
gitlab: enclave
registry: harbor.navy-xyz.mil/baseline
```

Leave `current` empty. An unset current version makes every upgrade-path constraint pass, so a fresh
site can take the newest release it is otherwise eligible for rather than being told it must step
through versions it never had.

Set `maxCriticalCves` and `requiresCapabilities` from the contract's actual obligations. These are
the accreditation commitments, and this file is where they become enforceable rather than aspirational.

> **Do not commit this to the corp repository.** For anything above IL2 it belongs in the enclave.
> See [Operating](operating.md) for why the aggregate is more sensitive than any one descriptor.

## 4. Verify the catalog, then resolve

Inside the enclave, before trusting anything:

```bash
cosign verify-blob catalog.manifest --signature catalog.manifest.sig \
  --certificate-identity-regexp '<your signer>' --certificate-oidc-issuer-regexp '<your issuer>'
exclave verify -catalog catalog -manifest catalog.manifest
```

Then ask what this site is entitled to:

```console
$ exclave plan -catalog catalog -fleet fleet/contracts
ENVIRONMENT   CURRENT  TARGET  STATUS
navy-xyz-il4  —        4.3.0   upgrade
```

If the answer is `no eligible release`, ask why before changing anything:

```console
$ exclave explain navy-xyz-il4
navy-xyz-il4 — tier production, classification il4, channel stable, kubernetes 1.29, max 0 critical CVEs
requires:  stigProfile

BLOCKED  4.4.0-rc.1  (channel candidate)
    FAIL  channel        channel stable does not accept candidate
ELIGIBLE 4.3.0  (channel stable)
    ok    kubernetes     kubernetes 1.29 satisfies >=1.28 <1.32
    ok    capability     stigProfile=rhel9-v1r3 satisfied
    ok    cve            0 critical CVEs within limit of 0
```

Four failure modes and their fixes:

| The plan says | It means | Fix |
|---|---|---|
| `kubernetes 1.26 not in >=1.28 <1.32` | the cluster is too old | upgrade the cluster, not the descriptor |
| `requires platform >=4.2, installed 4.0.x` | no direct migration path | take the intermediate release first |
| `2 critical CVEs, environment allows 0` | the release is not clean enough | wait for the fixed release; do not raise the ceiling |
| `release carries no scan result` | the release was never scanned | fix the pipeline that built it |

The temptation in rows three and four is to edit the descriptor. That converts a compliance control
into a comment, and the edit is visible in git forever.

## 5. Add the site to the roll-up

Nothing to configure. The next redacted export from this enclave includes the new site under an
opaque ID:

```console
$ EXCLAVE_REDACTION_SALT=$(cat /secure/salt) exclave redact -fleet fleet/contracts
SITE               CURRENT  TARGET  STATUS
site-88f525f0371b  —        4.3.0   upgrade
```

Record the `site-88f5…` → `navy-xyz-il4` mapping wherever you keep it — **not in this repository**.

## 6. Promote

Commit the resolved version into the enclave's GitOps repo and let the controller converge. From
here the contract is on the normal cadence: the catalog replicates, `exclave plan` says what it is
entitled to, and a human approves production.

---

## What you did not do

- Write a pipeline. The [golden templates](portfolio.md) are included by reference.
- Write a chart. The baseline is the same product every other contract runs.
- Write Terraform. You filled in a module's inputs.
- Touch the corp repository at all, beyond the catalog it already publishes.

If any of those turned out to be false, that is the signal worth chasing. The first fork of the
baseline is the end of the portfolio, and it always begins as a small reasonable exception.
