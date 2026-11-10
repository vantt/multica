package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/issueworkflow"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// vantt send-back-limit: a workflow may cap how many times agents send an
// issue back along a step's back edge (review → build, QA → build) with no
// person moving the issue in between. Past the cap the server refuses the
// agent's move with 409 {code: send_back_limit}; a person's move is never
// refused and starts a fresh count. No row in vantt_workflow_limit = no cap.

// SendBackLimitError refuses one more agent send-back.
type SendBackLimitError struct {
	Status, Back string
	Count, Limit int
}

func (e *SendBackLimitError) Error() string {
	return fmt.Sprintf("this issue was already sent back %d times by agents with no person in between (limit %d); "+
		"do not send it back to %q again: leave it in %q, comment what is still in dispute and @mention a person",
		e.Count, e.Limit, e.Back, e.Status)
}

func writeSendBackLimitError(w http.ResponseWriter, err error) bool {
	var limit *SendBackLimitError
	if !errors.As(err, &limit) {
		return false
	}
	writeJSON(w, http.StatusConflict, map[string]any{
		"error":  limit.Error(),
		"code":   "send_back_limit",
		"status": limit.Status,
		"back":   limit.Back,
		"count":  limit.Count,
		"limit":  limit.Limit,
	})
	return true
}

// workflowSendBackLimit returns the workflow's cap, 0 when it has none.
func (h *Handler) workflowSendBackLimit(ctx context.Context, workflowID pgtype.UUID) (int, error) {
	var limit int
	err := h.DB.QueryRow(ctx, `SELECT max_send_backs FROM vantt_workflow_limit WHERE workflow_id = $1`, workflowID).Scan(&limit)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return limit, err
}

// refuseSendBackOverLimit runs for an agent's status change from prev.Status
// to `to` inside workflow def.
func (h *Handler) refuseSendBackOverLimit(ctx context.Context, def *issueworkflow.Definition, prev db.Issue, to string) error {
	if def == nil || !def.ID.Valid {
		return nil
	}
	step, ok := def.Step(prev.Status)
	if !ok || step.BackStatusKey == "" || step.BackStatusKey != to {
		return nil
	}
	limit, err := h.workflowSendBackLimit(ctx, def.ID)
	if err != nil || limit == 0 {
		return err
	}
	// Status moves since the last one a person made, oldest first. Each agent
	// move along a back edge of the current workflow counts.
	rows, err := h.DB.Query(ctx, `
		SELECT COALESCE(details->>'from', ''), COALESCE(details->>'to', '')
		FROM activity_log
		WHERE issue_id = $1 AND action = 'status_changed' AND actor_type = 'agent'
		  AND created_at > COALESCE((
		      SELECT MAX(created_at) FROM activity_log
		      WHERE issue_id = $1 AND action = 'status_changed' AND actor_type = 'member'
		  ), '-infinity'::timestamptz)`, prev.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var from, toKey string
		if err := rows.Scan(&from, &toKey); err != nil {
			return err
		}
		if s, ok := def.Step(from); ok && s.BackStatusKey != "" && s.BackStatusKey == toKey {
			count++
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count >= limit {
		return &SendBackLimitError{Status: prev.Status, Back: to, Count: count, Limit: limit}
	}
	return nil
}

type workflowLimitsBody struct {
	MaxSendBacks *int `json:"max_send_backs"`
}

// GetIssueWorkflowLimits answers {max_send_backs: n | null}. Any member reads.
func (h *Handler) GetIssueWorkflowLimits(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	if _, ok := h.requireWorkspaceMember(w, r, workspaceID, "workspace not found"); !ok {
		return
	}
	idUUID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "workflow id")
	if !ok {
		return
	}
	if _, err := h.Queries.GetIssueWorkflow(r.Context(), db.GetIssueWorkflowParams{ID: idUUID, WorkspaceID: wsUUID}); err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}
	limit, err := h.workflowSendBackLimit(r.Context(), idUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read workflow limits")
		return
	}
	body := workflowLimitsBody{}
	if limit > 0 {
		body.MaxSendBacks = &limit
	}
	writeJSON(w, http.StatusOK, body)
}

// SetIssueWorkflowLimits sets or clears (null or 0) the cap. Same people who
// may edit the workflow.
func (h *Handler) SetIssueWorkflowLimits(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	member, ok := h.requireWorkspaceMember(w, r, workspaceID, "workspace not found")
	if !ok {
		return
	}
	idUUID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "workflow id")
	if !ok {
		return
	}
	var body workflowLimitsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.MaxSendBacks != nil && *body.MaxSendBacks < 0 {
		writeError(w, http.StatusBadRequest, "max_send_backs must be 0 (no limit) or more")
		return
	}
	current, err := h.Queries.GetIssueWorkflow(r.Context(), db.GetIssueWorkflowParams{ID: idUUID, WorkspaceID: wsUUID})
	if err != nil {
		writeError(w, http.StatusNotFound, "workflow not found")
		return
	}
	if !canEditIssueWorkflow(member, current) {
		writeError(w, http.StatusForbidden, "only the workflow owner and workspace admins can change this workflow")
		return
	}
	if body.MaxSendBacks == nil || *body.MaxSendBacks == 0 {
		_, err = h.DB.Exec(r.Context(), `DELETE FROM vantt_workflow_limit WHERE workflow_id = $1`, idUUID)
	} else {
		_, err = h.DB.Exec(r.Context(), `
			INSERT INTO vantt_workflow_limit (workflow_id, max_send_backs) VALUES ($1, $2)
			ON CONFLICT (workflow_id) DO UPDATE SET max_send_backs = EXCLUDED.max_send_backs, updated_at = now()`,
			idUUID, *body.MaxSendBacks)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save workflow limits")
		return
	}
	if body.MaxSendBacks != nil && *body.MaxSendBacks == 0 {
		body.MaxSendBacks = nil
	}
	writeJSON(w, http.StatusOK, body)
}
