package tfe

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-tfe"
	log "github.com/sirupsen/logrus"
)

// RunByTags triggers runs on all workspaces matching the given tags.
func RunByTags(client *tfe.Client, org string, tags []string, planOnly *bool) error {
	workspaces, err := ListByTags(client, org, tags)
	if err != nil {
		return err
	}

	return triggerRuns(client, workspaces, planOnly)
}

// RunByNames triggers runs on workspaces matching the given names.
func RunByNames(client *tfe.Client, org string, names []string, planOnly *bool) error {
	workspaces, err := ListByNames(client, org, names)
	if err != nil {
		return err
	}

	return triggerRuns(client, workspaces, planOnly)
}

func triggerRuns(client *tfe.Client, workspaces []*tfe.Workspace, planOnly *bool) error {
	ctx := context.Background()

	var errCount int

	for _, ws := range workspaces {
		msg := fmt.Sprintf("Triggered by tfe-run CLI for workspace %s", ws.Name)
		runOpts := tfe.RunCreateOptions{
			Workspace: ws,
			Message:   &msg,
		}

		if planOnly != nil {
			runOpts.PlanOnly = planOnly
		}

		planOnlyLabel := "workspace default"
		if planOnly != nil {
			planOnlyLabel = fmt.Sprintf("%t", *planOnly)
		}

		log.WithFields(log.Fields{
			"workspace": ws.Name,
			"plan_only": planOnlyLabel,
		}).Info("Triggering run")

		_, err := client.Runs.Create(ctx, runOpts)
		if err != nil {
			log.WithFields(log.Fields{
				"workspace": ws.Name,
			}).Errorf("Failed to trigger run: %v", err)

			errCount++

			continue
		}

		log.WithField("workspace", ws.Name).Info("Run triggered successfully")
	}

	if errCount > 0 {
		return fmt.Errorf("%d out of %d runs failed", errCount, len(workspaces))
	}

	log.Infof("Successfully triggered %d run(s)", len(workspaces))

	return nil
}
