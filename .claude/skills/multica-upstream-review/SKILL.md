---
name: multica-upstream-review
description: Review a new upstream Multica release before the fork moves to it — what changed, what touches the fork's patched files, conflicts found by a trial catchup — and write vantt/reviews/<tag>.md. Use when upstream tags a release or before bumping vantt/upstream.pin.
---

# Multica upstream review

1. `git fetch --tags upstream`; list `git log --format=%s <pin>..<tag>` and the CHANGELOG diff.
2. For every file in the `touches` lists of `vantt/patches/series.toml`, run
   `git diff --stat <pin> <tag> -- <file>`; note the ones upstream changed.
3. List new migrations (`git diff --name-only <pin> <tag> -- server/migrations`); fork
   migrations are `9001_vantt_*` and must keep sorting after upstream's.
4. Trial catchup on a fresh test database:
   `DATABASE_URL=… node vantt/scripts/catchup.mjs --ref <tag>`. On a conflict follow the
   patch's spec (multica-catchup skill), then `--continue`.
5. Run `go test ./internal/... -p 1` on the release branch.
6. Write `vantt/reviews/<tag>.md`: what changed that matters to the fork, conflicts and how
   they were rewritten, follow-ups. Re-export the patches from `vantt/<tag>`
   (`git format-patch <tag>..vantt/<tag>`, skipping the maintenance commit), bump
   `vantt/upstream.pin`, commit on main, push: CI rebuilds the images.
7. Deploy: set `MULTICA_IMAGE_TAG=<tag>-vantt.N` on the owner instance (fgOPC stack),
   back up the database first, smoke-test, keep the previous tag for rollback.
