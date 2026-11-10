package triage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/multica-ai/multica/server/internal/integrations/channel/engine"
	"github.com/multica-ai/multica/server/internal/service"
	"github.com/multica-ai/multica/server/internal/util"
)

// ChatIssues fills project and Type on an issue a chat `/issue` command
// creates without them, when the model is sure (Linear: auto-apply per field,
// chosen by the workspace; here for chat intake only). Anything else, or any
// failure, creates the issue exactly as before.
type ChatIssues struct {
	// The chat router's issue service; every other method passes through.
	engine.IssueCreator
	Suggester *Suggester
}

func (c *ChatIssues) Create(ctx context.Context, p service.IssueCreateParams, opts service.IssueCreateOpts) (service.IssueCreateResult, error) {
	if c.Suggester == nil || c.Suggester.Client == nil || !JobOn("chat") || p.ProjectID.Valid || !strings.HasSuffix(p.OriginType.String, "_chat") {
		return c.IssueCreator.Create(ctx, p, opts)
	}
	tctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	sug, _ := c.Suggester.Suggest(tctx, "chat", util.UUIDToString(p.WorkspaceID), p.Title, p.Description.String, Need{Project: true, Type: true})
	cancel()
	if sug.Project == nil && sug.Type == nil {
		return c.IssueCreator.Create(ctx, p, opts)
	}
	filled := p
	var note []string
	if sug.Project != nil {
		if id, err := util.ParseUUID(sug.Project.ID); err == nil {
			filled.ProjectID = id
			note = append(note, fmt.Sprintf("%s %d%%", sug.Project.Name, pct(sug.Project.Probability)))
		}
	}
	if sug.Type != nil {
		if prop, err := util.ParseUUID(sug.Type.PropertyID); err == nil {
			props := map[pgtype.UUID]json.RawMessage{}
			for k, v := range p.Properties {
				props[k] = v
			}
			if _, set := props[prop]; !set {
				raw, _ := json.Marshal(sug.Type.OptionID)
				props[prop] = raw
				filled.Properties = props
				note = append(note, fmt.Sprintf("%s %d%%", sug.Type.Label, pct(sug.Type.Probability)))
			}
		}
	}
	// The suggestion stays visible on the issue (MausBot: "Picked by Clef · 94%").
	line := "_Clef xếp: " + strings.Join(note, " · ") + "_"
	desc := strings.TrimSpace(p.Description.String)
	if desc != "" {
		desc += "\n\n"
	}
	filled.Description = pgtype.Text{String: desc + line, Valid: true}
	res, err := c.IssueCreator.Create(ctx, filled, opts)
	if err != nil {
		// A project whose workflow refuses the chat's status, or any other
		// refusal: fall back to the issue as the person typed it.
		return c.IssueCreator.Create(ctx, p, opts)
	}
	return res, nil
}

func pct(p float64) int { return int(p*100 + 0.5) }
