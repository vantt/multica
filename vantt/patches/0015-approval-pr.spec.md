# Patch 0015 — approval-pr (F1) · design D5b

Needs 0002 (approval-panel).

## Requirement
In the approval panel, list the issue's pull requests: those linked to the issue
(upstream GitHub integration), then GitHub PR links written in its description and
comments (agents post "Approve PR <link>"), newest mention first, one per PR, up to
three. Each row: repo#number (· title, state when linked) and **Approve on GitHub**,
opening the PR's `/files` review page in a new tab, plus the note "Approve the PR on
GitHub with your account, then Approve here." Approve itself is unchanged (moves to
the next step, e.g. `release`). Reference: Jira's development panel on approvals.

No server change; approving on GitHub stays with the owner's own GitHub account.

## Touch points
`packages/views/workflows/approval-prs.tsx` (+ test), one import and
`<ApprovalPullRequests>` after the gate note in `approval-panel.tsx`, its test mocks the
new module, `workflows.approval.{prs_label,approve_on_github,pr_then_approve}` in
every locale.

## Tests
See series.toml.
