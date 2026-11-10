package telegram

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type botCall struct {
	Method string
	Body   map[string]any
}

// fakeBot records Bot API calls and answers ok.
func fakeBot(t *testing.T) (*httptest.Server, func() []botCall) {
	t.Helper()
	var mu sync.Mutex
	var calls []botCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		calls = append(calls, botCall{Method: r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:], Body: body})
		mu.Unlock()
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":7,"chat":{"id":1}}}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []botCall { mu.Lock(); defer mu.Unlock(); return append([]botCall(nil), calls...) }
}

func TestPushParseCallback(t *testing.T) {
	if _, _, ok := parseDecisionCallback("vd:not-a-uuid:1"); ok {
		t.Error("bad uuid accepted")
	}
	id := uuid.NewString()
	if got, n, ok := parseDecisionCallback("vd:" + id + ":2"); !ok || got != id || n != 2 {
		t.Errorf("parse = %s %d %v", got, n, ok)
	}
}

// A decision reaches the bound member's DM with one button per option, and a
// press answers it as that member.
func TestPushDecisionButtons(t *testing.T) {
	if testPool == nil {
		t.Skip("no database")
	}
	ctx := context.Background()
	q := db.New(testPool)
	ws, agent, user, inst, issue := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	cfg, _ := json.Marshal(installConfig{AppID: "424242" + ws[:4], BotUsername: "pushbot", BotTokenEncrypted: base64.StdEncoding.EncodeToString([]byte("123:TOKEN"))})
	mustExec := func(sql string, args ...any) {
		t.Helper()
		if _, err := testPool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec(`INSERT INTO channel_installation (id, workspace_id, agent_id, channel_type, config, installer_user_id) VALUES ($1,$2,$3,'telegram',$4,$5)`, inst, ws, agent, cfg, user)
	mustExec(`INSERT INTO channel_user_binding (workspace_id, multica_user_id, installation_id, channel_type, channel_user_id) VALUES ($1,$2,$3,'telegram','555001')`, ws, user, inst)
	var decisionID string
	if err := testPool.QueryRow(ctx, `INSERT INTO issue_decision (issue_id, workspace_id, question, options, asked_by_type)
		VALUES ($1,$2,'Polar or Dodo?','[{"id":"a","label":"Polar","recommended":true},{"id":"b","label":"Dodo"}]','agent') RETURNING id`, issue, ws).Scan(&decisionID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testPool.Exec(ctx, `DELETE FROM channel_user_binding WHERE installation_id = $1`, inst)
		testPool.Exec(ctx, `DELETE FROM channel_installation WHERE id = $1`, inst)
		testPool.Exec(ctx, `DELETE FROM issue_decision WHERE id = $1`, decisionID)
	})

	srv, calls := fakeBot(t)
	type answered struct {
		ws, user, decision string
		index              int
	}
	var got []answered
	p := NewPusher(q, nil, "https://multica.example", nil)
	p.apiBase = srv.URL
	p.Answer = func(_ context.Context, workspaceID, userID, decisionID string, i int) (string, error) {
		got = append(got, answered{workspaceID, userID, decisionID, i})
		return []string{"Polar", "Dodo"}[i], nil
	}

	details, _ := json.Marshal(map[string]string{"decision_id": decisionID})
	item := map[string]any{
		"workspace_id": ws, "recipient_type": "member", "recipient_id": user, "type": InboxTypeDecisionRequested,
		"severity": "action_required", "issue_id": issue, "title": "Polar or Dodo?", "body": "Pick a payment provider",
		"actor_type": "agent", "actor_id": agent, "details": json.RawMessage(details),
	}
	if err := p.Deliver(ctx, item); err != nil {
		t.Fatal(err)
	}
	sent := calls()
	if len(sent) != 1 || sent[0].Method != "sendMessage" || sent[0].Body["chat_id"].(float64) != 555001 {
		t.Fatalf("calls = %+v", sent)
	}
	text, _ := sent[0].Body["text"].(string)
	if !strings.Contains(text, "Decision needed") || !strings.Contains(text, "/issues/"+issue) {
		t.Fatalf("text = %q", text)
	}
	rows := sent[0].Body["reply_markup"].(map[string]any)["inline_keyboard"].([]any)
	first := rows[0].([]any)[0].(map[string]any)
	if len(rows) != 2 || first["text"] != "★ Polar" || first["callback_data"] != "vd:"+decisionID+":0" {
		t.Fatalf("keyboard = %+v", rows)
	}

	// The bound user presses "Dodo".
	api := newBotAPI(srv.URL, "123:TOKEN", nil)
	instID := util.MustParseUUID(inst)
	p.HandleCallback(ctx, instID, api, CallbackQuery{ID: "cq1", From: User{ID: 555001}, Data: "vd:" + decisionID + ":1",
		Message: &Message{MessageID: 7, Chat: Chat{ID: 555001}}})
	if len(got) != 1 || got[0] != (answered{ws, user, decisionID, 1}) {
		t.Fatalf("answered = %+v", got)
	}
	after := calls()[1:]
	if len(after) != 3 || after[0].Method != "answerCallbackQuery" || after[0].Body["text"] != "✓ Dodo" ||
		after[1].Method != "editMessageReplyMarkup" || after[2].Method != "sendMessage" {
		t.Fatalf("after press = %+v", after)
	}

	// A Telegram user nobody bound cannot answer.
	p.HandleCallback(ctx, instID, api, CallbackQuery{ID: "cq2", From: User{ID: 999}, Data: "vd:" + decisionID + ":0"})
	if len(got) != 1 {
		t.Fatal("unbound user answered")
	}

	// No binding for the recipient: nothing is sent.
	other := map[string]any{"workspace_id": ws, "recipient_type": "member", "recipient_id": uuid.NewString(), "type": "mentioned", "title": "x"}
	before := len(calls())
	if err := p.Deliver(ctx, other); err != nil || len(calls()) != before {
		t.Fatalf("unbound delivery: err=%v calls=%d", err, len(calls())-before)
	}
}
