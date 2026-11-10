# Phase 07 — F5 decisions (K9) · design D4

## Reference
Jira approvals; Paperclip `decisions`, `decision_queues`, `issue_thread_interactions`.

## Requirement
An issue can carry a decision: question, options (one recommended), due, decider, what it
blocks. While open, the issue's agent does not run (a gate). Answer by UI or
`POST /api/issues/{id}/decision/answer` (option id, optional note) → comment + wake the
agent. Listed in Waiting on me as kind "Decision". The morning brief (fgOPC) calls the API.

## Steps
- [ ] Read Paperclip's implementation; spec `0005-decisions.spec.md`; table `issue_decision`.
- [ ] Server + core + views (D4 panel); agent skill note in fgOPC (how to open a decision).

## Done when
Phương opens a decision; owner answers from Telegram/page; the agent resumes.

## Rollback
Remove the patch from `series.toml` (or set the owner instance back to the previous
`MULTICA_IMAGE_TAG`). Migrations only add, so the previous image keeps working on the data.
