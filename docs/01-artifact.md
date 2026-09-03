# Plane 1 — Artifact

**Invariant:** what you ship is immutable, signed, and self-describing. It carries everything
needed to install it, including its own rules about where it may run.

A release is not a tag on a branch and not a row in a deployment tool's database. It is a file
that travels with the software and answers, without consulting anything else, the question
*"may this go here?"*

```yaml
product: acme-platform
version: 3.2.0
channel: stable

components:
  - name: auth-svc
    version: 1.4.3
    image: acme/auth-svc@sha256:18ac3e73…      # digest, never a tag

requires:
  kubernetes: ">=1.28 <1.32"
  schema: 7
  platform: ">=3.1"                            # the upgrade path

allowedClassifications: [il4, il5, il6]

provenance:
  sbom: sbom/spdx.json
  attestation: attestations/provenance.intoto.jsonl
```

Working example: [`examples/quickstart/catalog/`](https://github.com/TineoC/exclave/tree/main/examples/quickstart/catalog)

## Why constraints live in the artifact

Because the environment is where the decision has to be made, and the environment may be
offline for months. A rule that lives in your deployment tool cannot travel across an air gap;
a rule inside the release manifest arrives with the software that it governs.

This is also what makes the resolver auditable. Every decision it makes is traceable to a line
in a signed file, not to the state of a service you operate.

## Digests, not tags

Tags are mutable. An assessor will ask, and "we always push the same content to that tag" is
not an answer. The demo captures digests **by reading them back from the registry after the
push** — a release manifest claiming a digest nobody verified is worse than no digest at all.

```bash
d=$(docker inspect --format='{{index .RepoDigests 0}}' "$img" | cut -d@ -f2)
```

## Evidence travels too

SBOMs, Cosign signatures, in-toto attestations, scan results. The customer's assessor needs
these and will otherwise generate them badly. Ship them as release artifacts, not as a PDF
attached to an email six weeks later.

## Implementation

OCI artifacts as the universal container — Helm charts, SBOMs, attestations and arbitrary
blobs all push to any registry via [ORAS](https://oras.land/). Sign with
[Cosign](https://docs.sigstore.dev/), attest with [in-toto](https://in-toto.io/), and grade
yourself against [SLSA](https://slsa.dev/).

## Skip this plane and

You cannot answer "what is actually running in enclave 7?", and you cannot prove what you
shipped.
