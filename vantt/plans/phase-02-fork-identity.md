# Phase 02 — F0 fork-identity, owner instance on fork images (K4)

## Requirement
The running build says it is the fork: version `0.7.1-vantt.N` in the backend version
endpoint and the web settings footer, using the build args the images already get
(`VERSION`, `NEXT_PUBLIC_APP_VERSION`). Nothing else changes. Logo, name, copyright untouched.

## Steps
- [x] Upstream reads VERSION via `-X main.version` (backend, shown in Help as "Server version"),
      NEXT_PUBLIC_APP_VERSION in the web build; both display-only (no semver checks). CI passes
      `v<tag>-vantt.N` to both, so **F0 needs no patch**.
- [x] fgOPC `stack/env.example`: `MULTICA_BACKEND_IMAGE`, `MULTICA_WEB_IMAGE`, `MULTICA_IMAGE_TAG`.
- [x] Switch the owner instance (backup taken: ~/multica-spike/backups/pre-fork-20261110-151334.sql.gz) (fg-server, workspace opc-studio and others) to the fork
      images; back up the database first (`pg_dump`), note the old tag for rollback.
- [x] Smoke (2026-11-10, v0.7.1-vantt.2): web 200, health ok, server_version v0.7.1-vantt.2,
      CLI ok, `opc.mjs client plan` 0 changes (opc-studio, zalopay-smex), no backend errors.
      Rollback: `~/multica-spike/.env.bak-pre-fork` + `compose.sh up -d backend frontend`.

## Done when
Owner instance runs `v0.7.1-vantt.N`, everything behaves as before, version shows the fork.

## Rollback
Remove the patch from `series.toml` (or set the owner instance back to the previous
`MULTICA_IMAGE_TAG`). Migrations only add, so the previous image keeps working on the data.

## Current (2026-11-10)
Owner instance runs `v0.7.2-vantt.3` (upstream v0.7.2 + 11 patches). Backups:
`~/multica-spike/backups/pre-fork-*.sql.gz`, `pre-v072-*.sql.gz`; previous env `.env.bak-vantt2`.
