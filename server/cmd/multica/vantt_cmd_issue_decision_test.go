package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestDecisionBody(t *testing.T) {
	body, err := decisionBody(" Which gateway? ", []string{"Polar|fees 4%", "Dodo"}, "polar", "2026-11-15")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(body)
	want := `{"due_date":"2026-11-15","options":[{"description":"fees 4%","label":"Polar","recommended":true},{"label":"Dodo"}],"question":"Which gateway?"}`
	if string(raw) != want {
		t.Fatalf("body\n got %s\nwant %s", raw, want)
	}

	for name, c := range map[string]struct {
		q, rec, due string
		opts        []string
		msg         string
	}{
		"no question":  {"", "", "", []string{"A", "B"}, "--question"},
		"one option":   {"Q", "", "", []string{"A"}, "2 to 8"},
		"nine options": {"Q", "", "", strings.Split("a,b,c,d,e,f,g,h,i", ","), "2 to 8"},
		"empty label":  {"Q", "", "", []string{"A", "|why"}, "no label"},
		"unknown rec":  {"Q", "C", "", []string{"A", "B"}, "not one of"},
		"bad due":      {"Q", "", "15/11", []string{"A", "B"}, "YYYY-MM-DD"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := decisionBody(c.q, c.opts, c.rec, c.due)
			if err == nil || !strings.Contains(err.Error(), c.msg) {
				t.Fatalf("err = %v, want it to mention %q", err, c.msg)
			}
		})
	}
}

func TestIssueAskPostsDecision(t *testing.T) {
	const issueID = "0b5c1f7e-2f0a-4d7a-9a51-3f5f2b8b7c11"
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"d-1","status":"open"}`))
	}))
	defer srv.Close()

	t.Setenv("MULTICA_SERVER_URL", srv.URL)
	t.Setenv("MULTICA_WORKSPACE_ID", "ws-1")
	t.Setenv("MULTICA_TOKEN", "test-token")
	cmd := issueAskCmd
	for k, v := range map[string]string{"question": "Ship it?", "recommended": "Yes", "output": "json"} {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	_ = cmd.Flags().Set("option", "Yes")
	_ = cmd.Flags().Set("option", "No|wait a week")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := runIssueAsk(cmd, []string{issueID})
	w.Close()
	os.Stdout = old
	var out bytes.Buffer
	_, _ = io.Copy(&out, r)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "POST /api/issues/"+issueID+"/decisions" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotBody["question"] != "Ship it?" || len(gotBody["options"].([]any)) != 2 {
		t.Fatalf("body = %v", gotBody)
	}
	if !strings.Contains(out.String(), `"d-1"`) {
		t.Fatalf("output = %q", out.String())
	}
}

func TestDecisionAnswerLabel(t *testing.T) {
	d := map[string]any{
		"answer_option_id": "o2",
		"options":          []any{map[string]any{"id": "o1", "label": "A"}, map[string]any{"id": "o2", "label": "B"}},
	}
	if got := decisionAnswer(d); got != "B" {
		t.Fatalf("answer = %q", got)
	}
	if got := decisionAnswer(map[string]any{}); got != "" {
		t.Fatalf("open answer = %q", got)
	}
}
