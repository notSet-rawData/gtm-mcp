package gtm

import (
	"context"
	"fmt"

	tagmanager "google.golang.org/api/tagmanager/v2"
)

// Folder is a simplified representation of a GTM folder.
type Folder struct {
	FolderID string `json:"folderId"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Notes    string `json:"notes,omitempty"`
}

// FolderEntities contains the entities within a folder.
type FolderEntities struct {
	Tags      []string `json:"tags,omitempty"`
	Triggers  []string `json:"triggers,omitempty"`
	Variables []string `json:"variables,omitempty"`
}

// ListFolders returns all folders in a workspace.
func (c *Client) ListFolders(ctx context.Context, accountID, containerID, workspaceID string) ([]Folder, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.ListFoldersResponse, error) {
		return c.Service.Accounts.Containers.Workspaces.Folders.List(parent).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return toFolders(resp.Folder), nil
}

// GetFolderEntities returns the entities (tags, triggers, variables) in a folder.
func (c *Client) GetFolderEntities(ctx context.Context, accountID, containerID, workspaceID, folderID string) (*FolderEntities, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/folders/%s",
		accountID, containerID, workspaceID, folderID)

	resp, err := retryWithBackoff(ctx, 3, func() (*tagmanager.FolderEntities, error) {
		return c.Service.Accounts.Containers.Workspaces.Folders.Entities(path).Context(ctx).Do()
	})
	if err != nil {
		return nil, mapGoogleError(err)
	}

	entities := &FolderEntities{}

	for _, t := range resp.Tag {
		entities.Tags = append(entities.Tags, t.Name)
	}
	for _, t := range resp.Trigger {
		entities.Triggers = append(entities.Triggers, t.Name)
	}
	for _, v := range resp.Variable {
		entities.Variables = append(entities.Variables, v.Name)
	}

	return entities, nil
}

// CreateFolder creates a new folder in a workspace.
func (c *Client) CreateFolder(ctx context.Context, accountID, containerID, workspaceID, name, notes string) (*Folder, error) {
	parent := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s", accountID, containerID, workspaceID)

	folder := &tagmanager.Folder{
		Name:  name,
		Notes: notes,
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Folders.Create(parent, folder).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &Folder{
		FolderID: result.FolderId,
		Name:     result.Name,
		Path:     result.Path,
		Notes:    result.Notes,
	}, nil
}

// UpdateFolder updates a folder's name and/or notes.
func (c *Client) UpdateFolder(ctx context.Context, accountID, containerID, workspaceID, folderID, name, notes string) (*Folder, error) {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/folders/%s",
		accountID, containerID, workspaceID, folderID)

	folder := &tagmanager.Folder{}
	if name != "" {
		folder.Name = name
	}
	if notes != "" {
		folder.Notes = notes
	}

	result, err := c.Service.Accounts.Containers.Workspaces.Folders.Update(path, folder).Context(ctx).Do()
	if err != nil {
		return nil, mapGoogleError(err)
	}

	return &Folder{
		FolderID: result.FolderId,
		Name:     result.Name,
		Path:     result.Path,
		Notes:    result.Notes,
	}, nil
}

// MoveEntitiesToFolder moves tags, triggers, and/or variables into a folder.
func (c *Client) MoveEntitiesToFolder(ctx context.Context, accountID, containerID, workspaceID, folderID string, tagIDs, triggerIDs, variableIDs []string) error {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/folders/%s",
		accountID, containerID, workspaceID, folderID)

	call := c.Service.Accounts.Containers.Workspaces.Folders.MoveEntitiesToFolder(path, &tagmanager.Folder{})

	if len(tagIDs) > 0 {
		call = call.TagId(tagIDs...)
	}
	if len(triggerIDs) > 0 {
		call = call.TriggerId(triggerIDs...)
	}
	if len(variableIDs) > 0 {
		call = call.VariableId(variableIDs...)
	}

	return call.Context(ctx).Do()
}

// DeleteFolder deletes a folder from a workspace.
func (c *Client) DeleteFolder(ctx context.Context, accountID, containerID, workspaceID, folderID string) error {
	path := fmt.Sprintf("accounts/%s/containers/%s/workspaces/%s/folders/%s",
		accountID, containerID, workspaceID, folderID)

	return c.Service.Accounts.Containers.Workspaces.Folders.Delete(path).Context(ctx).Do()
}

func toFolders(folders []*tagmanager.Folder) []Folder {
	result := make([]Folder, 0, len(folders))
	for _, f := range folders {
		result = append(result, Folder{
			FolderID: f.FolderId,
			Name:     f.Name,
			Path:     f.Path,
			Notes:    f.Notes,
		})
	}
	return result
}
