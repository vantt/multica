package handler

import (
	"net/http"
	"testing"

	"github.com/multica-ai/multica/server/internal/testutil"
)

func decisionRequest(method, path string, body any, kv ...string) *http.Request {
	return withURLParams(newRequest(method, path+"?workspace_id="+testWorkspaceID, body), kv...)
}

func TestDecisionAskAnswer(t *testing.T) {
	issueID := dbfx.Issue(t, "needs a decision", testutil.Cols{"status": "todo"})

	var d DecisionResponse
	testutil.Call(t, testHandler.CreateIssueDecision, decisionRequest("POST", "/api/issues/"+issueID+"/decisions", map[string]any{
		"question": "Which payment provider?",
		"options": []map[string]any{
			{"label": "Polar", "description": "VND payout", "recommended": true},
			{"label": "Dodo"},
		},
		"due_date": "2026-11-14",
	}, "id", issueID)).Want(http.StatusCreated).JSON(&d)
	if d.Status != "open" || len(d.Options) != 2 || d.Options[0].ID == "" || !d.Options[0].Recommended {
		t.Fatalf("decision = %+v", d)
	}

	var open struct {
		Decisions []DecisionResponse `json:"decisions"`
	}
	testutil.Call(t, testHandler.ListOpenDecisions, newRequest("GET", "/api/issues/decisions/open?workspace_id="+testWorkspaceID, nil)).
		Want(http.StatusOK).JSON(&open)
	found := false
	for _, o := range open.Decisions {
		found = found || o.ID == d.ID
	}
	if !found {
		t.Fatal("open decision not listed")
	}

	// An unknown option is refused; a known one answers it.
	testutil.Call(t, testHandler.AnswerIssueDecision, decisionRequest("POST", "/api/issues/"+issueID+"/decisions/"+d.ID+"/answer",
		map[string]any{"option_id": "nope"}, "id", issueID, "decisionId", d.ID)).Want(http.StatusBadRequest)
	var answered DecisionResponse
	testutil.Call(t, testHandler.AnswerIssueDecision, decisionRequest("POST", "/api/issues/"+issueID+"/decisions/"+d.ID+"/answer",
		map[string]any{"option_id": d.Options[0].ID, "note": "go"}, "id", issueID, "decisionId", d.ID)).Want(http.StatusOK).JSON(&answered)
	if answered.Status != "answered" || answered.AnswerOptionID == nil || *answered.AnswerOptionID != d.Options[0].ID || answered.AnswerNote != "go" {
		t.Fatalf("answered = %+v", answered)
	}
	// Answering twice is a conflict.
	testutil.Call(t, testHandler.AnswerIssueDecision, decisionRequest("POST", "/api/issues/"+issueID+"/decisions/"+d.ID+"/answer",
		map[string]any{"option_id": d.Options[1].ID}, "id", issueID, "decisionId", d.ID)).Want(http.StatusConflict)
}

func TestDecisionValidationAndAgents(t *testing.T) {
	issueID := dbfx.Issue(t, "bad decision", testutil.Cols{"status": "todo"})
	testutil.Call(t, testHandler.CreateIssueDecision, decisionRequest("POST", "/api/issues/"+issueID+"/decisions",
		map[string]any{"question": "One option?", "options": []map[string]any{{"label": "Only"}}}, "id", issueID)).Want(http.StatusBadRequest)
	testutil.Call(t, testHandler.CreateIssueDecision, decisionRequest("POST", "/api/issues/"+issueID+"/decisions",
		map[string]any{"question": "Two recommended?", "options": []map[string]any{{"label": "A", "recommended": true}, {"label": "B", "recommended": true}}}, "id", issueID)).Want(http.StatusBadRequest)

	// An agent may ask but not answer.
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
	if d.AskedByType != "agent" {
		t.Fatalf("asked_by_type = %s", d.AskedByType)
	}
	answer := decisionRequest("POST", "/api/issues/"+issueID+"/decisions/"+d.ID+"/answer",
		map[string]any{"option_id": d.Options[0].ID}, "id", issueID, "decisionId", d.ID)
	answer.Header.Set("X-Actor-Source", "task_token")
	answer.Header.Set("X-Agent-ID", agentID)
	answer.Header.Set("X-Task-ID", taskID)
	testutil.Call(t, testHandler.AnswerIssueDecision, answer).Want(http.StatusForbidden)
}
