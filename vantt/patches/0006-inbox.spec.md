# Patch 0006 — inbox

feat(inbox): Telegram push with decision buttons; Needs you / For your info tabs

One patch per feature since 2026-11-11 (owner decision). Made from the old patches 0019 telegram-push, 0020 inbox-split; each part below is that patch's spec, unchanged. Code comments still name the old numbers (`vantt 0019`, `patch 0020`): the table in `series.toml` maps them.

## Tests (whole patch)
`cd server && go run ./cmd/migrate up >/dev/null && go test ./internal/handler/ -run 'Decision' -count=1 && go test ./internal/integrations/telegram/ -count=1 && cd .. && pnpm install --frozen-lockfile --ignore-scripts >/dev/null && cd packages/views && npx vitest run inbox locales`

## Part: old patch 0019 — telegram-push (owner decision 2026-11-10: use Multica's own Telegram channel, drop fgOPC's brief bot)

### Requirement
1. **Inbox → Telegram DM.** On `inbox:new` for a member who bound a Telegram bot
   (`channel_user_binding`), send the item to that bot's private chat with them:
   type label, title, body (a mention's comment text when the body is empty),
   link to the issue. Every item goes, as in the WeCom push: what a member mutes
   in Multica's notification settings never becomes an item (owner decision
   2026-11-10: use Multica's settings, no list of our own).
   Bot choice: the acting agent's bot, else the member's newest binding.
2. **Decision inbox item.** `POST /api/issues/{id}/decisions` (patch 0008) also
   creates `decision_requested` (`action_required`, details `decision_id`) for the
   issue's member subscribers, else workspace owners/admins; never the asker.
3. **Buttons.** A `decision_requested` push carries one inline button per option
   (`vd:<decision>:<index>`, ★ on the recommended one). The polling loop takes
   `callback_query`; a press by a Telegram user bound on that installation to a
   member of the decision's workspace answers it as that member: answer, then the
   comment `**question** → label` (same as the web panel), via the existing
   handlers in-process. Buttons are removed after a press; a closed decision
   answers "already answered".

Reference: Linear's personal Slack notifications; in-tree WeCom `handleInboxNew`.

### Invariants
- No Telegram binding → nothing sent; the in-app inbox is unchanged.
- Only a bound member can answer, only on an open decision, only once.
- Without `MULTICA_TELEGRAM_SECRET_KEY` nothing of this runs (same gate as upstream).

### Touch points
New: `integrations/telegram/vantt_push.go` (+test), `handler/vantt_decision_notify.go` (+test).
Small edits: `telegram/api.go` (Update.CallbackQuery, allowed_updates), `telegram/telegram_channel.go`
(callback dispatch, ChannelDeps.Callbacks), `cmd/server/router.go` (wiring), `handler/vantt_decision.go`
(one call), web: `InboxItemType` + label + 6 locale strings.

### On conflict
Re-add the `CallbackQuery` field and `"callback_query"` in `GetUpdates`; keep the
dispatch early-return before `inboundFromUpdateWithContext`.

### Tests
See series.toml.

## Part: old patch 0020 — inbox-split (design D11, owner approved 2026-11-11)

### Requirement
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

### Touch points
New `views/inbox/components/inbox-tabs.tsx` (+test); in `inbox-page.tsx` the tab state,
`visibleItems` from the tab, the tab row under the header; one chip in
`inbox-list-item.tsx`; `inbox.tabs.*` strings in every locale.

### Tests
See series.toml.
