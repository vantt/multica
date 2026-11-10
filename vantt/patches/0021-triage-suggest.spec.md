# Patch 0021 — triage-suggest (owner decision 2026-11-10: Cloudflare Workers AI, chat and form)

## Requirement
Suggest a new issue's **project** and **Type** (the property the workflow rules use)
with a fast decision model, Cloudflare's Clef on Workers AI (Jev's "System One" API
shape). Learned from OpenMausBot's decider and Linear's Triage Intelligence.

- Everything offered is read fresh from the workspace on every call (owner rule: no
  hard-coded options): projects with description and recent issue titles; the Type
  property's options with recent issues per option and the property description as
  guidance (a line "<option>: <meaning>" in it is attached to that option).
- Only answers with top probability >= 0.6 become suggestions (MausBot's bench: 92-99%
  right at that level). Fail open: no key, 1.5 s timeout, HTTP error, or an answer that
  fails validation (choice not offered, probabilities not summing to ~1, choice not the
  most likely) = no suggestion.
- `POST /api/issues/triage-suggest {title, description, project_id, has_type}` returns
  only the fields still missing and sure (the form; its UI is this patch's second part).
- Chat `/issue` (origin `*_chat`) without a project: project and Type are set when sure,
  description gains `_Clef xếp: <project> 87% · <Type> 91%_`; a refused create is retried
  as typed.
- One log line per call: job, model, ok/reason, choice, p, margin, latency, input tokens,
  16-char state hash. Never the issue text, the options or the token.
- Env: `MULTICA_WORKERS_AI_ACCOUNT_ID`, `MULTICA_WORKERS_AI_TOKEN` (empty = off),
  `MULTICA_TRIAGE_FORM=off`, `MULTICA_TRIAGE_CHAT=off`, `MULTICA_TRIAGE_MODEL=clef`.

## Eval (opt-in, real model)
`TRIAGE_LIVE_EVAL=1 TRIAGE_EVAL_DATABASE_URL=… TRIAGE_EVAL_WORKSPACE=… [TRIAGE_EVAL_CASES=internal/triage/testdata/opc-studio.json] go test ./internal/triage/ -run LiveEval -v`.
2026-11-10 on opc-studio, 18 owner-style requests: project 13 right / 0 wrong / 5 silent;
Type 7 / 1 / 10 (few Type examples in the workspace yet). Latency ~200 ms.

## Touch points
New package `server/internal/triage`; `handler/vantt_triage.go` (+test); `cmd/server/router.go`
(route, and wrapping the chat router's issue creator).

## Tests
See series.toml.
