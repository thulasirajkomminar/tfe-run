package tfe

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	log "github.com/sirupsen/logrus"
)

// Workspace is a lightweight view of a TFE workspace, decoupling callers from
// the verbose kiota-generated models.
type Workspace struct {
	ID   string
	Name string
}

// workspacesBuilder returns the request builder for an organization's workspaces.
func workspacesBuilder(client *tfe.Client, org string) *organizations.ItemWorkspacesRequestBuilder {
	return client.API.Organizations().ByOrganization_name(org).Workspaces()
}

// newWorkspace converts a kiota workspace model into a Workspace.
func newWorkspace(ws models.Workspacesable) Workspace {
	w := Workspace{}
	if ws == nil {
		return w
	}

	if id := ws.GetId(); id != nil {
		w.ID = *id
	}

	if attrs := ws.GetAttributes(); attrs != nil && attrs.GetName() != nil {
		w.Name = *attrs.GetName()
	}

	return w
}

// ListByTags retrieves all workspaces matching the given tags.
func ListByTags(client *tfe.Client, org string, tags []string) ([]Workspace, error) {
	ctx := context.Background()

	tagFilter := strings.Join(tags, ",")
	log.Infof("Searching for workspaces with tags: %s", tagFilter)

	config := &abstractions.RequestConfiguration[organizations.ItemWorkspacesRequestBuilderGetQueryParameters]{
		QueryParameters: &organizations.ItemWorkspacesRequestBuilderGetQueryParameters{
			Filtertagged: &tagFilter,
			Pagesize:     new(int32(100)),
		},
	}

	var workspaces []Workspace

	for {
		resp, err := workspacesBuilder(client, org).Get(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to list workspaces: %w", err)
		}

		for _, ws := range resp.GetData() {
			workspaces = append(workspaces, newWorkspace(ws))
		}

		next := nextPage(resp)
		if next == nil {
			break
		}

		config.QueryParameters.Pagenumber = next
	}

	if len(workspaces) == 0 {
		return nil, fmt.Errorf("no workspaces found matching tags: %s", tagFilter)
	}

	log.Infof("Found %d workspace(s) matching tags", len(workspaces))

	return workspaces, nil
}

// nextPage returns the next page number from a workspace list response, or nil
// when there are no further pages.
func nextPage(resp organizations.ItemWorkspacesGetResponseable) *int32 {
	meta := resp.GetMeta()
	if meta == nil {
		return nil
	}

	pagination := meta.GetPagination()
	if pagination == nil {
		return nil
	}

	return pagination.GetNextPage()
}

// ListByNames retrieves workspaces by their exact names.
func ListByNames(client *tfe.Client, org string, names []string) ([]Workspace, error) {
	ctx := context.Background()

	log.Infof("Looking up workspaces: %s", strings.Join(names, ", "))

	var workspaces []Workspace

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		resp, err := workspacesBuilder(client, org).ByWorkspace_name(name).Get(ctx, nil)
		if err != nil {
			log.Warnf("Workspace %q not found or inaccessible: %v", name, err)
			continue
		}

		if resp.GetData() == nil {
			log.Warnf("Workspace %q returned no data", name)
			continue
		}

		workspaces = append(workspaces, newWorkspace(resp.GetData()))
	}

	if len(workspaces) == 0 {
		return nil, errors.New("no accessible workspaces found from the provided names")
	}

	log.Infof("Found %d workspace(s) by name", len(workspaces))

	return workspaces, nil
}
