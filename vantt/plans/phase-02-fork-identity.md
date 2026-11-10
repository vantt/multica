# Phase 02 — F0 fork-identity, owner instance on fork images (K4)

## Requirement
The running build says it is the fork: version `0.7.1-vantt.N` in the backend version
endpoint and the web settings footer, using the build args the images already get
(`VERSION`, `NEXT_PUBLIC_APP_VERSION`). Nothing else changes. Logo, name, copyright untouched.

## Steps
- [ ] Find where upstream reads VERSION / NEXT_PUBLIC_APP_VERSION; confirm a fork tag value
      already shows correctly. If it does, F0 needs no code: record that in the spec.
- [ ] Else: smallest patch so `-vantt.N` shows; spec; series entry.
- [ ] fgOPC `stack/env.example`: `MULTICA_BACKEND_IMAGE`, `MULTICA_WEB_IMAGE`, `MULTICA_IMAGE_TAG`.
- [ ] Switch the owner instance (fg-server, workspace opc-studio and others) to the fork
      images; back up the database first (`pg_dump`), note the old tag for rollback.
- [ ] Smoke: login, list issues, run one agent task, `opc.mjs client plan` → 0 changes for
      every client.

## Done when
Owner instance runs `v0.7.1-vantt.N`, everything behaves as before, version shows the fork.

## Rollback
Remove the patch from `series.toml` (or set the owner instance back to the previous
`MULTICA_IMAGE_TAG`). Migrations only add, so the previous image keeps working on the data.
