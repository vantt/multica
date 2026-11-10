package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/testutil"
)

func TestIssueStatusSince(t *testing.T) {
	moved := dbfx.Issue(t, "moved to review", testutil.Cols{"status": "in_review"})
	fresh := dbfx.Issue(t, "never moved", testutil.Cols{"status": "todo"})
	closed := dbfx.Issue(t, "closed", testutil.Cols{"status": "done"})
	at := time.Now().Add(-72 * time.Hour).UTC().Truncate(time.Second)
	if _, err := testPool.Exec(context.Background(),
		`INSERT INTO activity_log (workspace_id, issue_id, actor_type, action, created_at) VALUES ($1, $2, 'member', 'status_changed', $3)`,
		testWorkspaceID, moved, at); err != nil {
		t.Fatal(err)
	}

	var res struct {
		Issues map[string]StatusSinceEntry `json:"issues"`
	}
	testutil.Call(t, testHandler.ListIssueStatusSince, newRequest("GET", "/api/issues/status-since?workspace_id="+testWorkspaceID, nil)).
		Want(http.StatusOK).JSON(&res)
	if got := res.Issues[moved]; got.Status != "in_review" || got.Since != at.Format(time.RFC3339) {
		t.Fatalf("moved = %+v, want in_review at %s", got, at.Format(time.RFC3339))
	}
	if got, ok := res.Issues[fresh]; !ok || got.Since == "" {
		t.Fatalf("fresh issue falls back to created_at, got %+v", got)
	}
	if _, ok := res.Issues[closed]; ok {
		t.Fatal("done issues are not listed")
	}
	testutil.Call(t, testHandler.ListIssueStatusSince, newRequest("GET", "/api/issues/status-since?workspace_id="+testWorkspaceID+"&project_id=nope", nil)).
		Want(http.StatusBadRequest)
}
