package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/multica-ai/multica/server/internal/testutil"
)

// vantt send-back-limit: agents may send an issue back along a back edge at
// most max_send_backs times with no person moving it in between.

func sendBackProject(t *testing.T, agentID string) (projectID, workflowID string) {
	t.Helper()
	projectID = deliveryProject(t, "Send-back project "+workflowTestSuffix(), agentID)
	setColumnHandoff(t, projectID, "in_review", map[string]any{
		"handler": map[string]any{"type": "agent", "id": agentID}, "instructions": "Review the PR.",
		"back_status_key": "in_progress",
	}).Want(http.StatusOK)
	workflowID = projectWorkflowID(t, projectID)
	t.Cleanup(func() {
		testPool.Exec(context.Background(), `DELETE FROM vantt_workflow_limit WHERE workflow_id = $1`, workflowID)
	})
	return projectID, workflowID
}

func setSendBackLimit(t *testing.T, workflowID string, body map[string]any) *testutil.Response {
	t.Helper()
	req := withURLParam(workflowRequest("PUT", "/api/issue-workflows/"+workflowID+"/limits", body), "id", workflowID)
	return testutil.Call(t, testHandler.SetIssueWorkflowLimits, req)
}

// recordStatusMove writes the timeline row the activity listener writes.
func recordStatusMove(t *testing.T, issueID, actorType, actorID, from, to string) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), `
		INSERT INTO activity_log (workspace_id, issue_id, actor_type, actor_id, action, details)
		VALUES ($1, $2, $3, $4, 'status_changed', jsonb_build_object('from', $5::text, 'to', $6::text))`,
		testWorkspaceID, issueID, actorType, actorID, from, to); err != nil {
		t.Fatalf("record status move: %v", err)
	}
}

func TestAgentSendBackLimit(t *testing.T) {
	seedTestCatalog(t)
	agentID := seededReadyAgentID(t)
	projectID, workflowID := sendBackProject(t, agentID)
	setSendBackLimit(t, workflowID, map[string]any{"max_send_backs": 2}).Want(http.StatusOK)

	issueID := dbfx.Issue(t, "disputed", testutil.Cols{"project_id": projectID, "status": "in_review"})
	taskID := runningTask(t, agentID, issueID, "in_review")
	// Two send-backs already, plus forward moves that do not count.
	for i := 0; i < 2; i++ {
		recordStatusMove(t, issueID, "agent", agentID, "in_review", "in_progress")
		recordStatusMove(t, issueID, "agent", agentID, "in_progress", "in_review")
	}

	body := agentUpdateIssue(t, issueID, agentID, taskID, map[string]any{"status": "in_progress"}).Want(http.StatusConflict).Map()
	if body["code"] != "send_back_limit" || body["status"] != "in_review" || body["back"] != "in_progress" {
		t.Fatalf("refusal = %v", body)
	}
	if status, _, _ := issueStatusAndAssignee(t, issueID); status != "in_review" {
		t.Fatalf("status = %q, the refused move must not apply", status)
	}
	// Other moves out of the step are not send-backs.
	agentUpdateIssue(t, issueID, agentID, taskID, map[string]any{"title": "disputed, noted"}).Want(http.StatusOK)

	// A person's move starts a fresh count.
	recordStatusMove(t, issueID, "member", testUserID, "in_progress", "in_review")
	agentUpdateIssue(t, issueID, agentID, taskID, map[string]any{"status": "in_progress"}).Want(http.StatusOK)
}

func TestMemberSendBackHasNoLimit(t *testing.T) {
	seedTestCatalog(t)
	agentID := seededReadyAgentID(t)
	projectID, workflowID := sendBackProject(t, agentID)
	setSendBackLimit(t, workflowID, map[string]any{"max_send_backs": 1}).Want(http.StatusOK)

	issueID := dbfx.Issue(t, "member sends back", testutil.Cols{"project_id": projectID, "status": "in_review"})
	recordStatusMove(t, issueID, "agent", agentID, "in_review", "in_progress")
	recordStatusMove(t, issueID, "agent", agentID, "in_progress", "in_review")
	updateIssueForTest(t, issueID, map[string]any{"status": "in_progress"}).Want(http.StatusOK)
}

func TestSendBackWithoutLimitIsUpstream(t *testing.T) {
	seedTestCatalog(t)
	agentID := seededReadyAgentID(t)
	projectID, workflowID := sendBackProject(t, agentID)

	issueID := dbfx.Issue(t, "no limit", testutil.Cols{"project_id": projectID, "status": "in_review"})
	taskID := runningTask(t, agentID, issueID, "in_review")
	for i := 0; i < 5; i++ {
		recordStatusMove(t, issueID, "agent", agentID, "in_review", "in_progress")
	}
	agentUpdateIssue(t, issueID, agentID, taskID, map[string]any{"status": "in_progress"}).Want(http.StatusOK)

	// Clearing a limit (0) answers null.
	setSendBackLimit(t, workflowID, map[string]any{"max_send_backs": 3}).Want(http.StatusOK)
	got := setSendBackLimit(t, workflowID, map[string]any{"max_send_backs": 0}).Want(http.StatusOK).Map()
	if got["max_send_backs"] != nil {
		t.Fatalf("cleared limit = %v, want null", got["max_send_backs"])
	}
	read := testutil.Call(t, testHandler.GetIssueWorkflowLimits,
		withURLParam(workflowRequest("GET", "/api/issue-workflows/"+workflowID+"/limits", nil), "id", workflowID)).Want(http.StatusOK).Map()
	if read["max_send_backs"] != nil {
		t.Fatalf("read limit = %v, want null", read["max_send_backs"])
	}
}
