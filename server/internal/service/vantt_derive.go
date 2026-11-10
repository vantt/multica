package service

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// vantt derived-property: a select property whose description says
// "Derived from <Source>" takes its value from the source select property's
// option name, the part before " · " ("Tech · Feature" → "Tech"). The owner
// picks Type; Domain fills itself (owner, 2026-11-10). Set only when the
// derived property has an option with that name; otherwise left alone.

var derivedFrom = regexp.MustCompile(`(?i)derived from\s+([^.(\n]+)`)

type selectOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func optionsOf(config []byte) []selectOption {
	var c struct {
		Options []selectOption `json:"options"`
	}
	_ = json.Unmarshal(config, &c)
	return c.Options
}

// DerivedPropertyValues returns the derived values implied by props (an issue's
// properties bag): derived property id → option id (JSON string). Only changes
// are returned.
func DerivedPropertyValues(ctx context.Context, q *db.Queries, workspaceID pgtype.UUID, props []byte) (map[string]json.RawMessage, error) {
	bag := map[string]json.RawMessage{}
	if len(props) > 0 {
		_ = json.Unmarshal(props, &bag)
	}
	if len(bag) == 0 {
		return nil, nil
	}
	defs, err := q.ListIssueProperties(ctx, db.ListIssuePropertiesParams{WorkspaceID: workspaceID})
	if err != nil {
		return nil, err
	}
	byName := map[string]db.ListIssuePropertiesRow{}
	for _, d := range defs {
		byName[strings.ToLower(strings.TrimSpace(d.Name))] = d
	}
	out := map[string]json.RawMessage{}
	for _, d := range defs {
		m := derivedFrom.FindStringSubmatch(d.Description)
		if d.Type != "select" || m == nil {
			continue
		}
		src, ok := byName[strings.ToLower(strings.TrimSpace(m[1]))]
		if !ok || src.Type != "select" {
			continue
		}
		var picked string
		if raw, ok := bag[uuidString(src.ID)]; !ok || json.Unmarshal(raw, &picked) != nil || picked == "" {
			continue
		}
		var name string
		for _, o := range optionsOf(src.Config) {
			if o.ID == picked {
				name = o.Name
			}
		}
		head := strings.TrimSpace(strings.SplitN(name, "·", 2)[0])
		for _, o := range optionsOf(d.Config) {
			if head != "" && strings.EqualFold(o.Name, head) {
				var cur string
				_ = json.Unmarshal(bag[uuidString(d.ID)], &cur)
				if cur != o.ID {
					v, _ := json.Marshal(o.ID)
					out[uuidString(d.ID)] = v
				}
			}
		}
	}
	return out, nil
}

// ApplyDerivedProperties merges the derived values into a properties bag.
func ApplyDerivedProperties(ctx context.Context, q *db.Queries, workspaceID pgtype.UUID, props []byte) ([]byte, error) {
	derived, err := DerivedPropertyValues(ctx, q, workspaceID, props)
	if err != nil || len(derived) == 0 {
		return props, err
	}
	bag := map[string]json.RawMessage{}
	_ = json.Unmarshal(props, &bag)
	for k, v := range derived {
		bag[k] = v
	}
	return json.Marshal(bag)
}
