# Phase 01 — Foundation (K1–K3)

## Steps
- [x] Fork `multica-ai/multica` → `vantt/multica` (public); clone with `--filter=blob:none`.
- [x] Remote `upstream`, push disabled.
- [x] `vantt/upstream.pin` = `v0.7.1` (the version the owner instance runs).
- [x] `vantt/scripts/catchup.mjs` + `lib/series.mjs`; empty `series.toml`; local run gives
      `vantt/v0.7.1` and `v0.7.1-vantt.1`, second run reports "already exists".
- [x] Workflows: `vantt-pr-check` (fork files only), `vantt-catchup` (manual or on series/pin
      change; opens a `[catchup]` issue on failure), `vantt-images` (GHCR, linux/amd64).
- [x] Skill `.claude/skills/multica-catchup`.
- [x] Disable upstream workflows on the fork (ci, desktop-smoke, mobile-verify,
      openclaw-config-smoke, release, ui-performance).
- [x] First CI catchup green; images `ghcr.io/vantt/multica-{backend,web}:v0.7.1-vantt.1` exist.
- [x] GHCR packages are public (linked to the public repo); the host pulls without a token.

## Done when
CI built and pushed both images for `v0.7.1-vantt.1` from an empty series.
