# Patch 0020 — inbox-split (design D11, owner approved 2026-11-11)

## Requirement
The active inbox shows two tabs above the list:
- **Needs you**: rows (an issue's notifications, grouped) where any unarchived
  notification has `severity` `action_required` (assigned to you, which is how a
  person-handled workflow step reaches you; a decision to answer, patch 0019; a run
  that failed) or `attention` (autopilot problems), and the issue is not in a done or
  closed status category. The row shows only its newest notification (often a comment),
  so the split reads all unarchived notifications. Found live on vantt.17: 13 "assigned"
  items hidden under newer comments.
- **For your info**: every other row.

It opens on Needs you while that tab holds anything; its count is red. Each Needs you
row starts with a chip naming its kind ("Decision needed", "Assigned to you", "Run
failed"…). Filters apply inside the tab; the archived view has no tabs (archive = Done).
Web only: every inbox item already carries `severity`. Telegram (0019) is not filtered
by tab: the owner chose to push everything and mute in notification settings.
Reference: OpenMausBot Inbox (Needs you / For your info / Done), Linear Inbox.

## Touch points
New `views/inbox/components/inbox-tabs.tsx` (+test); in `inbox-page.tsx` the tab state,
`visibleItems` from the tab, the tab row under the header; one chip in
`inbox-list-item.tsx`; `inbox.tabs.*` strings in every locale.

## Tests
See series.toml.
