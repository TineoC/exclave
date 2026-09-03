# A contract portfolio

Six contracts across IL2, IL4, IL5 and IL6, and a baseline catalog they draw from. No cluster
needed — this is the control plane, not a deployment.

```bash
just portfolio                            # the whole fleet
just explain-contract army-abc-il5 4.3.0  # one decision, check by check
```

## What each contract teaches

| Contract | IL | Outcome | Why it is here |
|---|---|---|---|
| `platform-lab` | 2 | takes `4.4.0-rc.1` | Tracks `candidate`, so release candidates get exercised somewhere before a contract sees one |
| `corp-lowside` | 2 | stays on `4.3.0` | Also tracks candidate, but gates on zero critical CVEs — the RC has two, so the gate holds |
| `navy-xyz-il4` | 4 | upgrades to `4.3.0` | The uneventful case. Customer-owned GCP folder |
| `army-abc-il5` | 5 | upgrades to `4.2.1` | Two versions behind; `4.3.0` cannot migrate from `4.0.x`, so it is routed through the intermediate and told why |
| `usaf-def-il6` | 6 | pinned at `4.2.1` | Air-gapped, and the ATO covers that version specifically. Here a pin is a compliance decision, not an oversight |
| `dla-ghi-il4` | 4 | nothing eligible | Cluster on Kubernetes 1.26. The cluster upgrade has to happen before any baseline upgrade can, and the plan says so |

## Two kinds of field

The resolver reads what it gates on — classification, channel, Kubernetes version, installed
version, capabilities, CVE ceiling. Everything else in a descriptor is a **record field**:

```yaml
cloud: gcp
orgTopology: customer-owned      # or contractor-owned
gcpFolder: folders/447120983001
billingAccount: 02BCDE-FG3456-789HIJ
gitlab: enclave                  # corp | enclave
registry: harbor.navy-xyz.mil/baseline
```

Ignored by the tool, load-bearing for the humans. Which billing account and whose organisation
belong beside the fields the machine reads, not in a spreadsheet somewhere else.

`orgTopology` is the one that matters most in practice: `customer-owned` means they gave you a
folder inside their organisation, `contractor-owned` means you bootstrapped the whole tree. The
landing-zone Terraform branches on exactly that value, and nothing below the folder differs.

## Compliance constraints

```yaml
# release — measured, not promised
provides:   {stigProfile: rhel9-v1r3, fips: true}
security:   {criticalCves: 0, highCves: 1}

# contract — what it demands
requiresCapabilities: {stigProfile: rhel9-v1r3, fips: true}
maxCriticalCves: 0
```

Capabilities match exactly; CVEs are a ceiling. A release carrying **no scan result is refused**
rather than assumed clean — an unscanned artifact and a clean one are different facts.

Adding an obligation is a YAML key, not a code change. `fedrampHigh`, `cmmcLevel`, `fipsMode`,
whatever your contracts actually require.

Full write-up: [the portfolio pattern](https://tineoc.github.io/exclave/portfolio.html).
