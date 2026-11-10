package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/multica-ai/multica/server/internal/integrations/telegram"
	"github.com/multica-ai/multica/server/internal/testutil"
)

// An agent's decision lands in the inbox of the people who answer it, and a
// chat button answers it as the member: decision answered, answer posted as
// their comment, second press refused.
func TestDecisionNotifyAndAnswerAs(t *testing.T) {
	issueID := dbfx.Issue(t, "decision via chat", testutil.Cols{"status": "todo"})
	seedTestCatalog(t)
	agentID := seededReadyAgentID(t)
	taskID := runningTask(t, agentID, issueID, "todo")
	ask := decisionRequest("POST", "/api/issues/"+issueID+"/decisions",
		map[string]any{"question": "Polar or Dodo?", "options": []map[string]any{{"label": "Polar"}, {"label": "Dodo"}}}, "id", issueID)
	ask.Header.Set("X-Actor-Source", "task_token")
	ask.Header.Set("X-Agent-ID", agentID)
	ask.Header.Set("X-Task-ID", taskID)
	var d DecisionResponse
	testutil.Call(t, testHandler.CreateIssueDecision, ask).Want(http.StatusCreated).JSON(&d)

	var n int
	dbfx.QueryRow(t, `SELECT count(*) FROM inbox_item WHERE recipient_id = $1 AND type = $2 AND severity = 'action_required'
		AND issue_id = $3 AND details->>'decision_id' = $4`, testUserID, telegram.InboxTypeDecisionRequested, issueID, d.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("decision inbox items for the workspace owner = %d", n)
	}

	ctx := context.Background()
	if _, err := testHandler.AnswerDecisionAs(ctx, testWorkspaceID, uuid.NewString(), d.ID, 0); err == nil {
		t.Fatal("a non-member answered")
	}
	label, err := testHandler.AnswerDecisionAs(ctx, testWorkspaceID, testUserID, d.ID, 1)
	if err != nil || label != "Dodo" {
		t.Fatalf("answer = %q, %v", label, err)
	}
	var status, option string
	dbfx.QueryRow(t, `SELECT status, answer_option_id FROM issue_decision WHERE id = $1`, d.ID).Scan(&status, &option)
	if status != "answered" || option != d.Options[1].ID {
		t.Fatalf("decision = %s %s", status, option)
	}
	dbfx.QueryRow(t, `SELECT count(*) FROM comment WHERE issue_id = $1 AND author_type = 'member' AND author_id = $2
		AND content = '**Polar or Dodo?** → Dodo'`, issueID, testUserID).Scan(&n)
	if n != 1 {
		t.Fatalf("answer comments = %d", n)
	}
	if _, err := testHandler.AnswerDecisionAs(ctx, testWorkspaceID, testUserID, d.ID, 0); !errors.Is(err, telegram.ErrDecisionClosed) {
		t.Fatalf("second press = %v", err)
	}
}
