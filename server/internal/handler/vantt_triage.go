package handler

// vantt patch 0021 (triage-suggest): the create-issue form asks for a project
// and Type suggestion while the person types (Linear's create-form
// suggestions); see internal/triage.

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/multica-ai/multica/server/internal/triage"
)

// Triage is set at boot when MULTICA_CLOUDFLARE_AI_* is configured; nil = off.
var Triage *triage.Suggester

type TriageSuggestRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ProjectID   string `json:"project_id"`
	HasType     bool   `json:"has_type"`
}

// TriageSuggest (POST /api/issues/triage-suggest) answers 200 with only the
// suggestions the model is sure of — often none. Off, failing or unsure all
// look the same to the client: an empty object.
func (h *Handler) TriageSuggest(w http.ResponseWriter, r *http.Request) {
	if _, ok := requireUserID(w, r); !ok {
		return
	}
	workspaceID := h.resolveWorkspaceID(r)
	if workspaceID == "" {
		writeError(w, http.StatusBadRequest, "workspace_id is required")
		return
	}
	var req TriageSuggestRequest
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil || json.Unmarshal(body, &req) != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if Triage == nil || !triage.JobOn("form") || strings.TrimSpace(req.Title) == "" {
		writeJSON(w, http.StatusOK, triage.Suggestion{})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), triage.DefaultTimeout)
	defer cancel()
	sug, _ := Triage.Suggest(ctx, "form", workspaceID, req.Title, req.Description,
		triage.Need{Project: strings.TrimSpace(req.ProjectID) == "", Type: !req.HasType})
	writeJSON(w, http.StatusOK, sug)
}
