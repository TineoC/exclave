# Diagrams

Three figures, each a self-contained HTML file with inline SVG — no build step, no external
assets beyond the webfont.

| File | Type | What it argues |
|---|---|---|
| [`boundary.html`](boundary.html) | Architecture · secure paved road | A signed bundle is the only thing that crosses; direct apply is blocked at the boundary |
| [`planes.html`](planes.html) | Layer stack | The four planes and the one invariant each refuses to give up |
| [`resolver.html`](resolver.html) | Flowchart | Why a release is blocked, and why a channel mismatch is deliberately not reported as one |

Rendered on the docs site at [Architecture](../architecture.md).

## Palette

These use exclave's own tokens — the same cool ground and single hot signal as the rest of the
project's material. Not a brand skin borrowed from anywhere.

| Role | Value | Use |
|---|---|---|
| `paper` | `#f2f4f6` | Page ground |
| `paper-2` | `#ffffff` | Node fill |
| `ink` | `#15181c` | Primary text and stroke |
| `muted` | `#5a616b` | Secondary text, default arrows |
| `soft` | `#8a929e` | Sublabels, arrow labels |
| `rule` | `rgba(21,24,28,0.12)` | Hairlines |
| `accent` | `#c2410c` | Focal — at most two elements per figure |
| `accent-tint` | `rgba(194,65,12,0.08)` | Fill behind an accent stroke |
| `link` | `#1d4e89` | External / API arrows |

Type is the IBM Plex superfamily in three roles: **Serif** for the title, **Sans** for node names,
**Mono** for anything a machine parses — ports, versions, commands, constraint expressions.

**One accent, two elements, maximum.** Orange marks what crosses the boundary and what the
resolver selects. Spending it on a third thing erases the signal.

## Editing

Open the file and edit the SVG directly. Every coordinate is on a 4px grid, connectors are
rounded right-angle elbows, and each arrow label carries an opaque mask with a 6–10px gap above
its line so the connector stays traceable.
