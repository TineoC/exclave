# Known gaps

A project that states its own gaps is easier to trust than one that implies completeness. This page
is the honest inventory: what exclave does not do, and what it would take.

## The largest one: the gate trusts its input

`exclave verify` proves nobody edited a release file after the pipeline wrote it. It does **not**
prove the pipeline wrote the truth.

```yaml
security:
  criticalCves: 0     # asserted by a build job. Nothing ties this to a scanner.
```

A signed catalog closes tampering. Dishonesty — or an honest bug in a pipeline that reports zero
because the scan step silently failed — passes straight through. Every compliance decision in the
portfolio rests on a number that is currently taken on faith.

**What would close it:** attestation-chain verification. The scanner emits a signed in-toto
attestation; the release references it by digest; `exclave verify` checks the attestation's
signature and that the declared count matches what the attestation actually says. Until then, treat
`criticalCves` as "what the pipeline claimed" and put the assurance in the pipeline's own controls.

The partial mitigation available today: refuse releases with no scan result at all, which the
resolver already does — an unscanned release is rejected rather than assumed clean when an
environment sets `maxCriticalCves`.

## The baseline does not exist

`examples/contracts/catalog/` describes a product with Istio, Keycloak, Harbor, Kyverno and GitLab
runners. That product is fiction — there is no chart in this repository that installs it. The
catalog demonstrates the *shape* of a versioned baseline, not a baseline you can deploy.

Building one is 8–12 weeks and is the largest single piece of work in the surrounding
[portfolio pattern](portfolio.md).

## Landing zone bodies are interface stubs

[`examples/landing-zone/`](../examples/landing-zone/) ships four modules with identical inputs and
outputs and **no resource blocks**. That is deliberate — untested infrastructure code that looks
real is worse than an interface that says what it is — but it means you cannot `terraform apply`
anything here. Fill the bodies from Cloud Foundation Fabric, the AWS Landing Zone Accelerator, or
Azure CAF.

## Golden pipelines do not exist

The portfolio pattern leans on a shared CI template library that every contract includes by
reference. There isn't one in this repository. It is the highest ratio of pain removed to effort in
the whole design and the piece most teams build last.

## No OSCAL generation

[Operating](operating.md) and [the portfolio pattern](portfolio.md) both argue that inherited
accreditation is what changes the economics, and that OSCAL component definitions are how a
contract's SSP becomes a diff rather than a document. Nothing here generates them. The release
schema has a `provenance.oscal` field that is a path nobody writes to.

## No change-board packet

There is no `exclave diff <from> <to>` producing what changed between two baseline versions —
component versions, constraint changes, CVE delta. Change boards ask for exactly that, and today
someone assembles it by hand.

## Redaction keeps the note

`exclave redact` strips names, hostnames, folders, billing references, windows and classification.
It keeps the `note`, which can mention an environment's Kubernetes version. That is drift detail
rather than identity, but it is still detail. If your program treats patch posture as sensitive even
unattributed, filter notes out of the JSON before release — there is no flag for it yet.

## Scale is untested

The resolver is O(environments × releases) with everything in memory. Fine for the dozens; nobody
has run it against thousands. If a fleet gets large enough for that to matter, the fix is
partitioning by boundary — which is what [Operating](operating.md) already requires for entirely
different reasons.

## No server, and that is deliberate

The resolver reads files and prints a table. It has no daemon, no database, no UI, and no
reconciliation loop of its own. Wanting a dashboard is reasonable; `--format json` exists so you can
build one without this project growing a web tier it would then have to secure and accredit.
