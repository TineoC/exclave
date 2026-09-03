# Plane 3 — Distribution

**Invariant:** artifacts cross boundaries as verified copies. No runtime egress, ever.
Mirroring is a first-class, documented, repeatable operation — not something an engineer
improvises on site.

## Connected mirror

```bash
skopeo sync --src docker --dest docker acme/ registry.customer.internal/acme/
oras copy registry.acme.io/charts/acme-platform:3.2.0 \
          registry.customer.internal/charts/acme-platform:3.2.0
```

## Air gap

One file crosses, and it is verified on arrival rather than trusted because of where it came
from:

```bash
zarf package create ./product --confirm
cosign verify-blob --signature zarf-package-*.sig zarf-package-*.tar.zst
zarf package deploy zarf-package-acme-platform-amd64-3.2.0.tar.zst --confirm
```

[Zarf](https://zarf.dev/) bundles images, charts, manifests and even a bootstrap registry into
a single transportable archive. It is usable entirely on its own — adopting it does not commit
you to Big Bang or to any particular platform.

Working example:
[`examples/airgap/`](https://github.com/TineoC/exclave/tree/main/examples/airgap)

## The air gap sits inside your promotion path

This is the structural point that reshapes everything, and it is usually discovered too late.

A single dev → staging → prod chain assumes one network. When production is in a classified
enclave, it is not. You get two pipelines and a package handoff:

```
LOW SIDE                          │  CDS  │   HIGH SIDE
                                  │       │
build → scan → sign → SBOM        │       │
  → dev → staging → verify        │       │
  → export bundle ────────────────┼──────►│ import → their change control
                                  │       │
```

Promotion into production stops being "a commit that changes one number" and becomes "an
approved, signed, scanned bundle crosses a boundary." Design for that on day one — retrofitting
an air gap into a connected pipeline is the expensive version.

## Skip this plane and

Installs fail at the worst possible moment because something quietly reached for the internet
in a facility that has none.
