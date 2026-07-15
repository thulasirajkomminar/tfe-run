// Package cmd defines the root command and main execution logic for the tfe-run CLI application.
package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/go-tfe/v2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	tfeclient "github.com/thulasirajkomminar/tfe-run/internal/tfe"
)

type runOptions struct {
	tags      string
	tagMatch  string
	workspace string
	org       string
	planOnly  string
	dryRun    bool
}

// Execute is the main entry point for the CLI application.
func Execute() error {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: false,
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05",
	})

	rootCmd := &cobra.Command{
		Use:   "tfe-run",
		Short: "CLI tool to trigger Terraform runs on TFE/HCP Terraform workspaces",
		Long: `tfe-run triggers Terraform runs on multiple workspaces in Terraform Enterprise
or HCP Terraform. Workspaces can be selected by tags or by name.

Token resolution order:
  1. TFE_TOKEN environment variable
  2. ~/.terraformrc (credentials block)
  3. ~/.terraform.d/credentials.tfrc.json

Organization resolution order:
  1. --org flag
  2. TFE_ORG environment variable`,
		RunE: runCmd,
	}

	rootCmd.Flags().String("org", "", "TFE/HCP Terraform organization name (or set TFE_ORG env var)")
	rootCmd.Flags().String("tags", "", "Comma-separated workspace tags to filter by")
	rootCmd.Flags().String("tagmatch", "all", "Tag matching mode: all (workspace has every tag) or any (workspace has at least one tag)")
	rootCmd.Flags().String("workspace", "", "Comma-separated workspace names")
	rootCmd.Flags().String("planonly", "", "Plan only run: true/false (empty = workspace default)")
	rootCmd.Flags().Bool("dry-run", false, "List the workspaces that would be run, without triggering anything")

	return rootCmd.Execute()
}

func parseFlags(cmd *cobra.Command) (runOptions, error) {
	tags, _ := cmd.Flags().GetString("tags")
	tagMatch, _ := cmd.Flags().GetString("tagmatch")
	workspace, _ := cmd.Flags().GetString("workspace")
	org, _ := cmd.Flags().GetString("org")
	planOnly, _ := cmd.Flags().GetString("planonly")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	opts := runOptions{
		tags:      tags,
		tagMatch:  strings.ToLower(tagMatch),
		workspace: workspace,
		org:       org,
		planOnly:  planOnly,
		dryRun:    dryRun,
	}

	if opts.tags == "" && opts.workspace == "" {
		return opts, errors.New("either --tags or --workspace must be specified")
	}

	if opts.tags != "" && opts.workspace != "" {
		return opts, errors.New("--tags and --workspace are mutually exclusive, specify only one")
	}

	if opts.tagMatch != "any" && opts.tagMatch != "all" {
		return opts, fmt.Errorf("invalid --tagmatch value %q: must be \"any\" or \"all\"", tagMatch)
	}

	return opts, nil
}

func parsePlanOnly(val string) *bool {
	if val == "" {
		return nil
	}

	b := strings.EqualFold(val, "true")

	return &b
}

func splitAndTrim(s string) []string {
	var parts []string

	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}

	return parts
}

func runCmd(cmd *cobra.Command, _ []string) error {
	opts, err := parseFlags(cmd)
	if err != nil {
		return err
	}

	resolvedOrg := resolveOrg(opts.org)
	if resolvedOrg == "" {
		return errors.New("organization is required: set --org flag or TFE_ORG environment variable")
	}

	client, err := tfeclient.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create TFE client: %w", err)
	}

	if opts.dryRun {
		log.Info("Dry run: no runs will be triggered")
	}

	workspaces, err := resolveWorkspaces(client, resolvedOrg, &opts)
	if err != nil {
		return err
	}

	if opts.dryRun {
		tfeclient.DryRun(workspaces)

		return nil
	}

	return tfeclient.TriggerRuns(client, workspaces, parsePlanOnly(opts.planOnly))
}

// resolveWorkspaces maps the CLI selection flags to the matching workspaces.
func resolveWorkspaces(client *tfe.Client, org string, opts *runOptions) ([]tfeclient.Workspace, error) {
	if opts.tags != "" {
		return tfeclient.ListByTags(client, org, splitAndTrim(opts.tags), tfeclient.TagMatchMode(opts.tagMatch))
	}

	return tfeclient.ListByNames(client, org, splitAndTrim(opts.workspace))
}

func resolveOrg(org string) string {
	return cmp.Or(org, os.Getenv("TFE_ORG"))
}
