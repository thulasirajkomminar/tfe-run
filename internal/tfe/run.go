package tfe

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	log "github.com/sirupsen/logrus"
)

// DryRun logs the runs that would be triggered on the given workspaces,
// without triggering anything.
func DryRun(workspaces []Workspace) {
	for _, ws := range workspaces {
		log.WithFields(log.Fields{
			"workspace": ws.Name,
			"tags":      strings.Join(ws.Tags, ", "),
		}).Info("Would trigger run")
	}

	log.Infof("Dry run complete: %d workspace(s) would be run", len(workspaces))
}

// TriggerRuns triggers a run on each of the given workspaces.
func TriggerRuns(client *tfe.Client, workspaces []Workspace, planOnly *bool) error {
	ctx := context.Background()

	var errCount int

	for _, ws := range workspaces {
		msg := fmt.Sprintf("Triggered by tfe-run CLI for workspace %s", ws.Name)

		planOnlyLabel := "workspace default"
		if planOnly != nil {
			planOnlyLabel = fmt.Sprintf("%t", *planOnly)
		}

		log.WithFields(log.Fields{
			"workspace": ws.Name,
			"plan_only": planOnlyLabel,
		}).Info("Triggering run")

		_, err := client.API.Runs().Post(ctx, newRunEnvelope(ws.ID, msg, planOnly), nil)
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

// newRunEnvelope builds the request body for creating a run targeting the given workspace.
func newRunEnvelope(workspaceID, msg string, planOnly *bool) models.RunsEnvelopeable {
	attrs := models.NewRuns_attributes()
	attrs.SetMessage(&msg)

	if planOnly != nil {
		attrs.SetPlanOnly(planOnly)
	}

	wsData := models.NewWorkspacesId_data()
	wsData.SetId(&workspaceID)
	wsData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESID_DATA_TYPE))

	wsRel := models.NewWorkspacesId()
	wsRel.SetData(wsData)

	rels := models.NewRuns_relationships()
	rels.SetWorkspace(wsRel)

	run := models.NewRuns()
	run.SetTypeEscaped(new(models.RUNS_RUNS_TYPE))
	run.SetAttributes(attrs)
	run.SetRelationships(rels)

	envelope := models.NewRunsEnvelope()
	envelope.SetData(run)

	return envelope
}
