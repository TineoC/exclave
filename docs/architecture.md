# Architecture

Three figures. Each one carries an argument that prose handles badly.

---

## The boundary sits inside your promotion path

A single dev → staging → prod chain assumes one network. When production lives in an enclave you
cannot reach, it does not — and that changes the shape of the whole system, not just its last
step.

<iframe src="diagrams/boundary.html" title="The boundary between a vendor's build side and a customer-controlled enclave" loading="lazy" style="width:100%;aspect-ratio:5/4;border:1px solid rgba(21,24,28,0.12);border-radius:6px;background:#f2f4f6"></iframe>

<p><a href="diagrams/boundary.html">Open full size →</a></p>

What the figure fixes in place:

- **The signed bundle is the only thing that crosses.** Not a connection, not a credential, not a
  deploy job. One artifact, verified on arrival rather than trusted because of where it came from.
- **The blocked path is drawn, not implied.** A direct apply from the build system into the
  customer's cluster stops dead at the boundary. Leaving it off the diagram is how it ends up in
  someone's pipeline.
- **The reconciler lives on the far side.** It pulls. Nothing on your side ever reaches in, which
  is the only arrangement a customer's security team will accept and the only one that survives an
  air gap.

See [Distribution](03-distribution.md) and [Reconciliation](04-reconciliation.md).

---

## Four planes, and what each refuses to compromise

<iframe src="diagrams/planes.html" title="The four planes of disconnected software delivery" loading="lazy" style="width:100%;aspect-ratio:5/4;border:1px solid rgba(21,24,28,0.12);border-radius:6px;background:#f2f4f6"></iframe>

<p><a href="diagrams/planes.html">Open full size →</a></p>

The stack reads top to bottom in delivery order. **Contract is marked focal deliberately** — it is
the highest-leverage of the four. A values file outlives the architecture, the CI system, and
usually the employer; get it wrong and every environment becomes a snowflake with its own
support matrix.

Each plane and its failure mode: [Artifact](01-artifact.md) · [Contract](02-contract.md) ·
[Distribution](03-distribution.md) · [Reconciliation](04-reconciliation.md).

---

## One release, one environment, five checks

<iframe src="diagrams/resolver.html" title="How the resolver decides whether a release may be installed in an environment" loading="lazy" style="width:100%;aspect-ratio:5/4;border:1px solid rgba(21,24,28,0.12);border-radius:6px;background:#f2f4f6"></iframe>

<p><a href="diagrams/resolver.html">Open full size →</a></p>

Five checks in order. Pass all five and the release is a candidate; fail one and it is reported
blocked **with the failing check named**.

The asymmetry on the left is the point. A failed channel check does not produce a blocker — it
produces silence. An environment tracking `stable` was never going to take an `edge` release, and
saying so on every row buries the one message it can act on:

```
customer-b-prod  3.0.1  3.1.4  upgrade (3.2.0 blocked: requires platform >=3.1, installed 3.0.1)
```

That parenthetical is the feature. Details in [the constraint resolver](constraints.md).

---

The figures live in [`docs/diagrams/`](https://github.com/TineoC/exclave/tree/main/docs/diagrams)
as self-contained HTML — no build step, no image pipeline. The palette and editing notes are in
their [README](https://github.com/TineoC/exclave/blob/main/docs/diagrams/README.md).
