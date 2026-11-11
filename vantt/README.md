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
  One patch per feature (workflow, goals, boards, decisions, artifact-reader, inbox,
  triage-suggest): a change to a feature goes into its patch and its spec gets a part.
- `node vantt/scripts/catchup.mjs`: branch `vantt/<tag>` = upstream tag + fork files +
  patches; tag `v<tag>-vantt.N`.
- `bash vantt/scripts/release.sh`: a release on the host: catchup in a throwaway
  worktree, push branch and tag, build and push `ghcr.io/vantt/multica-backend` and
  `ghcr.io/vantt/multica-web` (linux/amd64). Needs a scratch `DATABASE_URL` for the patch
  tests (or `--skip-tests`) and a GHCR token with `write:packages` (or `--no-push`).
  CI `vantt-catchup` and `vantt-images` remain, by hand only: GitHub disabled this fork's
  Actions on 2026-11-11.
- Remote `upstream` is fetch-only (`git remote set-url --push upstream DISABLE`).

## Rules

- Commits on `main` never touch upstream files (`vantt-pr-check`). Multica code changes
  go through `vantt/patches/` only.
- No configuration = same behaviour as upstream at the same tag.
- Migrations only add tables or nullable columns, so going back to the upstream image
  keeps data usable.
- The Multica logo, name and copyright notices are never removed or changed
  (Multica License, Part I(b)).
- Each patch aims upstream (`upstream_status` in `series.toml`); an upstream proposal is cut
  from the matching part of a feature patch.

## Plan

[`plans/plan.md`](plans/plan.md): patches F0–F8, order and status. Approved UI design: [`design/`](design/).
