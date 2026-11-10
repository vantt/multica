---
name: multica-catchup
description: Build or refresh the fork release vantt/<tag> of Multica (upstream tag + vantt/patches series), resolve patch conflicts using each patch's spec, and publish images. Use when asked to catch up with a new upstream Multica release, rebuild the fork, or fix a failing [catchup] issue.
---

# Multica catchup

The fork `vantt/multica` keeps upstream code untouched on `main`. Fork changes live only in
`vantt/`, `.claude/skills/multica-*`, `.github/workflows/vantt-*` (CI `vantt-pr-check` enforces it).
Changes to Multica itself are patches in `vantt/patches/`, applied in `series.toml` order.

## Run

```bash
node vantt/scripts/catchup.mjs --dry-run          # validate series, show plan
node vantt/scripts/catchup.mjs                    # tag in vantt/upstream.pin
node vantt/scripts/catchup.mjs --ref v0.7.2       # another upstream tag
```

Result: local branch `vantt/<tag>` and annotated tag `v<tag>-vantt.N` (N goes up when the
series changes on the same upstream tag; unchanged inputs = "already exists"). Nothing is
pushed. In CI, `vantt-catchup.yml` pushes the branch and tag and calls `vantt-images.yml`.

## Conflict (exit 3)

1. Read the patch's spec `vantt/patches/<NNNN>-<id>.spec.md`: requirement, invariants,
   touch points, how to rewrite.
2. Fix the conflicting files so every invariant still holds; never drop a behaviour the
   spec lists. `git add` them.
3. `node vantt/scripts/catchup.mjs --continue` (or `--abort`).
4. Re-export the fixed patch onto `main`: `git format-patch -1 <commit> --stdout >
   vantt/patches/<file>` on a branch from `main`, commit, push.

## Adding a patch

1. Branch from the release branch `vantt/<tag>`, implement, test (`make test`, `pnpm test`).
2. `git format-patch -1 HEAD -o /tmp/p` and copy to `vantt/patches/NNNN-<id>.patch`.
3. Write `NNNN-<id>.spec.md` (requirement, invariants, touch points, rewrite notes).
4. Add a `[[patches]]` block to `series.toml` (see its header). Commit on `main`.

## Moving to a new upstream release

Run the upstream review first (`multica-upstream-review` skill when present), then change
`vantt/upstream.pin`, push to `main`: CI rebuilds. Deploy by setting `MULTICA_IMAGE_TAG`
in fgOPC's stack (`MULTICA_BACKEND_IMAGE=ghcr.io/vantt/multica-backend`,
`MULTICA_WEB_IMAGE=ghcr.io/vantt/multica-web`).
