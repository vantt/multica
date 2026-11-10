package handler

import (
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/testutil"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

func goalRequest(method, path string, body any) *http.Request {
	return newRequest(method, path+"?workspace_id="+testWorkspaceID, body)
}

func createGoalForTest(t *testing.T, body map[string]any) GoalResponse {
	t.Helper()
	var g GoalResponse
	testutil.Call(t, testHandler.CreateGoal, goalRequest("POST", "/api/goals", body)).Want(http.StatusCreated).JSON(&g)
	t.Cleanup(func() {
		dbfx.Exec(t, `DELETE FROM goal_link WHERE goal_id = $1`, g.ID)
		dbfx.Exec(t, `DELETE FROM goal_value WHERE goal_id = $1`, g.ID)
		dbfx.Exec(t, `DELETE FROM goal WHERE id = $1`, g.ID)
	})
	return g
}

func TestGoalHealth(t *testing.T) {
	now := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	created := pgtype.Timestamptz{Time: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC), Valid: true}
	due := pgtype.Date{Time: time.Date(2026, 12, 19, 0, 0, 0, 0, time.UTC), Valid: true} // 40 days
	base := db.Goal{TargetValue: 40, Status: "active", CreatedAt: created, DueDate: due}

	cases := []struct {
		current float64
		want    string
	}{
		{10, "on_track"}, // 25% done, 25% of time used
		{6, "at_risk"},   // 15% done, 25% of time: ratio 0.6
		{0, "off_track"}, // nothing done
		{40, "on_track"}, // reached
	}
	for _, c := range cases {
		g := base
		g.CurrentValue = c.current
		if _, _, health, _, _ := goalHealth(g, now); health != c.want {
			t.Errorf("current %v: health = %s, want %s", c.current, health, c.want)
		}
	}

	// Lower is better: start 0.031, target 0.020.
	cost := db.Goal{Direction: "decrease", StartValue: 0.031, TargetValue: 0.020, CurrentValue: 0.031, Status: "active", CreatedAt: created, DueDate: due}
	if progress, _, health, _, _ := goalHealth(cost, now); progress != 0 || health != "off_track" {
		t.Errorf("flat cost goal = %v %s, want 0 off_track", progress, health)
	}
	cost.CurrentValue = 0.025
	if progress, _, _, _, _ := goalHealth(cost, now); progress < 0.5 || progress > 0.6 {
		t.Errorf("cost progress = %v, want about 0.55", progress)
	}

	noDue := base
	noDue.DueDate = pgtype.Date{}
	if _, _, health, _, _ := goalHealth(noDue, now); health != "no_target_date" {
		t.Errorf("no due date health = %s", health)
	}
	closed := base
	closed.Status = "achieved"
	if _, _, health, _, _ := goalHealth(closed, now); health != "achieved" {
		t.Errorf("closed health = %s", health)
	}
}

func TestGoalCreateValueLinkDelete(t *testing.T) {
	parent := createGoalForTest(t, map[string]any{"title": "Revenue", "unit": "USD", "target_value": 5000, "due_date": "2026-12-31"})
	child := createGoalForTest(t, map[string]any{
		"title": "Paying customers", "target_value": 50, "due_date": "2026-12-31",
		"parent_id": parent.ID, "measured_by": "script",
	})
	if child.ParentID == nil || *child.ParentID != parent.ID || child.CurrentValue != 0 || child.Health == "" {
		t.Fatalf("child = %+v", child)
	}

	// A new value updates current and history.
	req := withURLParam(goalRequest("POST", "/api/goals/"+child.ID+"/values", map[string]any{"value": 6, "note": "checkout export"}), "id", child.ID)
	var updated GoalResponse
	testutil.Call(t, testHandler.AddGoalValue, req).Want(http.StatusOK).JSON(&updated)
	if updated.CurrentValue != 6 {
		t.Fatalf("current = %v, want 6", updated.CurrentValue)
	}
	var detail struct {
		Goal    GoalResponse        `json:"goal"`
		History []GoalValueResponse `json:"history"`
	}
	testutil.Call(t, testHandler.GetGoal, withURLParam(goalRequest("GET", "/api/goals/"+child.ID, nil), "id", child.ID)).
		Want(http.StatusOK).JSON(&detail)
	if len(detail.History) != 1 || detail.History[0].Value != 6 || detail.History[0].Note != "checkout export" {
		t.Fatalf("history = %+v", detail.History)
	}

	// Links to an issue; unknown targets are refused.
	issueID := dbfx.Issue(t, "goal work", testutil.Cols{"status": "todo"})
	links := withURLParam(goalRequest("PUT", "/api/goals/"+child.ID+"/links", map[string]any{
		"links": []map[string]any{{"target_type": "issue", "target_id": issueID}},
	}), "id", child.ID)
	var linked GoalResponse
	testutil.Call(t, testHandler.SetGoalLinks, links).Want(http.StatusOK).JSON(&linked)
	if len(linked.Links) != 1 || linked.Links[0].TargetID != issueID {
		t.Fatalf("links = %+v", linked.Links)
	}
	bad := withURLParam(goalRequest("PUT", "/api/goals/"+child.ID+"/links", map[string]any{
		"links": []map[string]any{{"target_type": "issue", "target_id": "00000000-0000-0000-0000-000000000001"}},
	}), "id", child.ID)
	testutil.Call(t, testHandler.SetGoalLinks, bad).Want(http.StatusBadRequest)

	// A parent with children is not deleted; the child is.
	testutil.Call(t, testHandler.DeleteGoal, withURLParam(goalRequest("DELETE", "/api/goals/"+parent.ID, nil), "id", parent.ID)).Want(http.StatusConflict)
	testutil.Call(t, testHandler.DeleteGoal, withURLParam(goalRequest("DELETE", "/api/goals/"+child.ID, nil), "id", child.ID)).Want(http.StatusNoContent)

	var list struct {
		Goals []GoalResponse `json:"goals"`
	}
	testutil.Call(t, testHandler.ListGoals, goalRequest("GET", "/api/goals", nil)).Want(http.StatusOK).JSON(&list)
	for _, g := range list.Goals {
		if g.ID == child.ID {
			t.Fatal("deleted goal still listed")
		}
	}
}

func TestGoalValidation(t *testing.T) {
	testutil.Call(t, testHandler.CreateGoal, goalRequest("POST", "/api/goals", map[string]any{"title": "No target"})).Want(http.StatusBadRequest)
	testutil.Call(t, testHandler.CreateGoal, goalRequest("POST", "/api/goals", map[string]any{"title": "x", "target_value": 1, "direction": "sideways"})).Want(http.StatusBadRequest)
	testutil.Call(t, testHandler.CreateGoal, goalRequest("POST", "/api/goals", map[string]any{"title": "x", "target_value": 1, "due_date": "31/12"})).Want(http.StatusBadRequest)
}
