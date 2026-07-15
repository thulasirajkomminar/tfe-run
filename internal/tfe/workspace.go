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

const (
	// MatchAny selects workspaces carrying at least one of the tags (union).
	MatchAny TagMatchMode = "any"
	// MatchAll selects workspaces carrying every tag (intersection).
	MatchAll TagMatchMode = "all"
)

// Workspace is a lightweight view of a TFE workspace, decoupling callers from
// the verbose kiota-generated models.
type Workspace struct {
	ID   string
	Name string
	Tags []string
}

// TagMatchMode controls how multiple tags are combined when filtering workspaces.
type TagMatchMode string

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

	if attrs := ws.GetAttributes(); attrs != nil {
		if attrs.GetName() != nil {
			w.Name = *attrs.GetName()
		}

		w.Tags = attrs.GetTagNames()
	}

	return w
}

// ListByTags retrieves all workspaces matching the given tags according to
// the given match mode. Tags are matched client-side against each
// workspace's tag names: the tag filter query parameters exposed by the
// generated API client (filter[tagged], filter[tag-union]) are not
// implemented by TFE/HCP Terraform and are silently ignored, which would
// select every workspace in the organization.
func ListByTags(client *tfe.Client, org string, tags []string, mode TagMatchMode) ([]Workspace, error) {
	ctx := context.Background()

	log.Infof("Searching for workspaces matching %s of the tags: %s", mode, strings.Join(tags, ","))

	config := &abstractions.RequestConfiguration[organizations.ItemWorkspacesRequestBuilderGetQueryParameters]{
		QueryParameters: &organizations.ItemWorkspacesRequestBuilderGetQueryParameters{
			Pagesize: new(int32(100)),
		},
	}

	var workspaces []Workspace

	for {
		resp, err := workspacesBuilder(client, org).Get(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to list workspaces: %w", err)
		}

		for _, ws := range resp.GetData() {
			if w := newWorkspace(ws); matchesTags(w.Tags, tags, mode) {
				workspaces = append(workspaces, w)
			}
		}

		next := nextPage(resp)
		if next == nil {
			break
		}

		config.QueryParameters.Pagenumber = next
	}

	if len(workspaces) == 0 {
		return nil, fmt.Errorf("no workspaces found matching %s of the tags: %s", mode, strings.Join(tags, ","))
	}

	log.Infof("Found %d workspace(s) matching tags", len(workspaces))

	return workspaces, nil
}

// matchesTags reports whether a workspace's tags satisfy the wanted tags
// under the given match mode. Comparison is case-insensitive since TFE
// stores tag names in lowercase.
func matchesTags(have, want []string, mode TagMatchMode) bool {
	haveSet := make(map[string]struct{}, len(have))
	for _, t := range have {
		haveSet[strings.ToLower(t)] = struct{}{}
	}

	matched := 0

	for _, t := range want {
		if _, ok := haveSet[strings.ToLower(t)]; ok {
			matched++
		}
	}

	if mode == MatchAll {
		return matched == len(want)
	}

	return matched > 0
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
