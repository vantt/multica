package handler

// vantt patch 0019 (telegram-push): a new decision reaches the people who must
// answer it through the inbox (and from there their chat bot), and a chat
// button answers it the same way the web panel does.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/multica-ai/multica/server/internal/events"
	"github.com/multica-ai/multica/server/internal/inbox"
	"github.com/multica-ai/multica/server/internal/integrations/telegram"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

// decisionRecipients: the issue's member subscribers, else the workspace's
// owners and admins. The asker is never told about their own question.
func (h *Handler) decisionRecipients(ctx context.Context, issue db.Issue, actorType, actorID string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		if id == "" || seen[id] || (actorType == "member" && id == actorID) {
			return
		}
		seen[id] = true
		out = append(out, id)
	}
	if subs, err := h.Queries.ListIssueSubscribers(ctx, issue.ID); err == nil {
		for _, s := range subs {
			if s.UserType == "member" {
				add(util.UUIDToString(s.UserID))
			}
		}
	}
	if len(out) == 0 {
		if ids, err := h.Queries.ListWorkspaceManagerUserIDs(ctx, issue.WorkspaceID); err == nil {
			for _, id := range ids {
				add(util.UUIDToString(id))
			}
		}
	}
	return out
}

// notifyDecision puts the decision in each recipient's inbox (action required)
// and publishes inbox:new, which chat bots forward.
func (h *Handler) notifyDecision(ctx context.Context, issue db.Issue, d db.IssueDecision, actorType, actorID string) {
	details, _ := json.Marshal(map[string]string{"decision_id": util.UUIDToString(d.ID)})
	actor, _ := util.ParseUUID(actorID) // invalid (system) stays unset
	for _, recipient := range h.decisionRecipients(ctx, issue, actorType, actorID) {
		item, inserted, err := inbox.CreateIfEligible(ctx, h.Queries, db.CreateInboxItemParams{
			WorkspaceID:   issue.WorkspaceID,
			RecipientType: "member",
			RecipientID:   util.MustParseUUID(recipient),
			Type:          telegram.InboxTypeDecisionRequested,
			Severity:      "action_required",
			IssueID:       issue.ID,
			Title:         d.Question,
			Body:          util.StrToText(issue.Title),
			ActorType:     util.StrToText(actorType),
			ActorID:       actor,
			Details:       details,
		})
		if err != nil {
			slog.Error("decision inbox creation failed", "decision_id", util.UUIDToString(d.ID), "error", err)
			continue
		}
		if !inserted || h.Bus == nil {
			continue
		}
		h.Bus.Publish(events.Event{
			Type:        protocol.EventInboxNew,
			WorkspaceID: util.UUIDToString(issue.WorkspaceID),
			ActorType:   actorType,
			ActorID:     actorID,
			Payload: map[string]any{"item": map[string]any{
				"id":             util.UUIDToString(item.ID),
				"workspace_id":   util.UUIDToString(item.WorkspaceID),
				"recipient_type": item.RecipientType,
				"recipient_id":   util.UUIDToString(item.RecipientID),
				"type":           item.Type,
				"severity":       item.Severity,
				"issue_id":       util.UUIDToPtr(item.IssueID),
				"title":          item.Title,
				"body":           util.TextToPtr(item.Body),
				"read":           item.Read,
				"archived":       item.Archived,
				"created_at":     util.TimestampToString(item.CreatedAt),
				"actor_type":     util.TextToPtr(item.ActorType),
				"actor_id":       util.UUIDToPtr(item.ActorID),
				"details":        json.RawMessage(item.Details),
				"issue_status":   issue.Status,
			}},
		})
	}
}

// AnswerDecisionAs answers a decision for a member who pressed a chat button:
// the same two calls the web panel makes (answer, then the answer as their
// comment so the issue's agent wakes), run in-process as that member. The
// caller has already proven the chat identity is bound to userID.
func (h *Handler) AnswerDecisionAs(ctx context.Context, workspaceID, userID, decisionID string, optionIndex int) (string, error) {
	wsID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return "", err
	}
	uID, err := util.ParseUUID(userID)
	if err != nil {
		return "", err
	}
	if _, err := h.Queries.GetMemberByUserAndWorkspace(ctx, db.GetMemberByUserAndWorkspaceParams{UserID: uID, WorkspaceID: wsID}); err != nil {
		return "", fmt.Errorf("not a member of the workspace: %w", err)
	}
	dID, err := util.ParseUUID(decisionID)
	if err != nil {
		return "", err
	}
	d, err := h.Queries.GetIssueDecision(ctx, db.GetIssueDecisionParams{ID: dID, WorkspaceID: wsID})
	if err != nil {
		return "", fmt.Errorf("decision not found: %w", err)
	}
	if d.Status != "open" {
		return "", telegram.ErrDecisionClosed
	}
	resp := decisionToResponse(d)
	if optionIndex < 0 || optionIndex >= len(resp.Options) {
		return "", errors.New("no such option")
	}
	option := resp.Options[optionIndex]
	issueID := util.UUIDToString(d.IssueID)

	call := func(handle http.HandlerFunc, path string, body any, params map[string]string) int {
		raw, _ := json.Marshal(body)
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(raw))).WithContext(ctx)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-User-ID", userID)
		r.Header.Set("X-Workspace-ID", workspaceID)
		rctx := chi.NewRouteContext()
		for k, v := range params {
			rctx.URLParams.Add(k, v)
		}
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()
		handle(w, r)
		return w.Code
	}
	switch code := call(h.AnswerIssueDecision, "/api/issues/"+issueID+"/decisions/"+decisionID+"/answer",
		map[string]string{"option_id": option.ID, "note": ""}, map[string]string{"id": issueID, "decisionId": decisionID}); code {
	case http.StatusOK:
	case http.StatusConflict:
		return "", telegram.ErrDecisionClosed
	default:
		return "", fmt.Errorf("answer decision: status %d", code)
	}
	if code := call(h.CreateComment, "/api/issues/"+issueID+"/comments",
		map[string]string{"content": "**" + d.Question + "** → " + option.Label}, map[string]string{"id": issueID}); code != http.StatusCreated && code != http.StatusOK {
		return option.Label, fmt.Errorf("post answer comment: status %d", code)
	}
	return option.Label, nil
}
