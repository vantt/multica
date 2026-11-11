# Patch 0008 — email-log-domains

feat(email): log-only recipient domains for internal bot accounts

## Requirement
fgOPC runs bot accounts on a non-routable domain (`agents-<client>@opc.local`,
`ops-bot@opc.local`) and logs them in with the email code read from the backend log
(`stack/login-code.sh`, `opc-priv login-code`). Upstream prints codes to the log only
when no mail backend is set; once Resend or SMTP is on, codes and invitations for these
accounts go to the real backend: login breaks, and the bounces hurt the sending domain.

- `MULTICA_EMAIL_LOG_DOMAINS` (comma list, case-insensitive, `@`/dots trimmed): a
  recipient whose domain is listed, or a subdomain of one, gets the same stdout line as
  DEV mode (`[DEV] Verification code for <to>: <code>`, `[DEV] Invitation email to ...`)
  and nothing is sent. Every other recipient goes through SMTP/Resend as upstream.
- Unset or empty = upstream behaviour. Startup logs `EmailService: log-only domains ...`.

## Invariants
- The `[DEV] Verification code for <to>: ` prefix stays byte-identical (fgOPC greps it).
- Only exact domain or subdomain match: `x@notopc.local` and `x@opc.local.example.com`
  are sent normally.

## Touch points
`server/internal/service/email.go`: one field, its parse in `NewEmailService`, and
`!s.logOnly(to)` / `|| s.logOnly(to)` in `SendVerificationCode` and
`SendInvitationEmail`. New `vantt_email.go` (+test).

On conflict: re-apply the two guards in front of whatever backend dispatch the two send
functions have; the helper file applies as is.

## Tests
`cd server && go test ./internal/service/ -count=1`
