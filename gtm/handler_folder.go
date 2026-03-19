package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FolderToolInput is the unified input for the folder tool.
type FolderToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,update,delete,move,audit,revert,description:Operation to perform on folders"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	// Fields for get/update/delete/move:
	FolderID string `json:"folderId,omitempty" jsonschema:"description:Folder ID (required for get, update, delete, move)"`
	// Fields for create/update:
	Name  string `json:"name,omitempty" jsonschema:"description:Folder name (required for create/update)"`
	Notes string `json:"notes,omitempty" jsonschema:"description:Folder notes (optional)"`
	// Fields for move:
	TagIDs      []string `json:"tagIds,omitempty" jsonschema:"description:Tag IDs to move into the folder (for move action)"`
	TriggerIDs  []string `json:"triggerIds,omitempty" jsonschema:"description:Trigger IDs to move into the folder (for move action)"`
	VariableIDs []string `json:"variableIds,omitempty" jsonschema:"description:Variable IDs to move into the folder (for move action)"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
}


func handleFolderList(ctx context.Context, input FolderToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "folder_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	folders, err := wc.Client.ListFolders(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	out := ListFoldersOutput{Folders: folders}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleFolderGet(ctx context.Context, input FolderToolInput) (*mcp.CallToolResult, any, error) {
	if input.FolderID == "" {
		return nil, nil, fmt.Errorf("folderId is required for get action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	entities, err := wc.Client.GetFolderEntities(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.FolderID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetFolderEntitiesOutput{Entities: *entities}, nil
}

func handleFolderCreate(ctx context.Context, input FolderToolInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return nil, nil, fmt.Errorf("name is required for create action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	folder, err := wc.Client.CreateFolder(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.Name, input.Notes)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, CreateFolderOutput{
		Success: true,
		Folder:  *folder,
		Message: fmt.Sprintf("Folder '%s' created successfully", input.Name),
	}, nil
}

func handleFolderUpdate(ctx context.Context, input FolderToolInput) (*mcp.CallToolResult, any, error) {
	if input.FolderID == "" {
		return nil, nil, fmt.Errorf("folderId is required for update action")
	}
	if input.Name == "" {
		return nil, nil, fmt.Errorf("name is required for update action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	folder, err := wc.Client.UpdateFolder(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.FolderID, input.Name, input.Notes)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, UpdateFolderOutput{
		Success: true,
		Folder:  *folder,
		Message: fmt.Sprintf("Folder '%s' updated successfully", input.Name),
	}, nil
}

func handleFolderDelete(ctx context.Context, input FolderToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteFolderOutput{
			Success: false,
			Message: "Deletion requires confirm: true. WARNING: This will remove the folder (entities inside are NOT deleted, just unassigned).",
		}, nil
	}
	if input.FolderID == "" {
		return nil, nil, fmt.Errorf("folderId is required for delete action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := wc.Client.DeleteFolder(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.FolderID); err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DeleteFolderOutput{
		Success: true,
		Message: fmt.Sprintf("Folder %s deleted. Entities previously in this folder are now unassigned.", input.FolderID),
	}, nil
}

func handleFolderMove(ctx context.Context, input FolderToolInput) (*mcp.CallToolResult, any, error) {
	if input.FolderID == "" {
		return nil, nil, fmt.Errorf("folderId is required for move action")
	}
	if len(input.TagIDs) == 0 && len(input.TriggerIDs) == 0 && len(input.VariableIDs) == 0 {
		return nil, nil, fmt.Errorf("at least one of tagIds, triggerIds, or variableIds must be provided")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = wc.Client.MoveEntitiesToFolder(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID,
		input.FolderID, input.TagIDs, input.TriggerIDs, input.VariableIDs)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	total := len(input.TagIDs) + len(input.TriggerIDs) + len(input.VariableIDs)
	return nil, MoveToFolderOutput{
		Success: true,
		Message: fmt.Sprintf("Moved %d entities to folder %s (%d tags, %d triggers, %d variables)",
			total, input.FolderID, len(input.TagIDs), len(input.TriggerIDs), len(input.VariableIDs)),
	}, nil
}

func handleFolderAudit(ctx context.Context, input FolderToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// List folders
	folders, err := wc.Client.ListFolders(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list folders: %w", err)
	}

	// Build set of entity IDs in folders
	inFolder := make(map[string]bool)
	for _, f := range folders {
		entities, err := wc.Client.GetFolderEntities(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, f.FolderID)
		if err == nil {
			for _, id := range entities.Tags {
				inFolder["tag:"+id] = true
			}
			for _, id := range entities.Triggers {
				inFolder["trigger:"+id] = true
			}
			for _, id := range entities.Variables {
				inFolder["variable:"+id] = true
			}
		}
	}

	tags, err := wc.Client.ListTags(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tags: %w", err)
	}
	triggers, err := wc.Client.ListTriggers(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list triggers: %w", err)
	}
	variables, err := wc.Client.ListVariables(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list variables: %w", err)
	}

	var unorgTags, unorgTrigs, unorgVars []FolderAuditEntity

	for _, t := range tags {
		if !inFolder["tag:"+t.TagID] {
			unorgTags = append(unorgTags, FolderAuditEntity{ID: t.TagID, Name: t.Name, Type: t.Type})
		}
	}
	for _, tr := range triggers {
		if !inFolder["trigger:"+tr.TriggerID] {
			unorgTrigs = append(unorgTrigs, FolderAuditEntity{ID: tr.TriggerID, Name: tr.Name, Type: tr.Type})
		}
	}
	for _, v := range variables {
		if !inFolder["variable:"+v.VariableID] {
			unorgVars = append(unorgVars, FolderAuditEntity{ID: v.VariableID, Name: v.Name, Type: v.Type})
		}
	}

	totalEntities := len(tags) + len(triggers) + len(variables)
	totalUnorganized := len(unorgTags) + len(unorgTrigs) + len(unorgVars)
	pct := 0
	if totalEntities > 0 {
		pct = (totalUnorganized * 100) / totalEntities
	}

	summary := fmt.Sprintf(
		"%d folders, %d tags, %d triggers, %d variables. %d entities (%d%%) are NOT in any folder.",
		len(folders), len(tags), len(triggers), len(variables), totalUnorganized, pct,
	)

	return nil, AuditFolderStructureOutput{
		Folders:          folders,
		UnorganizedTags:  unorgTags,
		UnorganizedTrigs: unorgTrigs,
		UnorganizedVars:  unorgVars,
		Summary:          summary,
		NamingConvention: "Folders: group by vendor (GA4, Facebook, LinkedIn) or function (Analytics, Marketing, Consent, Utility, Ecommerce). Tags: [Vendor] - [Action] - [Detail]. Triggers: [Type] - [Event/Page]. Variables: [Type] - [Key].",
	}, nil
}

func handleFolderRevert(ctx context.Context, input FolderToolInput) (*mcp.CallToolResult, any, error) {
	if input.FolderID == "" {
		return nil, nil, fmt.Errorf("folderId is required for revert action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/folders/%s", wc.WorkspacePath(), input.FolderID)
	call := wc.Client.Service.Accounts.Containers.Workspaces.Folders.Revert(path).Context(tCtx)
	if input.Fingerprint != "" {
		call = call.Fingerprint(input.Fingerprint)
	}

	resp, err := call.Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, RevertOutput{
		Success: true,
		Message: fmt.Sprintf("Folder %s reverted to latest published version", input.FolderID),
		Entity:  resp.Folder,
	}, nil
}
