package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/service"
	"github.com/multica-ai/multica/server/internal/testutil"
	"github.com/multica-ai/multica/server/internal/triage"
	"github.com/multica-ai/multica/server/internal/util"
)

// fakeClefPicking answers each question with the option whose text starts
// with want[question] at probability p, so the test does not depend on how
// many other projects the shared test workspace holds.
func fakeClefPicking(t *testing.T, want map[string]string, p float64) *triage.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Questions map[string]struct {
				Criteria map[string]string `json:"criteria"`
			} `json:"questions"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		answers := map[string]any{}
		for id, q := range body.Questions {
			choice := ""
			probs := map[string]float64{}
			for k, text := range q.Criteria {
				if strings.HasPrefix(text, want[id]) {
					choice = k
				}
			}
			for k := range q.Criteria {
				if k == choice {
					probs[k] = p
				} else {
					probs[k] = (1 - p) / float64(len(q.Criteria)-1)
				}
			}
			answers[id] = map[string]any{"type": "choice", "choice": choice, "probabilities": probs}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]any{"answers": answers}})
	}))
	t.Cleanup(srv.Close)
	return triage.New("0123456789abcdef0123456789abcdef", "tok", srv.URL, nil)
}

func seedTriageWorkspace(t *testing.T) (projectID, propID, optionID string) {
	t.Helper()
	suffix := uuid.NewString()[:8]
	dbfx.QueryRow(t, `INSERT INTO project (workspace_id, title, description) VALUES ($1, $2, 'Story app for developers') RETURNING id::text`,
		testWorkspaceID, "Momentry "+suffix).Scan(&projectID)
	var other string
	dbfx.QueryRow(t, `INSERT INTO project (workspace_id, title, description) VALUES ($1, $2, 'Company blog') RETURNING id::text`,
		testWorkspaceID, "Blog "+suffix).Scan(&other)
	optionID = uuid.NewString()
	cfg, _ := json.Marshal(map[string]any{"options": []map[string]string{
		{"id": optionID, "name": "Tech · Bug " + suffix, "color": "#ef4444"},
		{"id": uuid.NewString(), "name": "Marketing · Post " + suffix, "color": "#ec4899"},
	}})
	dbfx.QueryRow(t, `INSERT INTO issue_property (workspace_id, name, type, config, description) VALUES ($1, 'Type', 'select', $2, 'Kind of work') RETURNING id::text`,
		testWorkspaceID, cfg).Scan(&propID)
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM issue_property WHERE id = $1`, propID)
		testPool.Exec(context.Background(), `DELETE FROM issue WHERE project_id IN ($1, $2)`, projectID, other)
		testPool.Exec(context.Background(), `DELETE FROM project WHERE id IN ($1, $2)`, projectID, other)
	})
	return projectID, propID, optionID
}

func TestTriageSuggestEndpointAndChat(t *testing.T) {
	projectID, propID, optionID := seedTriageWorkspace(t)
	var projectName string
	dbfx.QueryRow(t, `SELECT title FROM project WHERE id = $1`, projectID).Scan(&projectName)
	var optionName string
	dbfx.QueryRow(t, `SELECT o->>'name' FROM issue_property, jsonb_array_elements(config->'options') o WHERE id = $1 AND o->>'id' = $2`, propID, optionID).Scan(&optionName)

	prev := Triage
	t.Cleanup(func() { Triage = prev })
	Triage = &triage.Suggester{DB: testPool, Client: fakeClefPicking(t, map[string]string{"project": projectName, "type": optionName}, 0.9)}

	call := func(body map[string]any) triage.Suggestion {
		var s triage.Suggestion
		testutil.Call(t, testHandler.TriageSuggest, newRequest("POST", "/api/issues/triage-suggest", body)).Want(http.StatusOK).JSON(&s)
		return s
	}
	s := call(map[string]any{"title": "App crashes on long sessions"})
	if s.Project == nil || s.Project.ID != projectID || s.Type == nil || s.Type.OptionID != optionID || s.Type.PropertyID != propID || s.Provider != "clef" {
		t.Fatalf("suggestion = %+v", s)
	}
	// Fields the person already set are not asked for.
	if s := call(map[string]any{"title": "x", "project_id": projectID, "has_type": true}); s.Project != nil || s.Type != nil {
		t.Fatalf("asked for set fields: %+v", s)
	}
	// Unsure: nothing.
	Triage.Client = fakeClefPicking(t, map[string]string{"project": projectName, "type": optionName}, 0.55)
	if s := call(map[string]any{"title": "App crashes"}); s.Project != nil || s.Type != nil {
		t.Fatalf("unsure suggested: %+v", s)
	}

	// Chat /issue: filled when sure, with the attribution line.
	Triage.Client = fakeClefPicking(t, map[string]string{"project": projectName, "type": optionName}, 0.9)
	chat := &triage.ChatIssues{IssueCreator: testHandler.IssueService, Suggester: Triage}
	res, err := chat.Create(context.Background(), service.IssueCreateParams{
		WorkspaceID: util.MustParseUUID(testWorkspaceID), Title: "App crashes on long sessions", Status: "todo", Priority: "none",
		CreatorType: "member", CreatorID: util.MustParseUUID(testUserID),
		OriginType: pgtype.Text{String: "telegram_chat", Valid: true},
	}, service.IssueCreateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	var gotProject, gotType, desc string
	dbfx.QueryRow(t, `SELECT coalesce(project_id::text, ''), coalesce(properties->>$2, ''), coalesce(description, '') FROM issue WHERE id = $1`,
		util.UUIDToString(res.Issue.ID), propID).Scan(&gotProject, &gotType, &desc)
	if gotProject != projectID || gotType != optionID || !strings.Contains(desc, fmt.Sprintf("Clef xếp: %s 90%%", projectName)) {
		t.Fatalf("chat issue: project=%s type=%s desc=%q", gotProject, gotType, desc)
	}
	// Not from chat: untouched.
	res, err = chat.Create(context.Background(), service.IssueCreateParams{
		WorkspaceID: util.MustParseUUID(testWorkspaceID), Title: "Plain create", Status: "todo", Priority: "none",
		CreatorType: "member", CreatorID: util.MustParseUUID(testUserID),
	}, service.IssueCreateOpts{})
	if err != nil || res.Issue.ProjectID.Valid {
		t.Fatalf("plain create changed: %v %v", err, res.Issue.ProjectID)
	}
	testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, res.Issue.ID)
}
