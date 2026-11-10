package handler

// vantt decisions (patch decisions, F5): a question on an issue with options for
// a person to pick, one recommended. Reference: Jira approvals; Paperclip
// decisions. Waiting is enforced by the workflow (the issue sits in a step a
// person handles, patch owner-gates); answering records the choice, and the
// client posts it as a comment so the usual comment triggers wake the agent.

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type DecisionOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Recommended bool   `json:"recommended"`
}

type DecisionResponse struct {
	ID             string           `json:"id"`
	IssueID        string           `json:"issue_id"`
	Question       string           `json:"question"`
	Options        []DecisionOption `json:"options"`
	Status         string           `json:"status"`
	DueDate        *string          `json:"due_date"`
	AskedByType    string           `json:"asked_by_type"`
	AskedByID      *string          `json:"asked_by_id"`
	AnswerOptionID *string          `json:"answer_option_id"`
	AnswerNote     string           `json:"answer_note"`
	AnsweredBy     *string          `json:"answered_by"`
	AnsweredAt     *string          `json:"answered_at"`
	CreatedAt      string           `json:"created_at"`
}

func decisionToResponse(d db.IssueDecision) DecisionResponse {
	var options []DecisionOption
	_ = json.Unmarshal(d.Options, &options)
	if options == nil {
		options = []DecisionOption{}
	}
	var answeredAt *string
	if d.AnsweredAt.Valid {
		s := timestampToString(d.AnsweredAt)
		answeredAt = &s
	}
	return DecisionResponse{
		ID: uuidToString(d.ID), IssueID: uuidToString(d.IssueID), Question: d.Question, Options: options,
		Status: d.Status, DueDate: dateToPtr(d.DueDate), AskedByType: d.AskedByType, AskedByID: uuidToPtr(d.AskedByID),
		AnswerOptionID: textToPtr(d.AnswerOptionID), AnswerNote: d.AnswerNote, AnsweredBy: uuidToPtr(d.AnsweredBy),
		AnsweredAt: answeredAt, CreatedAt: timestampToString(d.CreatedAt),
	}
}

func (h *Handler) ListIssueDecisions(w http.ResponseWriter, r *http.Request) {
	issue, ok := h.loadIssueForUser(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	rows, err := h.Queries.ListIssueDecisions(r.Context(), db.ListIssueDecisionsParams{IssueID: issue.ID, WorkspaceID: issue.WorkspaceID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list decisions")
		return
	}
	out := make([]DecisionResponse, 0, len(rows))
	for _, d := range rows {
		out = append(out, decisionToResponse(d))
	}
	writeJSON(w, http.StatusOK, map[string]any{"decisions": out})
}

// ListOpenDecisions lists every open decision in the workspace (for the morning brief).
func (h *Handler) ListOpenDecisions(w http.ResponseWriter, r *http.Request) {
	wsID := h.resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceMember(w, r, wsID, "workspace not found"); !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, wsID, "workspace id")
	if !ok {
		return
	}
	rows, err := h.Queries.ListOpenIssueDecisions(r.Context(), wsUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list decisions")
		return
	}
	out := make([]DecisionResponse, 0, len(rows))
	for _, d := range rows {
		out = append(out, decisionToResponse(d))
	}
	writeJSON(w, http.StatusOK, map[string]any{"decisions": out})
}

type CreateDecisionRequest struct {
	Question string           `json:"question"`
	Options  []DecisionOption `json:"options"`
	DueDate  *string          `json:"due_date"`
}

// CreateIssueDecision opens a decision; members and agents may ask.
func (h *Handler) CreateIssueDecision(w http.ResponseWriter, r *http.Request) {
	issue, ok := h.loadIssueForUser(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	var req CreateDecisionRequest
	body, err := io.ReadAll(r.Body)
	if err != nil || json.Unmarshal(body, &req) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" || len(req.Options) < 2 || len(req.Options) > 8 {
		writeError(w, http.StatusBadRequest, "a decision needs a question and 2 to 8 options")
		return
	}
	recommended := 0
	for i := range req.Options {
		req.Options[i].Label = strings.TrimSpace(req.Options[i].Label)
		if req.Options[i].Label == "" {
			writeError(w, http.StatusBadRequest, "every option needs a label")
			return
		}
		req.Options[i].ID = uuid.NewString()
		if req.Options[i].Recommended {
			recommended++
		}
	}
	if recommended > 1 {
		writeError(w, http.StatusBadRequest, "at most one option is recommended")
		return
	}
	var due pgtype.Date
	if req.DueDate != nil && *req.DueDate != "" {
		d, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "due_date must be YYYY-MM-DD")
			return
		}
		due = pgtype.Date{Time: d, Valid: true}
	}
	options, _ := json.Marshal(req.Options)
	actorType, actorID := h.resolveActor(r, userID, uuidToString(issue.WorkspaceID))
	asker, _ := util.ParseUUID(actorID)
	d, err := h.Queries.CreateIssueDecision(r.Context(), db.CreateIssueDecisionParams{
		IssueID: issue.ID, WorkspaceID: issue.WorkspaceID, Question: req.Question, Options: options,
		DueDate: due, AskedByType: actorType, AskedByID: asker,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create decision")
		return
	}
	h.notifyDecision(r.Context(), issue, d, actorType, actorID) // patch 0019
	writeJSON(w, http.StatusCreated, decisionToResponse(d))
}

type AnswerDecisionRequest struct {
	OptionID string `json:"option_id"`
	Note     string `json:"note"`
}

// AnswerIssueDecision records a person's choice. Agents cannot answer.
func (h *Handler) AnswerIssueDecision(w http.ResponseWriter, r *http.Request) {
	issue, ok := h.loadIssueForUser(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	if actorType, _ := h.resolveActor(r, userID, uuidToString(issue.WorkspaceID)); actorType != "member" {
		writeError(w, http.StatusForbidden, "only a person answers a decision")
		return
	}
	decisionID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "decisionId"), "decision id")
	if !ok {
		return
	}
	cur, err := h.Queries.GetIssueDecision(r.Context(), db.GetIssueDecisionParams{ID: decisionID, WorkspaceID: issue.WorkspaceID})
	if err != nil || cur.IssueID != issue.ID {
		writeError(w, http.StatusNotFound, "decision not found")
		return
	}
	var req AnswerDecisionRequest
	body, err := io.ReadAll(r.Body)
	if err != nil || json.Unmarshal(body, &req) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	known := false
	for _, o := range decisionToResponse(cur).Options {
		known = known || o.ID == req.OptionID
	}
	if !known {
		writeError(w, http.StatusBadRequest, "option_id is not one of the decision's options")
		return
	}
	answerer, _ := util.ParseUUID(userID)
	d, err := h.Queries.AnswerIssueDecision(r.Context(), db.AnswerIssueDecisionParams{
		ID: decisionID, WorkspaceID: issue.WorkspaceID, AnswerOptionID: pgtype.Text{String: req.OptionID, Valid: true},
		AnswerNote: strings.TrimSpace(req.Note), AnsweredBy: answerer,
	})
	if err != nil {
		writeError(w, http.StatusConflict, "decision is no longer open")
		return
	}
	writeJSON(w, http.StatusOK, decisionToResponse(d))
}

func (h *Handler) CancelIssueDecision(w http.ResponseWriter, r *http.Request) {
	issue, ok := h.loadIssueForUser(w, r, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	decisionID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "decisionId"), "decision id")
	if !ok {
		return
	}
	cur, err := h.Queries.GetIssueDecision(r.Context(), db.GetIssueDecisionParams{ID: decisionID, WorkspaceID: issue.WorkspaceID})
	if err != nil || cur.IssueID != issue.ID {
		writeError(w, http.StatusNotFound, "decision not found")
		return
	}
	d, err := h.Queries.CancelIssueDecision(r.Context(), db.CancelIssueDecisionParams{ID: decisionID, WorkspaceID: issue.WorkspaceID})
	if err != nil {
		writeError(w, http.StatusConflict, "decision is no longer open")
		return
	}
	writeJSON(w, http.StatusOK, decisionToResponse(d))
}
