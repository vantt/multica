// Package triage suggests a new issue's project and Type with a fast decision
// model (vantt patch 0021).
//
// Learned from OpenMausBot's decider (server/decider): Cloudflare's Clef on
// Workers AI takes and returns Jev's "System One" shape — choice questions with
// `instructions` + `criteria`, answers with a choice and calibrated
// probabilities. It fails open (any problem = no suggestion), every answer is
// validated, thresholds use the top option's probability, and a log line never
// holds the issue text or the key. From Linear's Triage Intelligence: suggest
// project and team-like fields, accept or dismiss, auto-apply only where a
// workspace chose it.
package triage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.cloudflare.com/client/v4"
	// ModelFast answers in well under a second; ModelCareful is for hard cases.
	ModelFast    = "clef-flash"
	ModelCareful = "clef"
	// DefaultTimeout bounds one decision; the caller goes on without it.
	DefaultTimeout = 1500 * time.Millisecond
	// MinProbability: OpenMausBot's bench found answers at >= 0.6 right 92–99%
	// of the time; below it there is no suggestion.
	MinProbability = 0.6
	maxOptions     = 255
	sumTolerance   = 0.1
)

// Question is a choice question: option key → what it means.
type Question struct {
	Instructions string
	Options      map[string]string
}

// Answer is a validated choice.
type Answer struct {
	Choice        string
	PTop          float64
	Margin        float64
	Probabilities map[string]float64
}

// ErrOff means no account/token is configured: callers simply skip.
var ErrOff = errors.New("triage: decision model not configured")

// Client calls Clef on Workers AI.
type Client struct {
	accountID string
	token     string
	baseURL   string
	http      *http.Client
	logger    *slog.Logger
}

var accountRE = regexp.MustCompile(`^[0-9a-fA-F]{32}$`)

// FromEnv reads MULTICA_CLOUDFLARE_AI_ACCOUNT_ID and MULTICA_CLOUDFLARE_AI_TOKEN
// (MULTICA_CLOUDFLARE_AI_BASE_URL for tests). Nil when either is missing.
func FromEnv(logger *slog.Logger) *Client {
	acct := strings.TrimSpace(os.Getenv("MULTICA_CLOUDFLARE_AI_ACCOUNT_ID"))
	token := strings.TrimSpace(os.Getenv("MULTICA_CLOUDFLARE_AI_TOKEN"))
	if !accountRE.MatchString(acct) || token == "" {
		return nil
	}
	return New(acct, token, os.Getenv("MULTICA_CLOUDFLARE_AI_BASE_URL"), logger)
}

func New(accountID, token, baseURL string, logger *slog.Logger) *Client {
	if baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/"); baseURL == "" {
		baseURL = defaultBaseURL
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		accountID: strings.ToLower(accountID), token: token, baseURL: baseURL, logger: logger,
		// Never replay the token to wherever a redirect points.
		http: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
}

type wireQuestion struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type wireAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type wireResponse struct {
	Success *bool `json:"success"`
	Result  struct {
		Model   string                `json:"model"`
		Answers map[string]wireAnswer `json:"answers"`
		Usage   struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	} `json:"result"`
}

// Choose asks every question at once. job names the caller in the log. Any
// failure is an error and the caller goes on as if there were no model.
func (c *Client) Choose(ctx context.Context, job, model string, state any, questions map[string]Question) (map[string]Answer, error) {
	if c == nil {
		return nil, ErrOff
	}
	if model == "" {
		model = ModelFast
	}
	wire := map[string]wireQuestion{}
	for id, q := range questions {
		if len(q.Options) < 2 || len(q.Options) > maxOptions {
			return nil, fmt.Errorf("triage: question %s needs 2-%d options", id, maxOptions)
		}
		wire[id] = wireQuestion{Type: "choice", Instructions: q.Instructions, Criteria: q.Options}
	}
	body, err := json.Marshal(map[string]any{"model": model, "state": state, "questions": wire})
	if err != nil {
		return nil, err
	}
	started := time.Now()
	answers, inputTokens, err := c.call(ctx, model, body, questions)
	row := []any{"job", job, "provider", "workersai", "model", model, "ok", err == nil,
		"latency_ms", time.Since(started).Milliseconds(), "state_hash", stateHash(state)}
	if err != nil {
		row = append(row, "reason", err.Error())
	} else {
		for id, a := range answers {
			row = append(row, id+"_choice", a.Choice, id+"_p", math.Round(a.PTop*100)/100, id+"_margin", math.Round(a.Margin*100)/100)
		}
		row = append(row, "input_tokens", inputTokens)
	}
	// The issue text, the options and the token never reach the log.
	c.logger.InfoContext(ctx, "triage decision", row...)
	return answers, err
}

func (c *Client) call(ctx context.Context, model string, body []byte, questions map[string]Question) (map[string]Answer, int, error) {
	url := fmt.Sprintf("%s/accounts/%s/ai/run/@cf/cloudflare/%s", c.baseURL, c.accountID, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, 0, errors.New("timeout")
		}
		return nil, 0, errors.New("unreachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Error bodies are never read: they can echo the request.
		return nil, 0, fmt.Errorf("http %d", resp.StatusCode)
	}
	var w wireResponse
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil || (w.Success != nil && !*w.Success) {
		return nil, 0, errors.New("malformed")
	}
	out := map[string]Answer{}
	for id, q := range questions {
		a, ok := validate(w.Result.Answers[id], q)
		if !ok {
			return nil, 0, errors.New("malformed")
		}
		out[id] = a
	}
	return out, w.Result.Usage.InputTokens, nil
}

// validate: the choice was offered, probabilities are offered keys in [0,1]
// summing to ~1, and the choice is the most likely one.
func validate(raw wireAnswer, q Question) (Answer, bool) {
	if raw.Type != "" && raw.Type != "choice" {
		return Answer{}, false
	}
	if _, ok := q.Options[raw.Choice]; !ok || len(raw.Probabilities) == 0 {
		return Answer{}, false
	}
	sum := 0.0
	for k, p := range raw.Probabilities {
		if _, ok := q.Options[k]; !ok || math.IsNaN(p) || p < 0 || p > 1 {
			return Answer{}, false
		}
		sum += p
	}
	if math.Abs(sum-1) > sumTolerance {
		return Answer{}, false
	}
	top := raw.Probabilities[raw.Choice]
	runner := 0.0
	for k, p := range raw.Probabilities {
		if k != raw.Choice && p > runner {
			runner = p
		}
	}
	if runner > top+1e-9 {
		return Answer{}, false
	}
	return Answer{Choice: raw.Choice, PTop: top, Margin: top - runner, Probabilities: raw.Probabilities}, true
}

func stateHash(state any) string {
	b, _ := json.Marshal(state)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:16]
}
