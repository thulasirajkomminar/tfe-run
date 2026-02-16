package tfe

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe"
	log "github.com/sirupsen/logrus"
)

// ListByTags retrieves all workspaces matching the given tags.
func ListByTags(client *tfe.Client, org string, tags []string) ([]*tfe.Workspace, error) {
	ctx := context.Background()

	var allWorkspaces []*tfe.Workspace

	tagFilter := strings.Join(tags, ",")
	log.Infof("Searching for workspaces with tags: %s", tagFilter)

	opts := &tfe.WorkspaceListOptions{
		Tags: tagFilter,
		ListOptions: tfe.ListOptions{
			PageSize: 100,
		},
	}

	for {
		workspaces, err := client.Workspaces.List(ctx, org, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list workspaces: %w", err)
		}

		allWorkspaces = append(allWorkspaces, workspaces.Items...)

		if workspaces.CurrentPage >= workspaces.TotalPages {
			break
		}

		opts.PageNumber = workspaces.NextPage
	}

	if len(allWorkspaces) == 0 {
		return nil, fmt.Errorf("no workspaces found matching tags: %s", tagFilter)
	}

	log.Infof("Found %d workspace(s) matching tags", len(allWorkspaces))

	return allWorkspaces, nil
}

// ListByNames retrieves workspaces by their exact names.
func ListByNames(client *tfe.Client, org string, names []string) ([]*tfe.Workspace, error) {
	ctx := context.Background()

	var workspaces []*tfe.Workspace

	log.Infof("Looking up workspaces: %s", strings.Join(names, ", "))

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		ws, err := client.Workspaces.Read(ctx, org, name)
		if err != nil {
			log.Warnf("Workspace %q not found or inaccessible: %v", name, err)
			continue
		}

		workspaces = append(workspaces, ws)
	}

	if len(workspaces) == 0 {
		return nil, errors.New("no accessible workspaces found from the provided names")
	}

	log.Infof("Found %d workspace(s) by name", len(workspaces))

	return workspaces, nil
}
