package handler

// vantt goals (patch goals, F2): results with a number and a deadline that work
// moves. Reference: Atlassian Goals (On track / At risk / Off track), Linear
// initiatives. Health and pace are computed here so every client (the web,
// the morning brief) reads the same numbers.

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const goalUpdateDueAfter = 7 * 24 * time.Hour

type GoalLinkJSON struct {
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
}

type GoalResponse struct {
	ID               string         `json:"id"`
	WorkspaceID      string         `json:"workspace_id"`
	ParentID         *string        `json:"parent_id"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Unit             string         `json:"unit"`
	Direction        string         `json:"direction"`
	StartValue       float64        `json:"start_value"`
	TargetValue      float64        `json:"target_value"`
	CurrentValue     float64        `json:"current_value"`
	DueDate          *string        `json:"due_date"`
	Status           string         `json:"status"`
	MeasuredBy       string         `json:"measured_by"`
	OwnerType        *string        `json:"owner_type"`
	OwnerID          *string        `json:"owner_id"`
	Position         float64        `json:"position"`
	CurrentUpdatedAt string         `json:"current_updated_at"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
	Links            []GoalLinkJSON `json:"links"`
	// Repo is the source of truth (vantt): where the goal lives, and whether the
	// UI changed it since the last write-back.
	SourceRef *string `json:"source_ref"`
	Pending   bool    `json:"pending"`
	// Computed.
	Progress      float64  `json:"progress"`
	Expected      *float64 `json:"expected"`
	Health        string   `json:"health"`
	UpdateDue     bool     `json:"update_due"`
	NeededPerWeek *float64 `json:"needed_per_week"`
	ActualPerWeek *float64 `json:"actual_per_week"`
}

type GoalValueResponse struct {
	ID        string  `json:"id"`
	Value     float64 `json:"value"`
	Note      string  `json:"note"`
	ActorType string  `json:"actor_type"`
	ActorID   *string `json:"actor_id"`
	Synced    bool    `json:"synced"`
	CreatedAt string  `json:"created_at"`
}

// goalHealth computes progress, the share of time used, and the health label.
func goalHealth(g db.Goal, now time.Time) (progress float64, expected *float64, health string, neededPerWeek, actualPerWeek *float64) {
	span := g.TargetValue - g.StartValue
	done := g.CurrentValue - g.StartValue
	if span != 0 {
		progress = math.Max(0, math.Min(1, done/span))
	} else if g.CurrentValue == g.TargetValue {
		progress = 1
	}
	switch g.Status {
	case "achieved", "missed", "cancelled", "planned":
		health = g.Status
	}
	created := g.CreatedAt.Time
	weeksElapsed := now.Sub(created).Hours() / (24 * 7)
	if weeksElapsed >= 1.0/7 {
		a := done / weeksElapsed
		actualPerWeek = &a
	}
	if !g.DueDate.Valid {
		if health == "" {
			health = "no_target_date"
		}
		return
	}
	due := g.DueDate.Time.Add(24 * time.Hour) // end of the due day
	total := due.Sub(created)
	if total > 0 {
		e := math.Max(0, math.Min(1, now.Sub(created).Seconds()/total.Seconds()))
		expected = &e
	}
	weeksLeft := due.Sub(now).Hours() / (24 * 7)
	if weeksLeft > 0 {
		n := (g.TargetValue - g.CurrentValue) / weeksLeft
		neededPerWeek = &n
	}
	if health != "" {
		return
	}
	if progress >= 1 {
		health = "on_track"
		return
	}
	if expected == nil {
		health = "on_track"
		return
	}
	// Progress against the share of time used: early on everything is on track.
	if *expected < 0.05 {
		health = "on_track"
		return
	}
	switch ratio := progress / *expected; {
	case ratio >= 0.9:
		health = "on_track"
	case ratio >= 0.6:
		health = "at_risk"
	default:
		health = "off_track"
	}
	return
}

func goalToResponse(g db.Goal, links []GoalLinkJSON, now time.Time) GoalResponse {
	progress, expected, health, needed, actual := goalHealth(g, now)
	if links == nil {
		links = []GoalLinkJSON{}
	}
	return GoalResponse{
		ID:               uuidToString(g.ID),
		WorkspaceID:      uuidToString(g.WorkspaceID),
		ParentID:         uuidToPtr(g.ParentID),
		Title:            g.Title,
		Description:      g.Description,
		Unit:             g.Unit,
		Direction:        g.Direction,
		StartValue:       g.StartValue,
		TargetValue:      g.TargetValue,
		CurrentValue:     g.CurrentValue,
		DueDate:          dateToPtr(g.DueDate),
		Status:           g.Status,
		MeasuredBy:       g.MeasuredBy,
		OwnerType:        textToPtr(g.OwnerType),
		OwnerID:          uuidToPtr(g.OwnerID),
		Position:         g.Position,
		CurrentUpdatedAt: timestampToString(g.CurrentUpdatedAt),
		CreatedAt:        timestampToString(g.CreatedAt),
		UpdatedAt:        timestampToString(g.UpdatedAt),
		Links:            links,
		SourceRef:        textToPtr(g.SourceRef),
		Pending:          g.Pending,
		Progress:         progress,
		Expected:         expected,
		Health:           health,
		UpdateDue:        g.MeasuredBy == "manual" && isOpenGoal(g.Status) && now.Sub(g.CurrentUpdatedAt.Time) > goalUpdateDueAfter,
		NeededPerWeek:    needed,
		ActualPerWeek:    actual,
	}
}

func isOpenGoal(status string) bool { return status == "active" || status == "planned" }

func (h *Handler) goalLinksByGoal(r *http.Request, wsUUID pgtype.UUID) (map[string][]GoalLinkJSON, error) {
	rows, err := h.Queries.ListGoalLinks(r.Context(), wsUUID)
	if err != nil {
		return nil, err
	}
	out := map[string][]GoalLinkJSON{}
	for _, l := range rows {
		k := uuidToString(l.GoalID)
		out[k] = append(out[k], GoalLinkJSON{TargetType: l.TargetType, TargetID: uuidToString(l.TargetID)})
	}
	return out, nil
}

func (h *Handler) ListGoals(w http.ResponseWriter, r *http.Request) {
	wsID := h.resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceMember(w, r, wsID, "workspace not found"); !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, wsID, "workspace id")
	if !ok {
		return
	}
	goals, err := h.Queries.ListGoals(r.Context(), wsUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list goals")
		return
	}
	links, err := h.goalLinksByGoal(r, wsUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list goal links")
		return
	}
	now := time.Now()
	resp := make([]GoalResponse, 0, len(goals))
	for _, g := range goals {
		resp = append(resp, goalToResponse(g, links[uuidToString(g.ID)], now))
	}
	writeJSON(w, http.StatusOK, map[string]any{"goals": resp, "total": len(resp)})
}

func (h *Handler) loadGoal(w http.ResponseWriter, r *http.Request) (db.Goal, pgtype.UUID, bool) {
	wsID := h.resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceMember(w, r, wsID, "workspace not found"); !ok {
		return db.Goal{}, pgtype.UUID{}, false
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, wsID, "workspace id")
	if !ok {
		return db.Goal{}, pgtype.UUID{}, false
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "goal id")
	if !ok {
		return db.Goal{}, pgtype.UUID{}, false
	}
	g, err := h.Queries.GetGoalInWorkspace(r.Context(), db.GetGoalInWorkspaceParams{ID: id, WorkspaceID: wsUUID})
	if err != nil {
		writeError(w, http.StatusNotFound, "goal not found")
		return db.Goal{}, pgtype.UUID{}, false
	}
	return g, wsUUID, true
}

func (h *Handler) GetGoal(w http.ResponseWriter, r *http.Request) {
	g, wsUUID, ok := h.loadGoal(w, r)
	if !ok {
		return
	}
	links, err := h.goalLinksByGoal(r, wsUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list goal links")
		return
	}
	values, err := h.Queries.ListGoalValues(r.Context(), db.ListGoalValuesParams{GoalID: g.ID, WorkspaceID: wsUUID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list goal values")
		return
	}
	history := make([]GoalValueResponse, 0, len(values))
	for _, v := range values {
		history = append(history, GoalValueResponse{
			ID: uuidToString(v.ID), Value: v.Value, Note: v.Note, ActorType: v.ActorType,
			ActorID: uuidToPtr(v.ActorID), Synced: v.Synced, CreatedAt: timestampToString(v.CreatedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"goal": goalToResponse(g, links[uuidToString(g.ID)], time.Now()), "history": history})
}

// GoalWriteRequest is the body of create and update; every field is optional on update.
type GoalWriteRequest struct {
	ParentID    *string  `json:"parent_id"`
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Unit        *string  `json:"unit"`
	Direction   *string  `json:"direction"`
	StartValue  *float64 `json:"start_value"`
	TargetValue *float64 `json:"target_value"`
	DueDate     *string  `json:"due_date"`
	Status      *string  `json:"status"`
	MeasuredBy  *string  `json:"measured_by"`
	OwnerType   *string  `json:"owner_type"`
	OwnerID     *string  `json:"owner_id"`
	Position    *float64 `json:"position"`
	SourceRef   *string  `json:"source_ref"`
	Pending     *bool    `json:"pending"`
}

func validGoalEnum(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

// applyGoalWrite merges req into g and validates it. raw tells which nullable
// fields (parent_id, due_date, owner) were sent explicitly as null.
func (h *Handler) applyGoalWrite(r *http.Request, wsUUID pgtype.UUID, g *db.Goal, req GoalWriteRequest, raw map[string]json.RawMessage) (int, string) {
	if req.Title != nil {
		g.Title = strings.TrimSpace(*req.Title)
	}
	if g.Title == "" {
		return http.StatusBadRequest, "title is required"
	}
	if req.Description != nil {
		g.Description = *req.Description
	}
	if req.Unit != nil {
		g.Unit = strings.TrimSpace(*req.Unit)
	}
	if req.Direction != nil {
		g.Direction = *req.Direction
	}
	if !validGoalEnum(g.Direction, "increase", "decrease") {
		return http.StatusBadRequest, "direction must be increase or decrease"
	}
	if req.StartValue != nil {
		g.StartValue = *req.StartValue
	}
	if req.TargetValue != nil {
		g.TargetValue = *req.TargetValue
	}
	if req.Status != nil {
		g.Status = *req.Status
	}
	if !validGoalEnum(g.Status, "planned", "active", "achieved", "missed", "cancelled") {
		return http.StatusBadRequest, "invalid status"
	}
	if req.MeasuredBy != nil {
		g.MeasuredBy = *req.MeasuredBy
	}
	if !validGoalEnum(g.MeasuredBy, "manual", "script") {
		return http.StatusBadRequest, "measured_by must be manual or script"
	}
	if req.Position != nil {
		g.Position = *req.Position
	}
	if _, sent := raw["source_ref"]; sent {
		g.SourceRef = pgtype.Text{}
		if req.SourceRef != nil && strings.TrimSpace(*req.SourceRef) != "" {
			g.SourceRef = pgtype.Text{String: strings.TrimSpace(*req.SourceRef), Valid: true}
		}
	}
	if _, sent := raw["due_date"]; sent {
		g.DueDate = pgtype.Date{}
		if req.DueDate != nil && *req.DueDate != "" {
			d, err := time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				return http.StatusBadRequest, "due_date must be YYYY-MM-DD"
			}
			g.DueDate = pgtype.Date{Time: d, Valid: true}
		}
	}
	if _, sent := raw["parent_id"]; sent {
		g.ParentID = pgtype.UUID{}
		if req.ParentID != nil && *req.ParentID != "" {
			pid, err := util.ParseUUID(*req.ParentID)
			if err != nil {
				return http.StatusBadRequest, "invalid parent_id"
			}
			if g.ID.Valid && pid == g.ID {
				return http.StatusBadRequest, "a goal cannot be its own parent"
			}
			if _, err := h.Queries.GetGoalInWorkspace(r.Context(), db.GetGoalInWorkspaceParams{ID: pid, WorkspaceID: wsUUID}); err != nil {
				return http.StatusBadRequest, "parent goal not found"
			}
			g.ParentID = pid
		}
	}
	_, ownerTypeSent := raw["owner_type"]
	_, ownerIDSent := raw["owner_id"]
	if ownerTypeSent || ownerIDSent {
		g.OwnerType, g.OwnerID = pgtype.Text{}, pgtype.UUID{}
		if req.OwnerType != nil && req.OwnerID != nil && *req.OwnerID != "" {
			if !validGoalEnum(*req.OwnerType, "member", "agent") || !h.isWorkspaceEntity(r.Context(), *req.OwnerType, *req.OwnerID, uuidToString(wsUUID)) {
				return http.StatusBadRequest, "owner not found in this workspace"
			}
			oid, _ := util.ParseUUID(*req.OwnerID)
			g.OwnerType = pgtype.Text{String: *req.OwnerType, Valid: true}
			g.OwnerID = oid
		}
	}
	return 0, ""
}

// goalWriter checks who may write a goal and reports whether the caller is an
// agent. Owners/admins always; an agent (the PM mirroring the repo) when the
// goal comes from the repo (source_ref).
func (h *Handler) goalWriter(w http.ResponseWriter, r *http.Request, wsUUID pgtype.UUID, fromRepo bool) (agent bool, ok bool) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return false, false
	}
	actorType, _ := h.resolveActor(r, userID, uuidToString(wsUUID))
	if actorType == "agent" {
		if !fromRepo {
			writeError(w, http.StatusForbidden, "agents only write goals that come from the project repo (source_ref)")
			return true, false
		}
		return true, true
	}
	if _, ok := h.requireWorkspaceRole(w, r, uuidToString(wsUUID), "workspace not found", "owner", "admin"); !ok {
		return false, false
	}
	return false, true
}

func decodeGoalWrite(w http.ResponseWriter, r *http.Request) (GoalWriteRequest, map[string]json.RawMessage, bool) {
	var raw map[string]json.RawMessage
	var req GoalWriteRequest
	body, err := io.ReadAll(r.Body)
	if err != nil || json.Unmarshal(body, &raw) != nil || json.Unmarshal(body, &req) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return req, nil, false
	}
	return req, raw, true
}

func (h *Handler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	wsID := h.resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceMember(w, r, wsID, "workspace not found"); !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, wsID, "workspace id")
	if !ok {
		return
	}
	req, raw, ok := decodeGoalWrite(w, r)
	if !ok {
		return
	}
	agent, ok := h.goalWriter(w, r, wsUUID, req.SourceRef != nil && strings.TrimSpace(*req.SourceRef) != "")
	if !ok {
		return
	}
	if req.TargetValue == nil {
		writeError(w, http.StatusBadRequest, "target_value is required")
		return
	}
	g := db.Goal{Direction: "increase", Status: "active", MeasuredBy: "manual"}
	if code, msg := h.applyGoalWrite(r, wsUUID, &g, req, raw); code != 0 {
		writeError(w, code, msg)
		return
	}
	created, err := h.Queries.CreateGoal(r.Context(), db.CreateGoalParams{
		WorkspaceID: wsUUID, ParentID: g.ParentID, Title: g.Title, Description: g.Description, Unit: g.Unit,
		Direction: g.Direction, StartValue: g.StartValue, TargetValue: g.TargetValue, CurrentValue: g.StartValue,
		DueDate: g.DueDate, Status: g.Status, MeasuredBy: g.MeasuredBy, OwnerType: g.OwnerType, OwnerID: g.OwnerID,
		Position: g.Position, SourceRef: g.SourceRef,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create goal")
		return
	}
	// A goal made in the UI is a proposal until the PM writes it into the repo.
	if !agent {
		created, _ = h.Queries.UpdateGoal(r.Context(), updateParams(created, true))
	}
	writeJSON(w, http.StatusCreated, goalToResponse(created, nil, time.Now()))
}

func (h *Handler) UpdateGoal(w http.ResponseWriter, r *http.Request) {
	g, wsUUID, ok := h.loadGoal(w, r)
	if !ok {
		return
	}
	agent, ok := h.goalWriter(w, r, wsUUID, g.SourceRef.Valid)
	if !ok {
		return
	}
	req, raw, ok := decodeGoalWrite(w, r)
	if !ok {
		return
	}
	if code, msg := h.applyGoalWrite(r, wsUUID, &g, req, raw); code != 0 {
		writeError(w, code, msg)
		return
	}
	// A person's edit waits to be written back to the repo; the PM's mirror
	// clears it by sending pending=false.
	pending := g.Pending || !agent
	if agent && req.Pending != nil {
		pending = *req.Pending
	}
	updated, err := h.Queries.UpdateGoal(r.Context(), updateParams(g, pending))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update goal")
		return
	}
	links, _ := h.goalLinksByGoal(r, wsUUID)
	writeJSON(w, http.StatusOK, goalToResponse(updated, links[uuidToString(updated.ID)], time.Now()))
}

func updateParams(g db.Goal, pending bool) db.UpdateGoalParams {
	return db.UpdateGoalParams{
		ID: g.ID, WorkspaceID: g.WorkspaceID, ParentID: g.ParentID, Title: g.Title, Description: g.Description,
		Unit: g.Unit, Direction: g.Direction, StartValue: g.StartValue, TargetValue: g.TargetValue,
		DueDate: g.DueDate, Status: g.Status, MeasuredBy: g.MeasuredBy, OwnerType: g.OwnerType,
		OwnerID: g.OwnerID, Position: g.Position, SourceRef: g.SourceRef, Pending: pending,
	}
}

func (h *Handler) DeleteGoal(w http.ResponseWriter, r *http.Request) {
	g, wsUUID, ok := h.loadGoal(w, r)
	if !ok {
		return
	}
	if _, ok := h.requireWorkspaceRole(w, r, uuidToString(wsUUID), "workspace not found", "owner", "admin"); !ok {
		return
	}
	children, err := h.Queries.CountGoalChildren(r.Context(), db.CountGoalChildrenParams{ParentID: g.ID, WorkspaceID: wsUUID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete goal")
		return
	}
	if children > 0 {
		writeError(w, http.StatusConflict, "goal has child goals; move or delete them first")
		return
	}
	ctx := r.Context()
	if err := h.Queries.DeleteGoalLinks(ctx, db.DeleteGoalLinksParams{GoalID: g.ID, WorkspaceID: wsUUID}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete goal")
		return
	}
	if err := h.Queries.DeleteGoalValues(ctx, db.DeleteGoalValuesParams{GoalID: g.ID, WorkspaceID: wsUUID}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete goal")
		return
	}
	if err := h.Queries.DeleteGoal(ctx, db.DeleteGoalParams{ID: g.ID, WorkspaceID: wsUUID}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete goal")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type GoalValueRequest struct {
	Value *float64 `json:"value"`
	Note  string   `json:"note"`
	// RecordedAt (agents only): the date the repo's progress row carries.
	RecordedAt *string `json:"recorded_at"`
}

// AddGoalValue records a new current value. Owners and admins may always;
// any workspace member, or an agent with a task token, may when the goal is
// measured by script (the PM's daily run).
func (h *Handler) AddGoalValue(w http.ResponseWriter, r *http.Request) {
	g, wsUUID, ok := h.loadGoal(w, r)
	if !ok {
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	actorType, actorID := h.resolveActor(r, userID, uuidToString(wsUUID))
	if actorType == "agent" {
		if g.MeasuredBy != "script" && !g.SourceRef.Valid {
			writeError(w, http.StatusForbidden, "agents only record values of script or repo goals")
			return
		}
	} else if _, ok := h.requireWorkspaceRole(w, r, uuidToString(wsUUID), "workspace not found", "owner", "admin"); !ok {
		return
	}
	var req GoalValueRequest
	body, err := io.ReadAll(r.Body)
	if err != nil || json.Unmarshal(body, &req) != nil || req.Value == nil || math.IsNaN(*req.Value) || math.IsInf(*req.Value, 0) {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}
	actor, _ := util.ParseUUID(actorID)
	var recorded pgtype.Timestamptz
	if actorType == "agent" && req.RecordedAt != nil && *req.RecordedAt != "" {
		ts, err := time.Parse(time.RFC3339, *req.RecordedAt)
		if err != nil {
			d, derr := time.Parse("2006-01-02", *req.RecordedAt)
			if derr != nil {
				writeError(w, http.StatusBadRequest, "recorded_at must be RFC3339 or YYYY-MM-DD")
				return
			}
			ts = d
		}
		recorded = pgtype.Timestamptz{Time: ts, Valid: true}
	}
	// A number a person enters on a repo goal waits to be written back.
	synced := actorType == "agent" || !g.SourceRef.Valid
	if _, err := h.Queries.CreateGoalValue(r.Context(), db.CreateGoalValueParams{
		GoalID: g.ID, WorkspaceID: wsUUID, Value: *req.Value, Note: req.Note, ActorType: actorType, ActorID: actor,
		Synced: synced, RecordedAt: recorded,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record value")
		return
	}
	updated, err := h.Queries.SetGoalCurrent(r.Context(), db.SetGoalCurrentParams{ID: g.ID, WorkspaceID: wsUUID, CurrentValue: *req.Value})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to record value")
		return
	}
	links, _ := h.goalLinksByGoal(r, wsUUID)
	writeJSON(w, http.StatusOK, goalToResponse(updated, links[uuidToString(updated.ID)], time.Now()))
}

// MarkGoalValuesSynced records that values entered in the UI are now in the repo.
func (h *Handler) MarkGoalValuesSynced(w http.ResponseWriter, r *http.Request) {
	g, wsUUID, ok := h.loadGoal(w, r)
	if !ok {
		return
	}
	if _, ok := h.goalWriter(w, r, wsUUID, true); !ok {
		return
	}
	var req struct {
		IDs []string `json:"ids"`
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || json.Unmarshal(body, &req) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ids := make([]pgtype.UUID, 0, len(req.IDs))
	for _, s := range req.IDs {
		id, err := util.ParseUUID(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		ids = append(ids, id)
	}
	if err := h.Queries.MarkGoalValuesSynced(r.Context(), db.MarkGoalValuesSyncedParams{GoalID: g.ID, WorkspaceID: wsUUID, Ids: ids}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to mark values")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type GoalLinksRequest struct {
	Links []GoalLinkJSON `json:"links"`
}

// SetGoalLinks replaces the work linked to a goal (projects, epic issues).
func (h *Handler) SetGoalLinks(w http.ResponseWriter, r *http.Request) {
	g, wsUUID, ok := h.loadGoal(w, r)
	if !ok {
		return
	}
	if _, ok := h.goalWriter(w, r, wsUUID, g.SourceRef.Valid); !ok {
		return
	}
	var req GoalLinksRequest
	body, err := io.ReadAll(r.Body)
	if err != nil || json.Unmarshal(body, &req) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := r.Context()
	type link struct {
		kind string
		id   pgtype.UUID
	}
	var links []link
	for _, l := range req.Links {
		id, err := util.ParseUUID(l.TargetID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid target_id")
			return
		}
		switch l.TargetType {
		case "project":
			if _, err := h.Queries.GetProjectInWorkspace(ctx, db.GetProjectInWorkspaceParams{ID: id, WorkspaceID: wsUUID}); err != nil {
				writeError(w, http.StatusBadRequest, "project not found")
				return
			}
		case "issue":
			if _, err := h.Queries.GetIssueInWorkspace(ctx, db.GetIssueInWorkspaceParams{ID: id, WorkspaceID: wsUUID}); err != nil {
				writeError(w, http.StatusBadRequest, "issue not found")
				return
			}
		default:
			writeError(w, http.StatusBadRequest, "target_type must be project or issue")
			return
		}
		links = append(links, link{l.TargetType, id})
	}
	if err := h.Queries.DeleteGoalLinks(ctx, db.DeleteGoalLinksParams{GoalID: g.ID, WorkspaceID: wsUUID}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set links")
		return
	}
	for _, l := range links {
		if err := h.Queries.CreateGoalLink(ctx, db.CreateGoalLinkParams{GoalID: g.ID, WorkspaceID: wsUUID, TargetType: l.kind, TargetID: l.id}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to set links")
			return
		}
	}
	all, _ := h.goalLinksByGoal(r, wsUUID)
	writeJSON(w, http.StatusOK, goalToResponse(g, all[uuidToString(g.ID)], time.Now()))
}
