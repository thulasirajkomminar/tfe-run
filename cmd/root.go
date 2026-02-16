// Package cmd defines the root command and main execution logic for the tfe-run CLI application.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	tfeclient "github.com/thulasirajkomminar/tfe-run/internal/tfe"
)

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
	rootCmd.Flags().String("workspace", "", "Comma-separated workspace names")
	rootCmd.Flags().String("planonly", "", "Plan only run: true/false (empty = workspace default)")

	return rootCmd.Execute()
}

type runOptions struct {
	tags      string
	workspace string
	org       string
	planOnly  string
}

func parseFlags(cmd *cobra.Command) (runOptions, error) {
	tags, _ := cmd.Flags().GetString("tags")
	workspace, _ := cmd.Flags().GetString("workspace")
	org, _ := cmd.Flags().GetString("org")
	planOnly, _ := cmd.Flags().GetString("planonly")

	opts := runOptions{
		tags:      tags,
		workspace: workspace,
		org:       org,
		planOnly:  planOnly,
	}

	if opts.tags == "" && opts.workspace == "" {
		return opts, errors.New("either --tags or --workspace must be specified")
	}

	if opts.tags != "" && opts.workspace != "" {
		return opts, errors.New("--tags and --workspace are mutually exclusive, specify only one")
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
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
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

	isPlanOnly := parsePlanOnly(opts.planOnly)

	if opts.tags != "" {
		return tfeclient.RunByTags(client, resolvedOrg, splitAndTrim(opts.tags), isPlanOnly)
	}

	return tfeclient.RunByNames(client, resolvedOrg, splitAndTrim(opts.workspace), isPlanOnly)
}

func resolveOrg(org string) string {
	if org != "" {
		return org
	}

	if envOrg := os.Getenv("TFE_ORG"); envOrg != "" {
		return envOrg
	}

	return ""
}
