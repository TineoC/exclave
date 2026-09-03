# Landing zone

The least portable layer in any platform, and the one most often claimed to be portable.

Org hierarchies, IAM models and billing structures genuinely differ between providers. GCP has
Organization → Folder → **Project** with billing attached per project. AWS has Organization → OU →
**Account**, where accounts are heavyweight and slow to create. Azure has Tenant → Management Group
→ **Subscription**. On-prem has none of it. A module that pretends these are the same thing
produces a lowest-common-denominator abstraction nobody can use.

So the split is:

- **The interface is portable.** Identical inputs, identical outputs, across all four.
- **The implementation is not, and shouldn't be.** Each provider gets its own body.

That is the same move as `global.routing.mode` in the [quickstart](../quickstart/): one contract,
several implementations, the caller never learns which.

## The outputs are the real contract

This is what makes everything *above* the landing zone cloud-agnostic:

```
isolation_scope_id      project / account / subscription / cluster
cluster_name            cluster_endpoint       cluster_ca_certificate
kubeconfig_command      oidc_issuer_url        workload_identity_ref
registry_url            audit_sink_destination classification
```

The baseline, the GitOps controller and exclave consume **these ten outputs**. None of them ever
learns which cloud produced them. A provider that cannot satisfy an output has not implemented the
interface.

`registry_url` is the one whose shape never varies, because Harbor runs identically everywhere —
which is why it is a baseline component rather than a per-cloud managed registry.

## The two topologies

```hcl
topology     = "customer-owned"        # or "contractor-owned"
parent_scope = "folders/447120983001"  # folder / OU / management group
```

`customer-owned` means they created a scope inside their organisation and granted you roles on it.
`contractor-owned` means you bootstrapped the whole tree. **Nothing below the parent scope differs
between the two** — the variable exists so the module asserts the right permissions, not so it
builds two different things. If you find yourself writing a second copy of anything below that
line, the abstraction has leaked.

## What is here, and what is not

Interfaces and the concept mapping. The `resource` blocks are deliberately absent.

These modules `terraform validate` with no providers and no credentials, which is honest: untested
infrastructure code that *looks* real is worse than an interface that says what it is. Fill the
bodies from the canonical public implementations:

| Module | Fill from |
|---|---|
| `gcp` | [Cloud Foundation Fabric](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric) |
| `aws` | [Landing Zone Accelerator](https://aws.amazon.com/solutions/implementations/landing-zone-accelerator-on-aws/) |
| `azure` | [CAF enterprise-scale](https://github.com/Azure/terraform-azurerm-caf-enterprise-scale) |
| `onprem` | [Cluster API](https://cluster-api.sigs.k8s.io/), RKE2, or existing provisioning |

**`onprem` is the module that proves the interface is real.** It has no organisation, no folder
tree and no billing account. If the interface only worked when a hyperscaler supplied the
hierarchy, it was never portable — it was GCP with the names filed off.

## The claim is tested

```bash
just landing-zone
```

`ci/landing-zone-check.sh` asserts every module declares **identical** variables and outputs, that
each validates, and that no provider name reaches the interface's *shape*. Descriptions are exempt
on purpose: an opaque field like `parent_scope` is only usable if its docs say what it means on each
provider.

It exists to catch drift. The day someone adds a GCP-only variable to the shared interface, the
portability claim dies quietly unless something fails.
