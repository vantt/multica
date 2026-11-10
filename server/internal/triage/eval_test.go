package triage

// Live eval against the real model; spends real (tiny) money, so it runs only
// when asked, like OpenMausBot's room-routing.eval.test.ts:
//
//	TRIAGE_LIVE_EVAL=1 TRIAGE_EVAL_DATABASE_URL=postgres://… TRIAGE_EVAL_WORKSPACE=<uuid> \
//	MULTICA_CLOUDFLARE_AI_ACCOUNT_ID=… MULTICA_CLOUDFLARE_AI_TOKEN=… go test ./internal/triage/ -run LiveEval -v
//
// Every issue that already has a project and a Type is asked about with its
// answers hidden (and kept out of the examples); the report says how often a
// suggestion at >= MinProbability was right, and how often it stayed silent.

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLiveEval(t *testing.T) {
	if os.Getenv("TRIAGE_LIVE_EVAL") != "1" {
		t.Skip("set TRIAGE_LIVE_EVAL=1 for the live eval")
	}
	ctx := context.Background()
	client := FromEnv(nil)
	if client == nil {
		t.Fatal("set MULTICA_CLOUDFLARE_AI_ACCOUNT_ID and MULTICA_CLOUDFLARE_AI_TOKEN")
	}
	pool, err := pgxpool.New(ctx, os.Getenv("TRIAGE_EVAL_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ws := os.Getenv("TRIAGE_EVAL_WORKSPACE")
	s := &Suggester{DB: pool, Client: client}
	typeProp, _, _, err := s.types(ctx, ws)
	if err != nil || typeProp == "" {
		t.Fatalf("no Type property: %v", err)
	}
	// Cases: a fixture of requests the way the owner writes them
	// (TRIAGE_EVAL_CASES, names resolved here), else the workspace's own
	// issues that already have a project and a Type.
	type c struct{ id, title, desc, project, typ string }
	var cases []c
	if path := os.Getenv("TRIAGE_EVAL_CASES"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var fx []struct{ Title, Description, Project, Type string }
		if err := json.Unmarshal(raw, &fx); err != nil {
			t.Fatal(err)
		}
		for _, f := range fx {
			x := c{title: f.Title, desc: f.Description}
			_ = pool.QueryRow(ctx, `SELECT id::text FROM project WHERE workspace_id = $1 AND title = $2`, ws, f.Project).Scan(&x.project)
			_ = pool.QueryRow(ctx, `SELECT o->>'id' FROM issue_property, jsonb_array_elements(config->'options') o
				WHERE id::text = $1 AND o->>'name' = $2`, typeProp, f.Type).Scan(&x.typ)
			cases = append(cases, x)
		}
	} else {
		rows, err := pool.Query(ctx, `SELECT id::text, title, coalesce(description, ''), project_id::text, properties->>$2
			FROM issue WHERE workspace_id = $1 AND project_id IS NOT NULL AND properties ? $2
			ORDER BY created_at DESC LIMIT 40`, ws, typeProp)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var x c
			if err := rows.Scan(&x.id, &x.title, &x.desc, &x.project, &x.typ); err != nil {
				t.Fatal(err)
			}
			cases = append(cases, x)
		}
		rows.Close()
	}
	var pRight, pWrong, pSilent, tRight, tWrong, tSilent int
	for _, x := range cases {
		s.Exclude = x.id
		got, _ := s.Suggest(ctx, "eval", ws, x.title, x.desc, Need{Project: true, Type: true})
		switch {
		case got.Project == nil:
			pSilent++
			t.Logf("project silent: %q", x.title)
		case got.Project.ID == x.project:
			pRight++
		default:
			pWrong++
			t.Logf("project wrong: %q → %s (%.2f)", x.title, got.Project.Name, got.Project.Probability)
		}
		switch {
		case got.Type == nil:
			tSilent++
			t.Logf("type silent: %q", x.title)
		case got.Type.OptionID == x.typ:
			tRight++
		default:
			tWrong++
			t.Logf("type wrong: %q → %s (%.2f)", x.title, got.Type.Label, got.Type.Probability)
		}
	}
	t.Logf("%d issues · project: %d right, %d wrong, %d silent · type: %d right, %d wrong, %d silent",
		len(cases), pRight, pWrong, pSilent, tRight, tWrong, tSilent)
}
