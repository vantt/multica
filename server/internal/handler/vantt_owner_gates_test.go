package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/testutil"
)

// vantt owner-gates: a step whose handler is a person is a gate. An agent may
// not move the issue out of it, by status or by project; a member may.

func ownerGateProject(t *testing.T, agentID string) string {
	t.Helper()
	projectID := deliveryProject(t, "Owner gate project "+workflowTestSuffix(), agentID)
	setColumnHandoff(t, projectID, "in_review", map[string]any{
		"handler": map[string]any{"type": "member", "id": testUserID}, "instructions": "Approve the PR.",
	}).Want(http.StatusOK)
	return projectID
}

func runningTask(t *testing.T, agentID, issueID, step string) string {
	t.Helper()
	var runtimeID string
	dbfx.QueryRow(t, `SELECT runtime_id::text FROM agent WHERE id = $1`, agentID).Scan(&runtimeID)
	return dbfx.Task(t, agentID, testutil.Cols{
		"issue_id": issueID, "runtime_id": runtimeID, "status": "running", "started_at": time.Now(), "workflow_step": step,
	})
}

func TestAgentCannotLeaveOwnerGate(t *testing.T) {
	seedTestCatalog(t)
	agentID := seededReadyAgentID(t)
	projectID := ownerGateProject(t, agentID)
	issueID := dbfx.Issue(t, "gated", testutil.Cols{"project_id": projectID, "status": "in_review"})
	// A run of the agent on this issue, even one started for this very step.
	taskID := runningTask(t, agentID, issueID, "in_review")

	body := agentUpdateIssue(t, issueID, agentID, taskID, map[string]any{"status": "done"}).Want(http.StatusForbidden).Map()
	if body["code"] != "owner_gate" || body["status"] != "in_review" {
		t.Fatalf("refusal = %v", body)
	}
	if status, _, _ := issueStatusAndAssignee(t, issueID); status != "in_review" {
		t.Fatalf("status = %q, the refused change must not apply", status)
	}

	// Moving it to another project is leaving the gate too.
	other := createWorkflowTestProject(t, "Owner gate other "+workflowTestSuffix())
	agentUpdateIssue(t, issueID, agentID, taskID, map[string]any{"project_id": other}).Want(http.StatusForbidden)

	// Other fields still change.
	agentUpdateIssue(t, issueID, agentID, taskID, map[string]any{"title": "gated, noted"}).Want(http.StatusOK)

	// A member moves it on.
	updateIssueForTest(t, issueID, map[string]any{"status": "done"}).Want(http.StatusOK)
	if status, _, _ := issueStatusAndAssignee(t, issueID); status != "done" {
		t.Fatalf("status after the member's move = %q, want done", status)
	}
}

// An agent with no run on the issue (a board sync on another issue, say) is
// refused the same way.
func TestAgentFromAnotherIssueCannotLeaveOwnerGate(t *testing.T) {
	seedTestCatalog(t)
	agentID := seededReadyAgentID(t)
	projectID := ownerGateProject(t, agentID)
	gated := dbfx.Issue(t, "gated elsewhere", testutil.Cols{"project_id": projectID, "status": "in_review"})
	elsewhere := dbfx.Issue(t, "sync run", testutil.Cols{"project_id": projectID, "status": "todo"})
	taskID := runningTask(t, agentID, elsewhere, "todo")

	agentUpdateIssue(t, gated, agentID, taskID, map[string]any{"status": "todo"}).Want(http.StatusForbidden)
}

// Steps an agent handles are not gates: agents keep moving their own work.
func TestAgentStepIsNotAGate(t *testing.T) {
	seedTestCatalog(t)
	agentID := seededReadyAgentID(t)
	projectID := ownerGateProject(t, agentID)
	issueID := dbfx.Issue(t, "agent step", testutil.Cols{"project_id": projectID, "status": "todo"})
	updateIssueForTest(t, issueID, map[string]any{"status": "in_progress"}).Want(http.StatusOK)
	var taskID string
	dbfx.QueryRow(t, `SELECT id::text FROM agent_task_queue WHERE issue_id = $1 AND agent_id = $2`, issueID, agentID).Scan(&taskID)
	dbfx.Exec(t, `UPDATE agent_task_queue SET status = 'running', started_at = now() WHERE id = $1`, taskID)

	// Into the gate is fine: that is how work reaches the owner.
	agentUpdateIssue(t, issueID, agentID, taskID, map[string]any{"status": "in_review"}).Want(http.StatusOK)
}

// The batch endpoint skips a gated issue for an agent and applies the rest.
func TestBatchSkipsOwnerGateForAgent(t *testing.T) {
	seedTestCatalog(t)
	agentID := seededReadyAgentID(t)
	projectID := ownerGateProject(t, agentID)
	gated := dbfx.Issue(t, "batch gated", testutil.Cols{"project_id": projectID, "status": "in_review"})
	free := dbfx.Issue(t, "batch free", testutil.Cols{"project_id": projectID, "status": "backlog"})
	taskID := runningTask(t, agentID, free, "backlog")

	req := newRequest("POST", "/api/issues/batch-update", map[string]any{
		"issue_ids": []string{gated, free}, "updates": map[string]any{"status": "todo"},
	})
	req.Header.Set("X-Actor-Source", "task_token")
	req.Header.Set("X-Agent-ID", agentID)
	req.Header.Set("X-Task-ID", taskID)
	testutil.Call(t, testHandler.BatchUpdateIssues, req).Want(http.StatusOK)

	if status, _, _ := issueStatusAndAssignee(t, gated); status != "in_review" {
		t.Fatalf("gated status = %q, the batch must skip it", status)
	}
	if status, _, _ := issueStatusAndAssignee(t, free); status != "todo" {
		t.Fatalf("free status = %q, want todo", status)
	}
}
