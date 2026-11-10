# Fork `vantt/multica`

Fork of [multica-ai/multica](https://github.com/multica-ai/multica). Upstream code stays
untouched on `main`; fork changes live only in:

- `vantt/`: patches, catchup script, plans, design, reviews.
- `.claude/skills/multica-*`: agent instructions (catchup).
- `.github/workflows/vantt-*`: fork CI, on `main` only (release branches carry no workflows). Upstream workflows are disabled on this fork.

Layout copied from [vantt/herdr](https://github.com/vantt/herdr) and
[vantt/obsidian-hybrid-search](https://github.com/vantt/obsidian-hybrid-search).

## How it works

- `vantt/upstream.pin`: upstream tag the fork builds on.
- `vantt/patches/series.toml`: the patch series, in order; each patch has a spec.
- `node vantt/scripts/catchup.mjs`: branch `vantt/<tag>` = upstream tag + fork files +
  patches; tag `v<tag>-vantt.N`. CI `vantt-catchup` pushes it and `vantt-images` builds
  `ghcr.io/vantt/multica-backend` and `ghcr.io/vantt/multica-web` (linux/amd64).
- Remote `upstream` is fetch-only (`git remote set-url --push upstream DISABLE`).

## Rules

- Commits on `main` never touch upstream files (`vantt-pr-check`). Multica code changes
  go through `vantt/patches/` only.
- No configuration = same behaviour as upstream at the same tag.
- Migrations only add tables or nullable columns, so going back to the upstream image
  keeps data usable.
- The Multica logo, name and copyright notices are never removed or changed
  (Multica License, Part I(b)).
- Each patch aims upstream (`upstream_status` in `series.toml`).

## Plan

[`plans/plan.md`](plans/plan.md): patches F0–F8, order and status. Approved UI design: [`design/`](design/).
