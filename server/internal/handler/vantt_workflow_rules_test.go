package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/multica-ai/multica/server/internal/testutil"
)

// vantt workflow-by-type: a project's rule sends issues with a given select
// option to another workflow.

type ruleFixture struct {
	projectID, propertyID, copyOption, codeOption, contentWF string
}

func setupWorkflowRule(t *testing.T) ruleFixture {
	t.Helper()
	sfx := workflowTestSuffix()
	projectID := createWorkflowTestProject(t, "Rule project "+sfx)

	var prop PropertyResponse
	testutil.Call(t, testHandler.CreateProperty, goalRequest("POST", "/api/properties", map[string]any{
		"name": "Type " + sfx[len(sfx)-6:], "type": "select",
		"config": map[string]any{"options": []map[string]any{{"name": "Copy", "color": "#ec4899"}, {"name": "Code", "color": "#3b82f6"}}},
	})).Want(http.StatusCreated).JSON(&prop)
	t.Cleanup(func() { dbfx.Exec(t, `DELETE FROM issue_property WHERE id = $1`, prop.ID) })

	content := createTestWorkflow(t, "content "+sfx, "in_review")
	f := ruleFixture{projectID: projectID, propertyID: prop.ID, contentWF: content.ID}
	for _, o := range prop.Config.Options {
		if o.Name == "Copy" {
			f.copyOption = o.ID
		} else {
			f.codeOption = o.ID
		}
	}
	req := withURLParam(goalRequest("PUT", "/api/projects/"+projectID+"/workflow-rules", map[string]any{
		"rules": []map[string]any{{"property_id": prop.ID, "option_id": f.copyOption, "workflow_id": content.ID}},
	}), "id", projectID)
	testutil.Call(t, testHandler.SetProjectWorkflowRules, req).Want(http.StatusOK)
	t.Cleanup(func() { dbfx.Exec(t, `DELETE FROM project_workflow_rule WHERE project_id = $1`, projectID) })
	return f
}

func TestWorkflowRuleListAndValidation(t *testing.T) {
	seedTestCatalog(t)
	f := setupWorkflowRule(t)
	var list struct {
		Rules []WorkflowRuleJSON `json:"rules"`
	}
	testutil.Call(t, testHandler.ListProjectWorkflowRules, withURLParam(goalRequest("GET", "/api/projects/"+f.projectID+"/workflow-rules", nil), "id", f.projectID)).
		Want(http.StatusOK).JSON(&list)
	if len(list.Rules) != 1 || list.Rules[0].WorkflowID != f.contentWF {
		t.Fatalf("rules = %+v", list.Rules)
	}
	bad := withURLParam(goalRequest("PUT", "/api/projects/"+f.projectID+"/workflow-rules", map[string]any{
		"rules": []map[string]any{{"property_id": f.propertyID, "option_id": "not-an-option", "workflow_id": f.contentWF}},
	}), "id", f.projectID)
	testutil.Call(t, testHandler.SetProjectWorkflowRules, bad).Want(http.StatusBadRequest)
}

// An issue whose Type is Copy follows the content workflow: a status only that
// workflow lacks is refused, and one it lists is accepted.
func TestIssueFollowsRuleWorkflow(t *testing.T) {
	seedTestCatalog(t)
	f := setupWorkflowRule(t)
	props, _ := json.Marshal(map[string]string{f.propertyID: f.copyOption})
	issueID := dbfx.Issue(t, "copy issue", testutil.Cols{"project_id": f.projectID, "status": "todo", "properties": string(props)})

	// content lists backlog, in_review, blocked (+ todo, done, cancelled), not in_progress.
	updateIssueForTest(t, issueID, map[string]any{"status": "in_progress"}).Want(http.StatusBadRequest)
	updateIssueForTest(t, issueID, map[string]any{"status": "in_review"}).Want(http.StatusOK)

	// A Code issue in the same project stays on the project's workflow (Default).
	codeProps, _ := json.Marshal(map[string]string{f.propertyID: f.codeOption})
	code := dbfx.Issue(t, "code issue", testutil.Cols{"project_id": f.projectID, "status": "todo", "properties": string(codeProps)})
	updateIssueForTest(t, code, map[string]any{"status": "in_progress"}).Want(http.StatusOK)
}

// Switching Type moves a status the new workflow lacks to its equivalent.
func TestChangingTypeFollowsNewWorkflow(t *testing.T) {
	seedTestCatalog(t)
	f := setupWorkflowRule(t)
	issueID := dbfx.Issue(t, "becomes copy", testutil.Cols{"project_id": f.projectID, "status": "in_progress"})

	req := withURLParams(goalRequest("PUT", "/api/issues/"+issueID+"/properties/"+f.propertyID, map[string]any{"value": f.copyOption}),
		"id", issueID, "propertyId", f.propertyID)
	testutil.Call(t, testHandler.SetIssueProperty, req).Want(http.StatusOK)
	if status, _, _ := issueStatusAndAssignee(t, issueID); status == "in_progress" {
		t.Fatalf("status stayed in_progress, which the content workflow does not list")
	}
}
