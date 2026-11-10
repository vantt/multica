# Patch 0019 — telegram-push (owner decision 2026-11-10: use Multica's own Telegram channel, drop fgOPC's brief bot)

## Requirement
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

## Invariants
- No Telegram binding → nothing sent; the in-app inbox is unchanged.
- Only a bound member can answer, only on an open decision, only once.
- Without `MULTICA_TELEGRAM_SECRET_KEY` nothing of this runs (same gate as upstream).

## Touch points
New: `integrations/telegram/vantt_push.go` (+test), `handler/vantt_decision_notify.go` (+test).
Small edits: `telegram/api.go` (Update.CallbackQuery, allowed_updates), `telegram/telegram_channel.go`
(callback dispatch, ChannelDeps.Callbacks), `cmd/server/router.go` (wiring), `handler/vantt_decision.go`
(one call), web: `InboxItemType` + label + 6 locale strings.

## On conflict
Re-add the `CallbackQuery` field and `"callback_query"` in `GetUpdates`; keep the
dispatch early-return before `inboundFromUpdateWithContext`.

## Tests
See series.toml.
