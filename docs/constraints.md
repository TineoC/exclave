# The constraint resolver

Given a fleet of environments and a catalog of releases, decide which release each environment
is eligible for — and explain every exclusion.

No open source project does this. It is the gap [Apollo](https://blog.palantir.com/palantir-apollo-orchestration-constraint-based-continuous-deployment-for-modern-architectures-cdf42da19ba4)
fills commercially, and the reason this repository contains code at all rather than only prose.

```console
$ exclave plan
ENVIRONMENT      CURRENT  TARGET  STATUS
customer-a-prod  3.1.4    3.2.0   upgrade
customer-b-prod  3.0.1    3.1.4   upgrade (3.2.0 blocked: requires platform >=3.1, installed 3.0.1)
customer-c-prod  2.9.0    3.0.0   upgrade (3.2.0 blocked: schema 6 < 7)
customer-d-prod  2.8.0    —       no eligible release (kubernetes 1.26 not in >=1.28 <1.32)
internal-lab     3.3.0    3.3.0   current
```

## Explanation is the feature

A resolver that cannot explain itself is not operable. "No eligible release" with no reason
sends people chasing the wrong thing, and it is the failure mode every system of this kind
eventually hits — so every constraint produces a human-readable string whether it passes or
fails, and **those strings are covered by tests**, not just the selected versions.

```console
$ exclave explain customer-c-prod 3.2.0
customer-c-prod — tier production, classification il5, channel stable, kubernetes 1.29, schema 6
installed: 2.9.0   window: Sun 03:00-05:00 UTC

BLOCKED  3.2.0  (channel stable)
    ok    channel        channel stable accepts stable
    FAIL  schema         schema 6 < 7
    ok    kubernetes     kubernetes 1.29 satisfies >=1.28 <1.32
    FAIL  platform       requires platform >=3.1, installed 2.9.0
    ok    classification classification il5 is permitted
```

Every check is reported, not only the first failure. Knowing that *two* things block an upgrade
changes what you do next.

## The constraints

| Constraint | Source | Semantics |
|---|---|---|
| `requires.kubernetes` | release | semver range against the environment's cluster version |
| `requires.schema` | release | monotonic integer floor; the environment must be at or above it |
| `requires.platform` | release | **upgrade path** — the minimum *currently installed* version you may jump from |
| `forbids.environmentTier` | release | deny list matched against the environment's tier |
| `allowedClassifications` | release | allow list; empty means permissive |
| `channel` | both | ladder: `stable` < `candidate` < `edge`; an environment accepts its own rank and below |
| `pinned` | environment | exact version, overriding channel tracking — but still subject to every other constraint |

That is the complete list, and it should stay short. Resist adding a constraint until something
real demands it.

**Maintenance windows are reported, never scheduled.** This is a resolver, not a cron system.
Feed its output to whatever already owns your scheduling.

## Two design details worth copying

**The upgrade path is a release-side constraint, not a graph you maintain.** `requires.platform:
">=3.1"` means *this release knows how to migrate from 3.1 or later*. An environment on 3.0.1
gets an intermediate hop automatically, and is told why, without anyone drawing an upgrade
matrix.

**Only in-channel blockers are reported.** An environment tracking `stable` is not surprised
that an edge release exists, and saying so on every row buries the one message that matters: a
release it *would* take, held back by something it can act on. This was a bug once — it took
writing realistic fixtures to notice.

## Implementation

About 400 lines of Go: [`internal/resolve`](https://github.com/TineoC/exclave/tree/main/internal/resolve).
One dependency beyond the standard library and a YAML parser
([Masterminds/semver](https://github.com/Masterminds/semver)).

Start there, keep it small, and keep it explaining itself.
