package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

type Workspace struct {
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path"`
}

func (c *Client) ListWorkspaces(ctx context.Context, accountID, containerID string) ([]Workspace, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s", accountID, containerID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListWorkspacesResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toWorkspaces(resp.Workspace), nil
}

func toWorkspaces(workspaces []*tagmanager.Workspace) []Workspace {
	result := make([]Workspace, 0, len(workspaces))
	for _, w := range workspaces {
		result = append(result, Workspace{
			WorkspaceID: w.WorkspaceId,
			Name:        w.Name,
			Description: w.Description,
			Path:        w.Path,
		})
	}
	return result
}

type SyncStatus struct {
	HasConflicts        bool     `json:"hasConflicts"`
	ConflictCount       int      `json:"conflictCount"`
	ConflictingEntities []string `json:"conflictingEntities,omitempty"`
}

func (c *Client) SyncWorkspace(ctx context.Context, accountID, containerID, workspaceID string) (*SyncStatus, error) {
	path := BuildWorkspacePath(accountID, containerID, workspaceID)

	resp, err := c.Service.Accounts.Containers.Workspaces.Sync(path).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	status := &SyncStatus{
		HasConflicts:  len(resp.MergeConflict) > 0,
		ConflictCount: len(resp.MergeConflict),
	}

	for _, mc := range resp.MergeConflict {
		entityName := ""
		if mc.EntityInWorkspace != nil {
			if mc.EntityInWorkspace.Tag != nil {
				entityName = "tag:" + mc.EntityInWorkspace.Tag.Name
			} else if mc.EntityInWorkspace.Trigger != nil {
				entityName = "trigger:" + mc.EntityInWorkspace.Trigger.Name
			} else if mc.EntityInWorkspace.Variable != nil {
				entityName = "variable:" + mc.EntityInWorkspace.Variable.Name
			}
		} else if mc.EntityInBaseVersion != nil {
			if mc.EntityInBaseVersion.Tag != nil {
				entityName = "tag:" + mc.EntityInBaseVersion.Tag.Name
			} else if mc.EntityInBaseVersion.Trigger != nil {
				entityName = "trigger:" + mc.EntityInBaseVersion.Trigger.Name
			} else if mc.EntityInBaseVersion.Variable != nil {
				entityName = "variable:" + mc.EntityInBaseVersion.Variable.Name
			}
		}
		if entityName != "" {
			status.ConflictingEntities = append(status.ConflictingEntities, entityName)
		}
	}

	return status, nil
}

func (c *Client) DeleteWorkspace(ctx context.Context, accountID, containerID, workspaceID string) error {
	path := BuildWorkspacePath(accountID, containerID, workspaceID)
	err := c.Service.Accounts.Containers.Workspaces.Delete(path).Context(ctx).Do()
	if err != nil {
		return mapGoogleError(err)
	}
	return nil
}
