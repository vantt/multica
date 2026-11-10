package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"testing"
)

// Domain follows Type: a select property described "Derived from <Type>" takes the
// option named like the part of the Type option before " · ".
func TestDerivedPropertyFollowsSource(t *testing.T) {
	typ := createTestProperty(t, map[string]any{"name": "VType", "type": "select", "config": map[string]any{"options": []map[string]any{
		{"name": "Tech · Feature", "color": "#2563eb"}, {"name": "Marketing · Post", "color": "#db2777"}, {"name": "Other", "color": "#71717a"},
	}}})
	dom := createTestProperty(t, map[string]any{"name": "VDomain", "type": "select", "description": "Derived from VType (the part before ·).", "config": map[string]any{"options": []map[string]any{
		{"name": "Tech", "color": "#2563eb"}, {"name": "Marketing", "color": "#db2777"},
	}}})
	opt := func(p PropertyResponse, name string) string {
		for _, o := range p.Config.Options {
			if o.Name == name {
				return o.ID
			}
		}
		t.Fatalf("no option %s", name)
		return ""
	}
	domainOf := func(issueID string) string {
		var raw []byte
		if err := testPool.QueryRow(context.Background(), `SELECT properties FROM issue WHERE id = $1`, issueID).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		bag := map[string]string{}
		_ = json.Unmarshal(raw, &bag)
		return bag[dom.ID]
	}
	var first IssueResponse
	w0 := httptestRecorderCall(t, newRequest("POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{"title": "derive me", "status": "todo"}))
	if w0.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w0.Code, w0.Body.String())
	}
	_ = json.Unmarshal(w0.Body.Bytes(), &first)
	issueID := first.ID
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, issueID) })
	if w := setIssuePropertyRaw(t, issueID, typ.ID, opt(typ, "Tech · Feature")); w.Code != http.StatusOK {
		t.Fatalf("set type: %d %s", w.Code, w.Body.String())
	}
	if got := domainOf(issueID); got != opt(dom, "Tech") {
		t.Fatalf("domain = %q, want Tech", got)
	}
	setIssuePropertyRaw(t, issueID, typ.ID, opt(typ, "Marketing · Post"))
	if got := domainOf(issueID); got != opt(dom, "Marketing") {
		t.Fatalf("domain after change = %q, want Marketing", got)
	}
	// No matching option: the derived value is left alone.
	setIssuePropertyRaw(t, issueID, typ.ID, opt(typ, "Other"))
	if got := domainOf(issueID); got != opt(dom, "Marketing") {
		t.Fatalf("domain changed without a matching option: %q", got)
	}

	// On create, too.
	var created IssueResponse
	w := httptestRecorderCall(t, newRequest("POST", "/api/issues?workspace_id="+testWorkspaceID, map[string]any{
		"title": "derive on create", "properties": map[string]any{typ.ID: opt(typ, "Tech · Feature")},
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, created.ID) })
	if got := domainOf(created.ID); got != opt(dom, "Tech") {
		t.Fatalf("domain on create = %q, want Tech", got)
	}
}

func httptestRecorderCall(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	testHandler.CreateIssue(w, req)
	return w
}
