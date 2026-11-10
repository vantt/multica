package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

// vantt decisions: the CLI for /api/issues/{id}/decisions, so an agent (which
// talks to Multica only through this CLI) can ask a person to choose.
//
//	multica issue ask <issue> --question "…" --option "A" --option "B|why B" --recommended B [--due YYYY-MM-DD]
//	multica issue decision list <issue>
//	multica issue decision list --open
//	multica issue decision cancel <issue> <decision-id>
//
// Answering stays with people (the server refuses agents), so there is no
// answer command.

var issueAskCmd = &cobra.Command{
	Use:   "ask <issue-id>",
	Short: "Ask a person to choose between options on an issue (a decision)",
	Long: "Opens a decision on the issue: a question and 2 to 8 options, at most one recommended.\n" +
		"The people following the issue (else the workspace owners and admins) get it in their\n" +
		"inbox as \"Decision needed\". Their answer is posted on the issue as a comment.\n\n" +
		"An option is \"Label\" or \"Label|description\".",
	Args: exactArgs(1),
	RunE: runIssueAsk,
}

var issueDecisionCmd = &cobra.Command{
	Use:   "decision",
	Short: "List or cancel the decisions asked on issues",
}

var issueDecisionListCmd = &cobra.Command{
	Use:   "list [issue-id]",
	Short: "List an issue's decisions, or every open one in the workspace with --open",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runIssueDecisionList,
}

var issueDecisionCancelCmd = &cobra.Command{
	Use:   "cancel <issue-id> <decision-id>",
	Short: "Cancel an open decision",
	Args:  exactArgs(2),
	RunE:  runIssueDecisionCancel,
}

func init() {
	issueAskCmd.Flags().String("question", "", "The question, one sentence (required)")
	issueAskCmd.Flags().StringArray("option", nil, "An option, \"Label\" or \"Label|description\"; repeat 2 to 8 times")
	issueAskCmd.Flags().String("recommended", "", "Label of the option you recommend")
	issueAskCmd.Flags().String("due", "", "Due date, YYYY-MM-DD")
	issueAskCmd.Flags().String("output", "table", "Output format: table or json")

	issueDecisionListCmd.Flags().Bool("open", false, "Every open decision in the workspace")
	issueDecisionListCmd.Flags().String("output", "table", "Output format: table or json")
	issueDecisionListCmd.Flags().Bool("full-id", false, "Show full UUIDs in table output")
	issueDecisionCancelCmd.Flags().String("output", "table", "Output format: table or json")

	issueDecisionCmd.AddCommand(issueDecisionListCmd)
	issueDecisionCmd.AddCommand(issueDecisionCancelCmd)
	issueCmd.AddCommand(issueAskCmd)
	issueCmd.AddCommand(issueDecisionCmd)

	// main.go's init applies the help templates before this file's init
	// runs (files init in name order), so set them here.
	issueDecisionCmd.SetHelpTemplate(subHelpTemplate)
	issueDecisionCmd.SetUsageTemplate(subHelpTemplate)
	for _, c := range []*cobra.Command{issueAskCmd, issueDecisionListCmd, issueDecisionCancelCmd} {
		c.SetHelpTemplate(leafHelpTemplate)
		c.SetUsageTemplate(leafHelpTemplate)
	}
}

// decisionBody builds the create request and checks what the server would
// refuse, so a bad call fails before it is sent.
func decisionBody(question string, options []string, recommended, due string) (map[string]any, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, errors.New("--question is required")
	}
	if len(options) < 2 || len(options) > 8 {
		return nil, fmt.Errorf("give 2 to 8 --option, got %d", len(options))
	}
	recommended = strings.TrimSpace(recommended)
	found := recommended == ""
	opts := make([]map[string]any, 0, len(options))
	for _, raw := range options {
		label, desc, _ := strings.Cut(raw, "|")
		label, desc = strings.TrimSpace(label), strings.TrimSpace(desc)
		if label == "" {
			return nil, fmt.Errorf("option %q has no label", raw)
		}
		o := map[string]any{"label": label}
		if desc != "" {
			o["description"] = desc
		}
		if recommended != "" && strings.EqualFold(label, recommended) {
			o["recommended"] = true
			found = true
		}
		opts = append(opts, o)
	}
	if !found {
		return nil, fmt.Errorf("--recommended %q is not one of the options", recommended)
	}
	body := map[string]any{"question": question, "options": opts}
	if due = strings.TrimSpace(due); due != "" {
		if _, err := time.Parse("2006-01-02", due); err != nil {
			return nil, errors.New("--due must be YYYY-MM-DD")
		}
		body["due_date"] = due
	}
	return body, nil
}

func runIssueAsk(cmd *cobra.Command, args []string) error {
	question, _ := cmd.Flags().GetString("question")
	options, _ := cmd.Flags().GetStringArray("option")
	recommended, _ := cmd.Flags().GetString("recommended")
	due, _ := cmd.Flags().GetString("due")
	body, err := decisionBody(question, options, recommended, due)
	if err != nil {
		return err
	}
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	issueRef, err := resolveIssueRef(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve issue: %w", err)
	}
	var result map[string]any
	if err := client.PostJSON(ctx, "/api/issues/"+issueRef.ID+"/decisions", body, &result); err != nil {
		return fmt.Errorf("ask: %w", err)
	}
	if output, _ := cmd.Flags().GetString("output"); output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}
	fmt.Fprintf(os.Stdout, "Decision %s asked on %s. The answer arrives as a comment on the issue.\n", strVal(result, "id"), issueRef.Display)
	return nil
}

func runIssueDecisionList(cmd *cobra.Command, args []string) error {
	open, _ := cmd.Flags().GetBool("open")
	if open == (len(args) == 1) {
		return errors.New("give an issue, or --open for every open decision")
	}
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	path := "/api/issues/decisions/open"
	if !open {
		issueRef, err := resolveIssueRef(ctx, client, args[0])
		if err != nil {
			return fmt.Errorf("resolve issue: %w", err)
		}
		path = "/api/issues/" + issueRef.ID + "/decisions"
	}
	var result map[string]any
	if err := client.GetJSON(ctx, path, &result); err != nil {
		return fmt.Errorf("list decisions: %w", err)
	}
	decisions, _ := result["decisions"].([]any)
	if output, _ := cmd.Flags().GetString("output"); output == "json" {
		return cli.PrintJSON(os.Stdout, decisions)
	}
	fullID, _ := cmd.Flags().GetBool("full-id")
	printDecisionTable(decisions, fullID)
	return nil
}

func runIssueDecisionCancel(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := cli.APIContext(context.Background())
	defer cancel()
	issueRef, err := resolveIssueRef(ctx, client, args[0])
	if err != nil {
		return fmt.Errorf("resolve issue: %w", err)
	}
	var result map[string]any
	if err := client.PostJSON(ctx, "/api/issues/"+issueRef.ID+"/decisions/"+args[1]+"/cancel", map[string]any{}, &result); err != nil {
		return fmt.Errorf("cancel decision: %w", err)
	}
	if output, _ := cmd.Flags().GetString("output"); output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}
	fmt.Fprintln(os.Stdout, "Decision cancelled.")
	return nil
}

// decisionAnswer is the chosen option's label, or "" while open.
func decisionAnswer(d map[string]any) string {
	id := strVal(d, "answer_option_id")
	if id == "" {
		return ""
	}
	opts, _ := d["options"].([]any)
	for _, raw := range opts {
		if o, ok := raw.(map[string]any); ok && strVal(o, "id") == id {
			return strVal(o, "label")
		}
	}
	return id
}

func printDecisionTable(decisions []any, fullID bool) {
	headers := []string{"ID", "ISSUE", "STATUS", "QUESTION", "OPTIONS", "ANSWER"}
	rows := make([][]string, 0, len(decisions))
	for _, raw := range decisions {
		d, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		var labels []string
		opts, _ := d["options"].([]any)
		for _, o := range opts {
			if m, ok := o.(map[string]any); ok {
				l := strVal(m, "label")
				if rec, _ := m["recommended"].(bool); rec {
					l += " ★"
				}
				labels = append(labels, l)
			}
		}
		rows = append(rows, []string{
			displayID(strVal(d, "id"), fullID),
			displayID(strVal(d, "issue_id"), fullID),
			strVal(d, "status"),
			strVal(d, "question"),
			strings.Join(labels, " / "),
			decisionAnswer(d),
		})
	}
	cli.PrintTable(os.Stdout, headers, rows)
}
