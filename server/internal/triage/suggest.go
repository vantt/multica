package triage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Querier is the slice of a pgx pool the suggester reads with.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Need says which fields the caller still lacks.
type Need struct {
	Project bool
	Type    bool
}

type ProjectSuggestion struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Probability float64 `json:"probability"`
}

type TypeSuggestion struct {
	PropertyID  string  `json:"property_id"`
	OptionID    string  `json:"option_id"`
	Label       string  `json:"label"`
	Probability float64 `json:"probability"`
}

// Suggestion holds only what cleared MinProbability.
type Suggestion struct {
	Project  *ProjectSuggestion `json:"project,omitempty"`
	Type     *TypeSuggestion    `json:"type,omitempty"`
	Provider string             `json:"provider,omitempty"`
}

// Suggester builds the question from the workspace and asks the model.
type Suggester struct {
	DB     Querier
	Client *Client
	// Exclude keeps one issue out of the examples (the live eval asks about
	// issues that already exist).
	Exclude string
}

const (
	nameMax     = 80
	descMax     = 400
	titleMax    = 100
	examples    = 4
	issueTitle  = 300
	issueDetail = 1500
)

// Job switches (MausBot: one per job; absent means on).
func JobOn(job string) bool {
	return strings.TrimSpace(strings.ToLower(os.Getenv("MULTICA_TRIAGE_"+strings.ToUpper(job)))) != "off"
}

func clip(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

type option struct{ id, name, text string }

// Suggest asks for the fields in need. A disabled model, a failure or an
// unsure answer all come back as an empty Suggestion and a nil error.
func (s *Suggester) Suggest(ctx context.Context, job, workspaceID, title, description string, need Need) (Suggestion, error) {
	if s == nil || s.Client == nil || strings.TrimSpace(title) == "" || (!need.Project && !need.Type) {
		return Suggestion{}, nil
	}
	questions := map[string]Question{}
	var projects, types []option
	var typeProp string
	if need.Project {
		var err error
		if projects, err = s.projects(ctx, workspaceID); err != nil {
			return Suggestion{}, err
		}
		if len(projects) >= 2 {
			questions["project"] = Question{
				Instructions: "Which project should `issue` go to? Each option is a project: its purpose, then recent issues in it.",
				Options:      keyed("p", projects),
			}
		}
	}
	if need.Type {
		var err error
		var guidance string
		if typeProp, guidance, types, err = s.types(ctx, workspaceID); err != nil {
			return Suggestion{}, err
		}
		if len(types) >= 2 {
			instructions := "What kind of work is `issue`? Each option is a value of the workspace's property, with recent issues that have it."
			if guidance != "" {
				instructions += " The workspace describes this property as: " + guidance
			}
			questions["type"] = Question{Instructions: instructions, Options: keyed("t", types)}
		}
	}
	if len(questions) == 0 {
		return Suggestion{}, nil
	}
	state := map[string]any{"issue": map[string]string{"title": clip(title, issueTitle), "description": clip(description, issueDetail)}}
	model := ModelFast
	if m := strings.TrimSpace(os.Getenv("MULTICA_TRIAGE_MODEL")); m == ModelCareful {
		model = m
	}
	answers, err := s.Client.Choose(ctx, job, model, state, questions)
	if err != nil {
		return Suggestion{}, nil // fail open
	}
	out := Suggestion{Provider: "clef"}
	if a, ok := answers["project"]; ok && a.PTop >= MinProbability {
		if p := pick("p", a.Choice, projects); p != nil {
			out.Project = &ProjectSuggestion{ID: p.id, Name: p.name, Probability: a.PTop}
		}
	}
	if a, ok := answers["type"]; ok && a.PTop >= MinProbability {
		if t := pick("t", a.Choice, types); t != nil {
			out.Type = &TypeSuggestion{PropertyID: typeProp, OptionID: t.id, Label: t.name, Probability: a.PTop}
		}
	}
	if out.Project == nil && out.Type == nil {
		return Suggestion{}, nil
	}
	return out, nil
}

// Short keys keep ids out of the model's input.
func keyed(prefix string, opts []option) map[string]string {
	m := map[string]string{}
	for i, o := range opts {
		m[fmt.Sprintf("%s%d", prefix, i+1)] = o.text
	}
	return m
}

func pick(prefix, key string, opts []option) *option {
	var i int
	if _, err := fmt.Sscanf(key, prefix+"%d", &i); err != nil || i < 1 || i > len(opts) {
		return nil
	}
	return &opts[i-1]
}

func describe(name, desc string, recent []string) string {
	text := clip(name, nameMax)
	if d := clip(desc, descMax); d != "" {
		text += ": " + d
	}
	if len(recent) > 0 {
		text += ". Recent: " + strings.Join(recent, "; ")
	}
	return text
}

func (s *Suggester) projects(ctx context.Context, ws string) ([]option, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT p.id::text, p.title, coalesce(p.description, ''),
		       coalesce((SELECT array_agg(t.title ORDER BY t.created_at DESC) FROM (
		           SELECT title, created_at FROM issue i WHERE i.project_id = p.id AND i.id::text <> $3
		           ORDER BY created_at DESC LIMIT $2) t), '{}')
		FROM project p
		WHERE p.workspace_id = $1 AND p.status NOT IN ('completed', 'cancelled')
		ORDER BY p.created_at`, ws, examples, s.Exclude)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []option
	for rows.Next() {
		var id, name, desc string
		var recent []string
		if err := rows.Scan(&id, &name, &desc, &recent); err != nil {
			return nil, err
		}
		for i := range recent {
			recent[i] = clip(recent[i], titleMax)
		}
		out = append(out, option{id: id, name: name, text: describe(name, desc, recent)})
	}
	return out, rows.Err()
}

// types: the select property the workspace's workflow rules use (patch 0005),
// else a select property named "Type"; its description (the workspace's own
// guidance) and its options, each with recent issues that carry it. All read
// fresh from the workspace on every call: nothing about options is built in.
func (s *Suggester) types(ctx context.Context, ws string) (string, string, []option, error) {
	var propID, propDesc string
	var config []byte
	err := s.DB.QueryRow(ctx, `
		SELECT ip.id::text, ip.config, coalesce(ip.description, '') FROM issue_property ip
		WHERE ip.workspace_id = $1 AND ip.type = 'select' AND ip.archived_at IS NULL
		  AND (ip.id IN (SELECT property_id FROM project_workflow_rule WHERE workspace_id = $1) OR lower(ip.name) = 'type')
		ORDER BY (ip.id IN (SELECT property_id FROM project_workflow_rule WHERE workspace_id = $1)) DESC, ip.position
		LIMIT 1`, ws).Scan(&propID, &config, &propDesc)
	if err == pgx.ErrNoRows {
		return "", "", nil, nil
	}
	if err != nil {
		return "", "", nil, err
	}
	var cfg struct {
		Options []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"options"`
	}
	if json.Unmarshal(config, &cfg) != nil {
		return "", "", nil, nil
	}
	recent := map[string][]string{}
	rows, err := s.DB.Query(ctx, `
		SELECT opt, title FROM (
		  SELECT properties->>$2 AS opt, title,
		         row_number() OVER (PARTITION BY properties->>$2 ORDER BY created_at DESC) AS rn
		  FROM issue WHERE workspace_id = $1 AND properties ? $2 AND id::text <> $4) t
		WHERE rn <= $3`, ws, propID, examples, s.Exclude)
	if err != nil {
		return "", "", nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var opt, title string
		if err := rows.Scan(&opt, &title); err != nil {
			return "", "", nil, err
		}
		recent[opt] = append(recent[opt], clip(title, titleMax))
	}
	names := make([]string, 0, len(cfg.Options))
	for _, o := range cfg.Options {
		names = append(names, o.Name)
	}
	guidance := optionGuidance(propDesc, names)
	var out []option
	for _, o := range cfg.Options {
		out = append(out, option{id: o.ID, name: o.Name, text: describe(o.Name, guidance[o.Name], recent[o.ID])})
	}
	return propID, clip(propDesc, descMax), out, rows.Err()
}

// optionGuidance reads what each option means from the property's own
// description, the one place a workspace already writes about it (Linear:
// admin-written guidance). One option per line, "<option>: <meaning>"; lines
// that do not start with an option's name are ignored. Nothing is hard-coded:
// options, meanings and examples are all read fresh on every call.
func optionGuidance(description string, names []string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(description, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "-*•"))
		for _, n := range names {
			if rest, ok := strings.CutPrefix(line, n); ok {
				if rest = strings.TrimSpace(strings.TrimLeft(rest, ":—-– ")); rest != "" {
					out[n] = rest
				}
			}
		}
	}
	return out
}
