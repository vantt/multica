package handler

// vantt workflow-by-type (patch workflow-by-type, F3): in a project, issues whose
// select property (e.g. Type) has a given option follow another workflow.
// Reference: Jira workflow schemes (issue type → workflow).

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/issuestatus"
	"github.com/multica-ai/multica/server/internal/issueworkflow"
	"github.com/multica-ai/multica/server/internal/util"
	"github.com/multica-ai/multica/server/pkg/protocol"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type WorkflowRuleJSON struct {
	PropertyID string `json:"property_id"`
	OptionID   string `json:"option_id"`
	WorkflowID string `json:"workflow_id"`
}

func (h *Handler) loadProjectForRules(w http.ResponseWriter, r *http.Request) (db.Project, bool) {
	wsID := h.resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceMember(w, r, wsID, "workspace not found"); !ok {
		return db.Project{}, false
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, wsID, "workspace id")
	if !ok {
		return db.Project{}, false
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "project id")
	if !ok {
		return db.Project{}, false
	}
	project, err := h.Queries.GetProjectInWorkspace(r.Context(), db.GetProjectInWorkspaceParams{ID: id, WorkspaceID: wsUUID})
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return db.Project{}, false
	}
	return project, true
}

func rulesToJSON(rules []db.ProjectWorkflowRule) []WorkflowRuleJSON {
	out := make([]WorkflowRuleJSON, 0, len(rules))
	for _, rule := range rules {
		out = append(out, WorkflowRuleJSON{PropertyID: uuidToString(rule.PropertyID), OptionID: rule.OptionID, WorkflowID: uuidToString(rule.WorkflowID)})
	}
	return out
}

func (h *Handler) ListProjectWorkflowRules(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectForRules(w, r)
	if !ok {
		return
	}
	rules, err := h.Queries.ListProjectWorkflowRules(r.Context(), db.ListProjectWorkflowRulesParams{ProjectID: project.ID, WorkspaceID: project.WorkspaceID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list rules")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": rulesToJSON(rules)})
}

// SetProjectWorkflowRules replaces a project's rules. Owner/admin only. Every
// rule names a select property of the workspace, one of its options and a
// workflow of the workspace. Existing issues keep their status; they follow
// the new workflow from their next move.
func (h *Handler) SetProjectWorkflowRules(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectForRules(w, r)
	if !ok {
		return
	}
	if _, ok := h.requireWorkspaceRole(w, r, uuidToString(project.WorkspaceID), "workspace not found", "owner", "admin"); !ok {
		return
	}
	var req struct {
		Rules []WorkflowRuleJSON `json:"rules"`
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || json.Unmarshal(body, &req) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := r.Context()
	type parsed struct {
		property, workflow pgtype.UUID
		option             string
	}
	var rules []parsed
	seen := map[string]bool{}
	for _, rule := range req.Rules {
		propID, err1 := util.ParseUUID(rule.PropertyID)
		wfID, err2 := util.ParseUUID(rule.WorkflowID)
		if err1 != nil || err2 != nil || rule.OptionID == "" {
			writeError(w, http.StatusBadRequest, "each rule needs property_id, option_id and workflow_id")
			return
		}
		prop, err := h.Queries.GetIssueProperty(ctx, db.GetIssuePropertyParams{ID: propID, WorkspaceID: project.WorkspaceID})
		if err != nil || prop.Type != "select" {
			writeError(w, http.StatusBadRequest, "property must be a select property of this workspace")
			return
		}
		cfg := parsePropertyConfig(prop.Config)
		known := false
		for _, o := range cfg.Options {
			known = known || o.ID == rule.OptionID
		}
		if !known {
			writeError(w, http.StatusBadRequest, "option is not one of the property's options")
			return
		}
		if _, err := h.Queries.GetIssueWorkflow(ctx, db.GetIssueWorkflowParams{ID: wfID, WorkspaceID: project.WorkspaceID}); err != nil {
			writeError(w, http.StatusBadRequest, "workflow not found")
			return
		}
		key := rule.PropertyID + "/" + rule.OptionID
		if seen[key] {
			writeError(w, http.StatusBadRequest, "duplicate rule for the same option")
			return
		}
		seen[key] = true
		rules = append(rules, parsed{propID, wfID, rule.OptionID})
	}
	if err := h.Queries.DeleteProjectWorkflowRules(ctx, db.DeleteProjectWorkflowRulesParams{ProjectID: project.ID, WorkspaceID: project.WorkspaceID}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set rules")
		return
	}
	for i, rule := range rules {
		if err := h.Queries.CreateProjectWorkflowRule(ctx, db.CreateProjectWorkflowRuleParams{
			ProjectID: project.ID, WorkspaceID: project.WorkspaceID, PropertyID: rule.property,
			OptionID: rule.option, WorkflowID: rule.workflow, Position: int32(i),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to set rules")
			return
		}
	}
	saved, _ := h.Queries.ListProjectWorkflowRules(ctx, db.ListProjectWorkflowRulesParams{ProjectID: project.ID, WorkspaceID: project.WorkspaceID})
	writeJSON(w, http.StatusOK, map[string]any{"rules": rulesToJSON(saved)})
}

// ruleUsesProperty reports whether the issue's project picks workflows by this property.
func (h *Handler) ruleUsesProperty(ctx context.Context, issue db.Issue, propertyID pgtype.UUID) bool {
	if !issue.ProjectID.Valid {
		return false
	}
	rules, err := h.Queries.ListProjectWorkflowRules(ctx, db.ListProjectWorkflowRulesParams{ProjectID: issue.ProjectID, WorkspaceID: issue.WorkspaceID})
	if err != nil {
		return false
	}
	for _, rule := range rules {
		if rule.PropertyID == propertyID {
			return true
		}
	}
	return false
}

// refuseAgentRuleChange stops an agent from switching an issue's workflow
// (by changing the property a rule reads) while the issue waits on a person.
func (h *Handler) refuseAgentRuleChange(ctx context.Context, issue db.Issue, propertyID pgtype.UUID, callerType string) error {
	if callerType != "agent" || !h.ruleUsesProperty(ctx, issue, propertyID) {
		return nil
	}
	def, project, err := issueworkflow.ForIssue(ctx, h.Queries, issue.WorkspaceID, issue.ProjectID, issue.Properties)
	if err != nil {
		return err
	}
	if issueworkflow.PersonGate(def, issue.Status, project, issue.CreatorType, issue.CreatorID) {
		return &issueworkflow.GateError{Status: issue.Status}
	}
	return nil
}

// followRuleWorkflow keeps an issue inside the workflow its new property value
// picks: a status the new workflow does not list moves to the equivalent one,
// as a project move does. No handoff runs; the next status change hands off.
func (h *Handler) followRuleWorkflow(r *http.Request, prev, updated db.Issue, actorType, actorID string) {
	ctx := r.Context()
	if !h.ruleUsesAny(ctx, updated) {
		return
	}
	def, _, err := issueworkflow.ForIssue(ctx, h.Queries, updated.WorkspaceID, updated.ProjectID, updated.Properties)
	if err != nil || def == nil || def.IsDefault || def.Has(updated.Status) {
		return
	}
	target := issueworkflow.MoveTarget(*def, updated.Status, func(key string) string {
		return issuestatus.Category(ctx, h.Queries, updated.WorkspaceID, key)
	})
	moved, err := h.Queries.UpdateIssueStatus(ctx, db.UpdateIssueStatusParams{ID: updated.ID, Status: target, WorkspaceID: updated.WorkspaceID})
	if err != nil {
		slog.Warn("workflow rule: remap status failed", "issue_id", uuidToString(updated.ID), "error", err)
		return
	}
	prefix := h.getIssuePrefix(ctx, moved.WorkspaceID)
	resp := issueToResponse(moved, prefix)
	h.fillStatusCategory(ctx, moved.WorkspaceID, &resp)
	h.publish(protocol.EventIssueUpdated, uuidToString(moved.WorkspaceID), actorType, actorID, map[string]any{
		"issue":          resp,
		"status_changed": true,
		"prev_status":    prev.Status,
		"creator_type":   prev.CreatorType,
		"creator_id":     uuidToString(prev.CreatorID),
	})
}

func (h *Handler) ruleUsesAny(ctx context.Context, issue db.Issue) bool {
	if !issue.ProjectID.Valid {
		return false
	}
	rules, err := h.Queries.ListProjectWorkflowRules(ctx, db.ListProjectWorkflowRulesParams{ProjectID: issue.ProjectID, WorkspaceID: issue.WorkspaceID})
	return err == nil && len(rules) > 0
}
