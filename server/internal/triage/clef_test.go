package triage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fakeClef(t *testing.T, answer string, status int, delay time.Duration) (*Client, *[]map[string]any) {
	t.Helper()
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/accounts/0123456789abcdef0123456789abcdef/ai/run/@cf/cloudflare/clef-flash") || r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("request %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		bodies = append(bodies, b)
		time.Sleep(delay)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(srv.Close)
	return New("0123456789ABCDEF0123456789ABCDEF", "tok", srv.URL, nil), &bodies
}

var q2 = map[string]Question{"project": {Instructions: "Which?", Options: map[string]string{"p1": "A", "p2": "B"}}}

func TestChooseValidAnswer(t *testing.T) {
	c, bodies := fakeClef(t, `{"success":true,"result":{"answers":{"project":{"type":"choice","choice":"p2","probabilities":{"p1":0.1,"p2":0.9}}}}}`, 200, 0)
	got, err := c.Choose(context.Background(), "test", "", map[string]any{"issue": "x"}, q2)
	if err != nil || got["project"].Choice != "p2" || got["project"].PTop != 0.9 || got["project"].Margin < 0.79 {
		t.Fatalf("got %+v %v", got, err)
	}
	b := (*bodies)[0]
	qs := b["questions"].(map[string]any)["project"].(map[string]any)
	if b["model"] != "clef-flash" || qs["type"] != "choice" || qs["criteria"].(map[string]any)["p1"] != "A" {
		t.Fatalf("wire body = %+v", b)
	}
}

// Every bad answer is refused (MausBot: a choice not offered, probabilities
// that are not probabilities, a choice that is not the most likely).
func TestChooseRejects(t *testing.T) {
	for name, body := range map[string]string{
		"not offered":   `{"result":{"answers":{"project":{"choice":"p9","probabilities":{"p1":0.1,"p9":0.9}}}}}`,
		"not top":       `{"result":{"answers":{"project":{"choice":"p1","probabilities":{"p1":0.1,"p2":0.9}}}}}`,
		"bad sum":       `{"result":{"answers":{"project":{"choice":"p2","probabilities":{"p1":0.5,"p2":0.9}}}}}`,
		"missing":       `{"result":{"answers":{}}}`,
		"not json":      `nope`,
		"success=false": `{"success":false,"result":{"answers":{"project":{"choice":"p2","probabilities":{"p1":0.1,"p2":0.9}}}}}`,
	} {
		c, _ := fakeClef(t, body, 200, 0)
		if _, err := c.Choose(context.Background(), "test", "", nil, q2); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	c, _ := fakeClef(t, `{}`, 429, 0)
	if _, err := c.Choose(context.Background(), "test", "", nil, q2); err == nil || err.Error() != "http 429" {
		t.Errorf("429: %v", err)
	}
	slow, _ := fakeClef(t, `{}`, 200, 300*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := slow.Choose(ctx, "test", "", nil, q2); err == nil {
		t.Error("timeout accepted")
	}
	var off *Client
	if _, err := off.Choose(context.Background(), "test", "", nil, q2); err != ErrOff {
		t.Errorf("nil client: %v", err)
	}
}

func TestOptionGuidanceAndPick(t *testing.T) {
	g := optionGuidance("Kind of work.\n- Tech · Bug: something broken\nTech · Feature — new things", []string{"Tech · Bug", "Tech · Feature", "Tech · Doc"})
	if g["Tech · Bug"] != "something broken" || g["Tech · Feature"] != "new things" || g["Tech · Doc"] != "" {
		t.Fatalf("guidance = %+v", g)
	}
	opts := []option{{id: "a"}, {id: "b"}}
	if p := pick("t", "t2", opts); p == nil || p.id != "b" {
		t.Fatal("pick t2")
	}
	if pick("t", "t3", opts) != nil || pick("t", "x1", opts) != nil {
		t.Fatal("pick out of range")
	}
}
