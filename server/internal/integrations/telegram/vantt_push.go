package telegram

// vantt patch 0019 (telegram-push): a member's inbox notifications reach their
// Telegram DM through the bot they bound, and a decision arrives with one
// button per option. Pressing one answers the decision as that member.
//
// Reference: Linear's personal Slack notifications; in-tree, the WeCom smart
// bot already pushes inbox:new (wecom/outbound.go handleInboxNew) — this is the
// same route for Telegram, plus inline buttons for decisions (patch 0008).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// CallbackQuery is the Bot API object for a pressed inline button.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

// CallbackHandler receives button presses from one installation's polling loop.
type CallbackHandler interface {
	HandleCallback(ctx context.Context, installationID pgtype.UUID, api *botAPI, cq CallbackQuery)
}

// DecisionAnswerer records option optionIndex of a decision as userID's answer
// and wakes the issue's agent, exactly as the web panel does. It returns the
// option's label; ErrDecisionClosed when the decision is no longer open.
type DecisionAnswerer func(ctx context.Context, workspaceID, userID, decisionID string, optionIndex int) (string, error)

// ErrDecisionClosed reports a press on a decision that was already answered or cancelled.
var ErrDecisionClosed = errors.New("decision is no longer open")

// InboxTypeDecisionRequested is the inbox item type a new decision raises.
const InboxTypeDecisionRequested = "decision_requested"

const (
	decisionCallbackPrefix = "vd:"
	pushBodyMaxRunes       = 3000
)

type Pusher struct {
	q       *db.Queries
	decrypt Decrypter
	appURL  string
	logger  *slog.Logger
	apiBase string
	client  *http.Client
	// Answer answers a decision; nil turns the buttons off.
	Answer DecisionAnswerer
	// spawn runs a delivery; tests run it inline.
	spawn func(func())
}

func NewPusher(q *db.Queries, decrypt Decrypter, appURL string, logger *slog.Logger) *Pusher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pusher{
		q: q, decrypt: decrypt, appURL: strings.TrimRight(appURL, "/"), logger: logger,
		client: &http.Client{Timeout: 15 * time.Second},
		spawn:  func(f func()) { go f() },
	}
}

// Register subscribes to inbox:new. The bus is synchronous, so delivery runs
// off the publisher's goroutine.
func (p *Pusher) Register(bus *events.Bus) {
	bus.Subscribe(protocol.EventInboxNew, p.handleInboxNew)
}

func (p *Pusher) handleInboxNew(e events.Event) {
	payload, ok := e.Payload.(map[string]any)
	if !ok {
		return
	}
	item, ok := payload["item"].(map[string]any)
	if !ok || pushString(item["recipient_type"]) != "member" || !worthPushing(item) {
		return
	}
	p.spawn(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := p.Deliver(ctx, item); err != nil {
			p.logger.WarnContext(ctx, "telegram push: inbox delivery failed", "error", err,
				"recipient_id", pushString(item["recipient_id"]), "type", pushString(item["type"]))
		}
	})
}

// pushTypes are the info-level items that ask something of the person; other
// info items (comments on subscribed issues, status moves) stay in the in-app
// inbox, or a busy agent team would flood the chat. Anything above info goes.
var pushTypes = map[string]bool{
	InboxTypeDecisionRequested: true, "mentioned": true, "issue_assigned": true, "review_requested": true,
}

func worthPushing(item map[string]any) bool {
	return pushTypes[pushString(item["type"])] || (pushString(item["severity"]) != "" && pushString(item["severity"]) != "info")
}

// Deliver sends one inbox item to its recipient's Telegram DM. A recipient
// with no Telegram binding is not an error: the in-app inbox still has it.
func (p *Pusher) Deliver(ctx context.Context, item map[string]any) error {
	workspaceID, err := util.ParseUUID(pushString(item["workspace_id"]))
	if err != nil {
		return nil
	}
	recipientID, err := util.ParseUUID(pushString(item["recipient_id"]))
	if err != nil {
		return nil
	}
	binding, err := p.bindingFor(ctx, item, workspaceID, recipientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lookup binding: %w", err)
	}
	inst, err := p.q.GetChannelInstallation(ctx, db.GetChannelInstallationParams{ID: binding.InstallationID, ChannelType: string(TypeTelegram)})
	if err != nil || inst.Status != "active" {
		return nil
	}
	creds, err := decodeCredentials(inst.Config, p.decrypt)
	if err != nil {
		return fmt.Errorf("decode credentials: %w", err)
	}
	// A private chat's id is the user's id, and a binding exists only after
	// the member wrote to the bot, so the bot may message them.
	chatID, err := strconv.ParseInt(binding.ChannelUserID, 10, 64)
	if err != nil {
		return nil
	}
	text, keyboard := p.render(ctx, item, workspaceID)
	if text == "" {
		return nil
	}
	return sendPush(ctx, newBotAPI(p.apiBase, creds.BotToken, p.client), chatID, text, keyboard)
}

// bindingFor prefers the bot of the agent whose activity raised the item, the
// same rule as FindChannelBindingForMemberOnAgentInstallation's other callers.
func (p *Pusher) bindingFor(ctx context.Context, item map[string]any, workspaceID, recipientID pgtype.UUID) (db.ChannelUserBinding, error) {
	if pushString(item["actor_type"]) == "agent" {
		if agentID, err := util.ParseUUID(pushString(item["actor_id"])); err == nil {
			b, err := p.q.FindChannelBindingForMemberOnAgentInstallation(ctx, db.FindChannelBindingForMemberOnAgentInstallationParams{
				WorkspaceID: workspaceID, MulticaUserID: recipientID, ChannelType: string(TypeTelegram), AgentID: agentID,
			})
			if err == nil {
				return b, nil
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return db.ChannelUserBinding{}, err
			}
		}
	}
	return p.q.FindChannelBindingForMember(ctx, db.FindChannelBindingForMemberParams{
		WorkspaceID: workspaceID, MulticaUserID: recipientID, ChannelType: string(TypeTelegram),
	})
}

type inlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

type inlineKeyboard struct {
	InlineKeyboard [][]inlineButton `json:"inline_keyboard"`
}

var pushTypeLabels = map[string]string{
	InboxTypeDecisionRequested: "Decision needed",
	"mentioned":                "Mentioned you",
	"new_comment":              "New comment",
	"issue_assigned":           "Assigned to you",
	"status_changed":           "Status changed",
	"review_requested":         "Review requested",
	"task_failed":              "Run failed",
	"agent_blocked":            "Agent blocked",
	"agent_completed":          "Agent finished",
	"children_done":            "Sub-issues finished",
}

func pushTypeLabel(t string) string {
	if l, ok := pushTypeLabels[t]; ok {
		return l
	}
	return strings.ReplaceAll(t, "_", " ")
}

// render builds the message (Telegram HTML) and, for a decision, its buttons.
func (p *Pusher) render(ctx context.Context, item map[string]any, workspaceID pgtype.UUID) (string, *inlineKeyboard) {
	typ := pushString(item["type"])
	title := pushString(item["title"])
	if title == "" && typ == "" {
		return "", nil
	}
	details := map[string]any{}
	switch d := item["details"].(type) {
	case json.RawMessage:
		_ = json.Unmarshal(d, &details)
	case []byte:
		_ = json.Unmarshal(d, &details)
	case map[string]any:
		details = d
	}
	body := pushString(item["body"])
	if body == "" {
		// A mention carries the comment id, not its text.
		if cid, err := util.ParseUUID(pushString(details["comment_id"])); err == nil {
			if c, err := p.q.GetComment(ctx, cid); err == nil {
				body = c.Content
			}
		}
	}
	var b strings.Builder
	b.WriteString("<b>" + html.EscapeString(pushTypeLabel(typ)) + "</b> · " + html.EscapeString(title))
	if body != "" {
		if utf8.RuneCountInString(body) > pushBodyMaxRunes {
			body = string([]rune(body)[:pushBodyMaxRunes]) + "…"
		}
		b.WriteString("\n\n" + formatHTML(body))
	}
	if link := p.link(ctx, item, workspaceID); link != "" {
		b.WriteString("\n\n<a href=\"" + html.EscapeString(link) + "\">Open in Multica</a>")
	}
	var keyboard *inlineKeyboard
	if typ == InboxTypeDecisionRequested && p.Answer != nil {
		keyboard = p.decisionButtons(ctx, workspaceID, pushString(details["decision_id"]))
	}
	return b.String(), keyboard
}

func (p *Pusher) link(ctx context.Context, item map[string]any, workspaceID pgtype.UUID) string {
	if p.appURL == "" {
		return ""
	}
	seg := util.UUIDToString(workspaceID)
	if ws, err := p.q.GetWorkspace(ctx, workspaceID); err == nil && ws.Slug != "" {
		seg = ws.Slug
	}
	if issueID := pushString(item["issue_id"]); issueID != "" {
		return p.appURL + "/" + url.PathEscape(seg) + "/issues/" + url.PathEscape(issueID)
	}
	return p.appURL + "/" + url.PathEscape(seg) + "/inbox"
}

type pushOption struct {
	Label       string `json:"label"`
	Recommended bool   `json:"recommended"`
}

func (p *Pusher) decisionButtons(ctx context.Context, workspaceID pgtype.UUID, decisionID string) *inlineKeyboard {
	id, err := util.ParseUUID(decisionID)
	if err != nil {
		return nil
	}
	d, err := p.q.GetIssueDecision(ctx, db.GetIssueDecisionParams{ID: id, WorkspaceID: workspaceID})
	if err != nil || d.Status != "open" {
		return nil
	}
	var options []pushOption
	if json.Unmarshal(d.Options, &options) != nil {
		return nil
	}
	kb := &inlineKeyboard{}
	for i, o := range options {
		label := o.Label
		if o.Recommended {
			label = "★ " + label
		}
		kb.InlineKeyboard = append(kb.InlineKeyboard, []inlineButton{{
			Text: label, CallbackData: decisionCallbackPrefix + util.UUIDToString(id) + ":" + strconv.Itoa(i),
		}})
	}
	return kb
}

// HandleCallback answers a decision from a pressed button. Only a Telegram
// user bound to a member of the decision's workspace can answer.
func (p *Pusher) HandleCallback(ctx context.Context, installationID pgtype.UUID, api *botAPI, cq CallbackQuery) {
	toast := func(text string) {
		_ = api.call(ctx, "answerCallbackQuery", map[string]any{"callback_query_id": cq.ID, "text": text}, nil)
	}
	decisionID, index, ok := parseDecisionCallback(cq.Data)
	if !ok || p.Answer == nil {
		toast("This button is not active.")
		return
	}
	binding, err := p.q.GetChannelUserBindingByUserID(ctx, db.GetChannelUserBindingByUserIDParams{
		InstallationID: installationID, ChannelUserID: strconv.FormatInt(cq.From.ID, 10),
	})
	if err != nil {
		toast("Link your Multica account first: send this bot a message.")
		return
	}
	label, err := p.Answer(ctx, util.UUIDToString(binding.WorkspaceID), util.UUIDToString(binding.MulticaUserID), decisionID, index)
	switch {
	case errors.Is(err, ErrDecisionClosed):
		toast("This decision was already answered.")
	case err != nil:
		p.logger.WarnContext(ctx, "telegram push: answer decision failed", "decision_id", decisionID, "error", err)
		toast("Could not record the answer. Try again in Multica.")
		return
	default:
		toast("✓ " + label)
	}
	if cq.Message != nil {
		// Drop the buttons so the decision cannot be pressed twice.
		_ = api.call(ctx, "editMessageReplyMarkup", map[string]any{
			"chat_id": cq.Message.Chat.ID, "message_id": cq.Message.MessageID, "reply_markup": inlineKeyboard{InlineKeyboard: [][]inlineButton{}},
		}, nil)
		if err == nil {
			_, _ = api.SendMessage(ctx, sendMessageParams{
				ChatID: cq.Message.Chat.ID, Text: "✓ " + label,
				ReplyParameters: &replyParameters{MessageID: cq.Message.MessageID, AllowSendingWithoutReply: true},
			})
		}
	}
}

func parseDecisionCallback(data string) (string, int, bool) {
	rest, ok := strings.CutPrefix(data, decisionCallbackPrefix)
	if !ok {
		return "", 0, false
	}
	id, idx, ok := strings.Cut(rest, ":")
	if !ok {
		return "", 0, false
	}
	n, err := strconv.Atoi(idx)
	if err != nil || n < 0 {
		return "", 0, false
	}
	if _, err := util.ParseUUID(id); err != nil {
		return "", 0, false
	}
	return id, n, true
}

type pushMessageParams struct {
	ChatID                int64           `json:"chat_id"`
	Text                  string          `json:"text"`
	ParseMode             string          `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool            `json:"disable_web_page_preview,omitempty"`
	ReplyMarkup           *inlineKeyboard `json:"reply_markup,omitempty"`
}

// sendPush sends HTML and falls back to plain text when Telegram rejects the markup.
func sendPush(ctx context.Context, api *botAPI, chatID int64, text string, keyboard *inlineKeyboard) error {
	err := api.call(ctx, "sendMessage", pushMessageParams{
		ChatID: chatID, Text: text, ParseMode: "HTML", DisableWebPagePreview: true, ReplyMarkup: keyboard,
	}, nil)
	if err != nil && isHTMLParseError(err) {
		err = api.call(ctx, "sendMessage", pushMessageParams{ChatID: chatID, Text: text, ReplyMarkup: keyboard}, nil)
	}
	return err
}

func pushString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case *string:
		if s != nil {
			return *s
		}
	case pgtype.Text:
		if s.Valid {
			return s.String
		}
	}
	return ""
}
