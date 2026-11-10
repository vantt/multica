package handler

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// vantt board-age (design D3): when each open issue entered its current status,
// for the board's "days in column" badge. Taken from the activity log's last
// status_changed entry, else the issue's creation. Reference: Jira's
// "days in column" on board cards.

type StatusSinceEntry struct {
	Status string `json:"status"`
	Since  string `json:"since"`
}

// ListIssueStatusSince serves GET /api/issues/status-since[?project_id=].
// Only issues whose status is not done or cancelled are listed.
func (h *Handler) ListIssueStatusSince(w http.ResponseWriter, r *http.Request) {
	wsID := h.resolveWorkspaceID(r)
	if _, ok := h.requireWorkspaceMember(w, r, wsID, "workspace not found"); !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, wsID, "workspace id")
	if !ok {
		return
	}
	var project pgtype.UUID
	if p := r.URL.Query().Get("project_id"); p != "" {
		id, ok := parseUUIDOrBadRequest(w, p, "project_id")
		if !ok {
			return
		}
		project = id
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT i.id::text, i.status, COALESCE(MAX(a.created_at), i.created_at)
		FROM issue i
		LEFT JOIN activity_log a ON a.issue_id = i.id AND a.action = 'status_changed'
		WHERE i.workspace_id = $1
		  AND ($2::uuid IS NULL OR i.project_id = $2)
		  AND i.status NOT IN ('done', 'cancelled')
		GROUP BY i.id, i.status, i.created_at`, wsUUID, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list status times")
		return
	}
	defer rows.Close()
	out := map[string]StatusSinceEntry{}
	for rows.Next() {
		var id, status string
		var since time.Time
		if err := rows.Scan(&id, &status, &since); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to read status times")
			return
		}
		out[id] = StatusSinceEntry{Status: status, Since: since.UTC().Format(time.RFC3339)}
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read status times")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"issues": out})
}
