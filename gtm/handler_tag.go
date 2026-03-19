package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TagToolInput is the unified input for the tag tool.
type TagToolInput struct {
	Action    string `json:"action" jsonschema:"enum:list,get,create,update,delete,revert,description:Operation to perform on tags"`
	AccountID string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	// Fields for get/update/delete:
	TagID string `json:"tagId,omitempty" jsonschema:"description:Tag ID (required for get, update, delete)"`
	// Fields for create/update:
	Name               string   `json:"name,omitempty" jsonschema:"description:Tag name (required for create/update)"`
	Type               string   `json:"type,omitempty" jsonschema:"description:Tag type e.g. gaawe (GA4), html (Custom HTML) (required for create/update)"`
	FiringTriggerIDs   []string `json:"firingTriggerIds,omitempty" jsonschema:"description:Trigger IDs that fire this tag (required for create/update)"`
	BlockingTriggerIDs []string `json:"blockingTriggerIds,omitempty" jsonschema:"description:Trigger IDs that block this tag (optional)"`
	ParametersJSON     string   `json:"parametersJson,omitempty" jsonschema:"description:Tag parameters as JSON array (optional). Each parameter: {type, key, value}"`
	SetupTagJSON       string   `json:"setupTagJson,omitempty" jsonschema:"description:Setup tag sequencing as JSON array (optional). Each element: {tagName, stopOnSetupFailure}"`
	TeardownTagJSON    string   `json:"teardownTagJson,omitempty" jsonschema:"description:Teardown tag sequencing as JSON array (optional). Each element: {tagName, stopTeardownOnFailure}"`
	Notes              string   `json:"notes,omitempty" jsonschema:"description:Tag notes (optional)"`
	Paused             bool     `json:"paused,omitempty" jsonschema:"description:Whether tag is paused (optional)"`
	// Fields for delete:
	Confirm     bool   `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
}


func handleTagList(ctx context.Context, input TagToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "tag_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tags, err := wc.Client.ListTags(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	out := ListTagsOutput{Tags: tags}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleTagGet(ctx context.Context, input TagToolInput) (*mcp.CallToolResult, any, error) {
	if input.TagID == "" {
		return nil, nil, fmt.Errorf("tagId is required for get action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tag, err := wc.Client.GetTag(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetTagOutput{Tag: *tag}, nil
}

func handleTagCreate(ctx context.Context, input TagToolInput) (*mcp.CallToolResult, any, error) {
	if err := ValidateTagInput(input.Name, input.Type, input.FiringTriggerIDs); err != nil {
		return nil, nil, err
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var params []Parameter
	if input.ParametersJSON != "" {
		if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
			return nil, nil, err
		}
	}

	var setupTags []SetupTagInput
	if input.SetupTagJSON != "" {
		if err := json.Unmarshal([]byte(input.SetupTagJSON), &setupTags); err != nil {
			return nil, nil, fmt.Errorf("invalid setupTagJson: %w", err)
		}
	}

	var teardownTags []TeardownTagInput
	if input.TeardownTagJSON != "" {
		if err := json.Unmarshal([]byte(input.TeardownTagJSON), &teardownTags); err != nil {
			return nil, nil, fmt.Errorf("invalid teardownTagJson: %w", err)
		}
	}

	tagInput := &TagInput{
		Name:              input.Name,
		Type:              input.Type,
		FiringTriggerId:   input.FiringTriggerIDs,
		BlockingTriggerId: input.BlockingTriggerIDs,
		Parameter:         params,
		Notes:             input.Notes,
		Paused:            input.Paused,
		SetupTag:          setupTags,
		TeardownTag:       teardownTags,
	}

	tag, err := wc.Client.CreateTag(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, tagInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, CreateTagOutput{
		Success: true,
		Tag:     *tag,
		Message: "Tag created successfully",
	}, nil
}

func handleTagUpdate(ctx context.Context, input TagToolInput) (*mcp.CallToolResult, any, error) {
	if input.TagID == "" {
		return nil, nil, fmt.Errorf("tagId is required for update action")
	}
	if err := ValidateTagInput(input.Name, input.Type, input.FiringTriggerIDs); err != nil {
		return nil, nil, err
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildTagPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)

	var params []Parameter
	if input.ParametersJSON != "" {
		if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
			return nil, nil, err
		}
	}

	var setupTags []SetupTagInput
	var clearSetup bool
	if input.SetupTagJSON != "" {
		if err := json.Unmarshal([]byte(input.SetupTagJSON), &setupTags); err != nil {
			return nil, nil, fmt.Errorf("invalid setupTagJson: %w", err)
		}
		if len(setupTags) == 0 {
			clearSetup = true
		}
	}

	var teardownTags []TeardownTagInput
	var clearTeardown bool
	if input.TeardownTagJSON != "" {
		if err := json.Unmarshal([]byte(input.TeardownTagJSON), &teardownTags); err != nil {
			return nil, nil, fmt.Errorf("invalid teardownTagJson: %w", err)
		}
		if len(teardownTags) == 0 {
			clearTeardown = true
		}
	}

	tagInput := &TagInput{
		Name:              input.Name,
		Type:              input.Type,
		FiringTriggerId:   input.FiringTriggerIDs,
		BlockingTriggerId: input.BlockingTriggerIDs,
		Parameter:         params,
		Notes:             input.Notes,
		Paused:            input.Paused,
		SetupTag:          setupTags,
		TeardownTag:       teardownTags,
		ClearSetupTag:     clearSetup,
		ClearTeardownTag:  clearTeardown,
	}

	tag, err := wc.Client.UpdateTag(tCtx, path, tagInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, UpdateTagOutput{
		Success: true,
		Tag:     *tag,
		Message: "Tag updated successfully",
	}, nil
}

func handleTagDelete(ctx context.Context, input TagToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteTagOutput{
			Success: false,
			Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		}, nil
	}
	if input.TagID == "" {
		return nil, nil, fmt.Errorf("tagId is required for delete action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildTagPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)
	if err := wc.Client.DeleteTag(tCtx, path); err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DeleteTagOutput{
		Success: true,
		Message: fmt.Sprintf("Tag %s deleted successfully", input.TagID),
	}, nil
}

func handleTagRevert(ctx context.Context, input TagToolInput) (*mcp.CallToolResult, any, error) {
	if input.TagID == "" {
		return nil, nil, fmt.Errorf("tagId is required for revert action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildTagPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)
	call := wc.Client.Service.Accounts.Containers.Workspaces.Tags.Revert(path).Context(tCtx)
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
		Message: fmt.Sprintf("Tag %s reverted to latest published version", input.TagID),
		Entity:  resp.Tag,
	}, nil
}
