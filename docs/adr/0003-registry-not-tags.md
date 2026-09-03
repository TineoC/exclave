# ADR-0003 — The registry is the version source of truth

**Status:** accepted, reversing an earlier decision

## Context

The first design computed each service's next version from git tags and pushed a new tag from
CI. Two problems surfaced almost immediately.

**A race.** Two merges seconds apart both read the same latest tag, both compute the same next
version, and the second push either fails or silently duplicates.

**A credential.** Pushing a tag means CI holds a token with write access to the source
repository — the worst credential in the pipeline, and the first thing a supply-chain reviewer
will flag.

## Decision

Derive the next version from the **OCI chart registry**, which is already immutable, already
queryable, and already the thing the reconciliation agent pulls from.

Tags are still created, through the Releases API with a scoped job token, for changelogs and
`git describe`. Nothing reads them, and a failed tag never blocks a deploy.

## Consequences

- Immutable pushes make the race impossible: the loser of a collision retries and takes the
  next patch version.
- CI needs no write access to source. See [ADR-0004](0004-ci-never-writes-source.md).
- The registry is queried during release, which is one more thing that must be up. Acceptable:
  it must be up to publish anyway.

## When this flips

At very large scale — past roughly forty services, or when hand-read changelogs stop being
trustworthy — move to a reviewed release-MR flow (Changesets, release-please) where a merged,
human-approved commit creates the version. The registry remains the source of truth; the bump
just stops being automatic.
