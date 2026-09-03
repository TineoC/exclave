# Operating this against a real contract

The examples in this repository are fiction, and safe. A real fleet is not, and the difference is
worth being precise about before anyone commits a descriptor with a customer's name in it.

## What is actually sensitive

Take the shipped IL6 example and imagine it real:

```yaml
name: usaf-def-il6                       # customer identity
classification: il6                      # that this site is classified at all
current: "4.2.1"                         # its patch level
pinned: "4.2.1"                          # that it is frozen, and therefore behind
maintenanceWindow: "quarterly, coordinated"   # when it is in flux
registry: harbor.usaf-def.smil/baseline  # a .smil name indicates a classified network
gcpFolder: folders/558231094002          # infrastructure identifier
billingAccount: 04DEFG-HI5678-90JKLM     # financial
```

Reasonable people will argue about the individual fields. **Nobody should argue about the
aggregate.** A directory of these is a map of which classified sites run known-vulnerable versions
and when each one is unattended. Every field could be individually defensible and the directory
would still be a targeting document.

That aggregation risk is why this page exists, and it is the part most teams miss because each field
looks harmless on its own.

## The rule

> **The catalog is shareable. The fleet is not.**

| | Describes | Contains customer data | Where it lives |
|---|---|---|---|
| **Catalog** | your product | no | corp low side, signed, replicated everywhere |
| **Fleet** | customer environments | yes | inside each boundary, never aggregated above it |

The resolver runs **inside** each boundary, against a locally-held fleet and a replicated catalog.
Corp never holds the aggregate.

```
CORP LOW SIDE (IL2)                    EACH ENCLAVE (IL4 / IL5 / IL6)
  catalog/        all releases   ──►     catalog/    replicated, verified
  fleet/          IL2 sites only         fleet/      this contract only
  manifest + signature                   exclave plan runs HERE
                                              │
        redacted roll-up  ◄───────────────────┘
        (human release decision)
```

## The ceiling is enforced, not remembered

Corp CI runs with a ceiling and physically cannot ingest a descriptor above it:

```console
$ exclave validate -fleet fleet/ -max-classification il2
error: fleet/army-abc-il5.yaml: environment "army-abc-il5" is classified il5, above the il2
ceiling this fleet allows — descriptors above the ceiling belong inside their own boundary, not here
```

An over-classified descriptor is an **error, never a silent skip**. Skipping would be worse than
failing: the operator would believe they had a complete picture. An unrecognised classification is
refused too, rather than assumed low — guessing in that direction is how spillage happens.

Put `-max-classification il2` in the corp pipeline on day one, before there is anything to leak.

## The roll-up is a human decision

Leadership still wants a portfolio view. It travels as a redacted artifact:

```console
$ EXCLAVE_REDACTION_SALT=$(cat /secure/salt) exclave redact --format json
SITE               CURRENT  TARGET      STATUS
site-e08f4c01af35  4.0.3    4.2.1       upgrade (4.3.0 blocked: requires platform >=4.2)
site-3a441c1f6f5b  3.9.0    —           no eligible release (kubernetes 1.26 not in >=1.28 <1.32)
```

Names, hostnames, folders, billing references, maintenance windows and classification are gone.
Drift survives, because drift is the entire point of the view.

Three rules, none of which should be softened:

1. **There is no automated high-to-low path.** `exclave redact` produces the artifact; it does not
   release it. Moving it across a boundary is a human review through whatever process your program
   already uses. Any design that wires this into a pipeline is wrong, and will be treated as a
   finding.
2. **The salt lives outside the repository**, as does the site-ID-to-customer mapping. Committing
   the mapping next to the output makes the redaction decorative. The tool refuses to run without a
   salt rather than defaulting to an empty one, because an unsalted digest over guessable names like
   `navy-xyz-il4` is a dictionary attack away from being no redaction at all — while still looking
   redacted to whoever reviews it.
3. **Rotate the salt when the mapping is re-issued.** Site IDs change, which is the point; keep the
   old mapping if you need historical continuity.

One thing redaction deliberately keeps: the `note`. It can mention an environment's Kubernetes
version, which is drift detail rather than identity. If your program treats patch posture as
sensitive even without attribution, strip notes before release — the JSON output makes that a
one-line filter.

## Trusting the catalog

The compliance gate is only as good as the file it reads. Nothing stops someone editing
`criticalCves: 4` down to `0`, or lowering a `requires.platform`, so the catalog gets a manifest and
the manifest gets signed:

```bash
# corp low side, at release time
exclave manifest -catalog catalog > catalog.manifest
cosign sign-blob --yes catalog.manifest --output-signature catalog.manifest.sig

# inside each enclave, before the catalog is trusted
cosign verify-blob catalog.manifest --signature catalog.manifest.sig \
  --certificate-identity-regexp '<your signer>' --certificate-oidc-issuer-regexp '<your issuer>'
exclave verify -catalog catalog -manifest catalog.manifest
```

`verify` names the files that drifted rather than only reporting that something did:

```console
catalog does not match catalog.manifest:
  modified                 baseline/4.3.0/release.yaml
error: 1 file(s) drifted from the signed manifest
```

The manifest is deterministic — sorted relative paths, forward slashes, one digest over the lot — so
the same catalog yields identical bytes on any machine. A manifest whose bytes varied could not be
signed at all.

**This closes tampering, not dishonesty.** A signed catalog proves nobody edited the file after the
pipeline wrote it. It does not prove the pipeline wrote the truth: `criticalCves: 0` is still a
number a build job asserted rather than one tied to a scanner's own signed output. See
[Known gaps](gaps.md).

## Harbor replication

The catalog moves along the same path as the images. Harbor replication rules push from the corp
registry into each enclave's Harbor, pull-through where connectivity allows and via a Zarf bundle
where it does not. **Nothing replicates in the other direction.** If you find yourself configuring
a pull rule from an enclave back to corp, stop — that is the aggregation risk arriving through the
back door.

## Onboarding and offboarding

Adding a contract is [its own walkthrough](onboarding.md).

Offboarding is worth writing down before you need it: revoke the enclave's Harbor robot accounts,
remove the descriptor from the enclave repo, retire the site ID from the salt mapping, and keep the
promotion history — it is the audit trail for everything that ran there, and deleting it to tidy up
destroys evidence you may be asked for years later.
