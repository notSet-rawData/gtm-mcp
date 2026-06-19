package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TagToolInput struct {
	Action                       string      `json:"action" jsonschema:"enum:list,get,create,update,delete,revert,append_list_entry,remove_list_entry,list_entries,description:Operation to perform on tags"`
	AccountID                    string      `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID                  string      `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID                  string      `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	TagID                        string      `json:"tagId,omitempty" jsonschema:"description:Tag ID (required for get, update, delete)"`
	Name                         string      `json:"name,omitempty" jsonschema:"description:Tag name (required for create/update)"`
	Type                         string      `json:"type,omitempty" jsonschema:"description:Tag type e.g. gaawe (GA4), html (Custom HTML) (required for create/update)"`
	FiringTriggerIDs             []string    `json:"firingTriggerIds,omitempty" jsonschema:"description:Trigger IDs that fire this tag (required for create/update)"`
	BlockingTriggerIDs           []string    `json:"blockingTriggerIds,omitempty" jsonschema:"description:Trigger IDs that block this tag (optional)"`
	Parameter                    []Parameter `json:"parameter,omitempty" jsonschema:"description:Tag parameters as array of objects. Each: {type, key, value}. Supports nested list/map."`
	ParametersJSON               string      `json:"parametersJson,omitempty" jsonschema:"description:DEPRECATED: Tag parameters as JSON string. Use parameter array instead."`
	SetupTagJSON                 string      `json:"setupTagJson,omitempty" jsonschema:"description:Setup tag sequencing as JSON array (optional). Each element: {tagName, stopOnSetupFailure}"`
	TeardownTagJSON              string      `json:"teardownTagJson,omitempty" jsonschema:"description:Teardown tag sequencing as JSON array (optional). Each element: {tagName, stopTeardownOnFailure}"`
	Notes                        string      `json:"notes,omitempty" jsonschema:"description:Tag notes (optional)"`
	Paused                       *bool       `json:"paused,omitempty" jsonschema:"description:Whether tag is paused (optional)"`
	PriorityJSON                 string      `json:"priorityJson,omitempty" jsonschema:"description:Tag firing priority as JSON Parameter object e.g. {type: integer, value: 100}. Higher values fire first."`
	ParentFolderID               string      `json:"parentFolderId,omitempty" jsonschema:"description:Parent folder ID for organizational purposes (optional)"`
	ScheduleStartMs              int64       `json:"scheduleStartMs,omitempty" jsonschema:"description:Start timestamp in milliseconds for scheduling tag activation (optional)"`
	ScheduleEndMs                int64       `json:"scheduleEndMs,omitempty" jsonschema:"description:End timestamp in milliseconds for scheduling tag deactivation (optional)"`
	MonitoringMetadataTagNameKey string      `json:"monitoringMetadataTagNameKey,omitempty" jsonschema:"description:Key for the tag name in monitoring metadata (optional)"`
	ConsentSettingsJSON          string      `json:"consentSettingsJson,omitempty" jsonschema:"description:Consent settings as JSON object e.g. {consentStatus: needed, consentType: {type: list, list: [{type: template, value: ad_storage}]}} (optional)"`
	Confirm                      bool        `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint                  string      `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
	// Fields for list entry operations (append_list_entry, remove_list_entry, list_entries):
	ListParameterKey string            `json:"listParameterKey,omitempty" jsonschema:"description:Key name of the list parameter to operate on (e.g. propertyConfigsList). Required for append_list_entry, remove_list_entry, list_entries."`
	Entry            *Parameter        `json:"entry,omitempty" jsonschema:"description:Entry to append. Must be type=map with a map array of key/value pairs. Required for append_list_entry."`
	DeduplicateBy    []string          `json:"deduplicateBy,omitempty" jsonschema:"description:List of map keys to use for deduplication. If an existing entry matches ALL these keys, the append is skipped."`
	MatchBy          map[string]string `json:"matchBy,omitempty" jsonschema:"description:Map of key=value pairs to match entries for removal. Required for remove_list_entry."`
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

	params, err := resolveParameters(input.Parameter, input.ParametersJSON)
	if err != nil {
		return nil, nil, err
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

	var priority *Parameter
	if input.PriorityJSON != "" {
		var p Parameter
		if err := json.Unmarshal([]byte(input.PriorityJSON), &p); err != nil {
			return nil, nil, fmt.Errorf("invalid priorityJson: %w", err)
		}
		priority = &p
	}

	var consentSettings *ConsentSettingInput
	if input.ConsentSettingsJSON != "" {
		var cs ConsentSettingInput
		if err := json.Unmarshal([]byte(input.ConsentSettingsJSON), &cs); err != nil {
			return nil, nil, fmt.Errorf("invalid consentSettingsJson: %w", err)
		}
		consentSettings = &cs
	}

	tagInput := &TagInput{
		Name:                         input.Name,
		Type:                         input.Type,
		FiringTriggerId:              input.FiringTriggerIDs,
		BlockingTriggerId:            input.BlockingTriggerIDs,
		Parameter:                    params,
		Notes:                        input.Notes,
		Paused:                       input.Paused,
		SetupTag:                     setupTags,
		TeardownTag:                  teardownTags,
		Priority:                     priority,
		ParentFolderID:               input.ParentFolderID,
		ScheduleStartMs:              input.ScheduleStartMs,
		ScheduleEndMs:                input.ScheduleEndMs,
		MonitoringMetadataTagNameKey: input.MonitoringMetadataTagNameKey,
		ConsentSettings:              consentSettings,
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

	params, err := resolveParameters(input.Parameter, input.ParametersJSON)
	if err != nil {
		return nil, nil, err
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

	var priority *Parameter
	if input.PriorityJSON != "" {
		var p Parameter
		if err := json.Unmarshal([]byte(input.PriorityJSON), &p); err != nil {
			return nil, nil, fmt.Errorf("invalid priorityJson: %w", err)
		}
		priority = &p
	}

	var consentSettings *ConsentSettingInput
	if input.ConsentSettingsJSON != "" {
		var cs ConsentSettingInput
		if err := json.Unmarshal([]byte(input.ConsentSettingsJSON), &cs); err != nil {
			return nil, nil, fmt.Errorf("invalid consentSettingsJson: %w", err)
		}
		consentSettings = &cs
	}

	tagInput := &TagInput{
		Name:                         input.Name,
		Type:                         input.Type,
		FiringTriggerId:              input.FiringTriggerIDs,
		BlockingTriggerId:            input.BlockingTriggerIDs,
		Parameter:                    params,
		Notes:                        input.Notes,
		Paused:                       input.Paused,
		SetupTag:                     setupTags,
		TeardownTag:                  teardownTags,
		ClearSetupTag:                clearSetup,
		ClearTeardownTag:             clearTeardown,
		Priority:                     priority,
		ParentFolderID:               input.ParentFolderID,
		ScheduleStartMs:              input.ScheduleStartMs,
		ScheduleEndMs:                input.ScheduleEndMs,
		MonitoringMetadataTagNameKey: input.MonitoringMetadataTagNameKey,
		ConsentSettings:              consentSettings,
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

// --- List entry operations ---

func handleTagAppendListEntry(ctx context.Context, input TagToolInput) (*mcp.CallToolResult, any, error) {
	if input.TagID == "" {
		return nil, nil, fmt.Errorf("tagId is required for append_list_entry action")
	}
	if input.ListParameterKey == "" {
		return nil, nil, fmt.Errorf("listParameterKey is required for append_list_entry action")
	}
	if input.Entry == nil {
		return nil, nil, fmt.Errorf("entry is required for append_list_entry action — must be a map-type parameter")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// GET current tag
	tag, err := wc.Client.GetTag(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tag %s: %w", input.TagID, err)
	}

	params := getTagParams(tag)
	if params == nil {
		return nil, nil, fmt.Errorf("tag %s (%s) has no parameters", input.TagID, tag.Name)
	}

	idx, existingEntries := findListParameter(params, input.ListParameterKey)
	if idx == -1 {
		return nil, nil, fmt.Errorf("tag %s (%s) does not have a list parameter with key %q", input.TagID, tag.Name, input.ListParameterKey)
	}

	previousSize := len(existingEntries)

	// Determine merge config
	var merge *MergeConfig
	// v1: always skip duplicates (merge support in v2)

	action, updatedEntries, _ := appendListEntry(existingEntries, *input.Entry, input.DeduplicateBy, merge)

	if action == "skipped" {
		return nil, AppendListEntryOutput{
			Success:          true,
			EntityType:       "tag",
			EntityID:         input.TagID,
			EntityName:       tag.Name,
			ListParameterKey: input.ListParameterKey,
			Action:           action,
			PreviousSize:     previousSize,
			CurrentSize:      previousSize,
			Entry:            flattenEntry(*input.Entry),
			Fingerprint:      tag.Fingerprint,
		}, nil
	}

	// Update the list in params
	params[idx].List = updatedEntries

	// UPDATE tag
	path := BuildTagPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)
	tagInput := &TagInput{
		Name:            tag.Name,
		Type:            tag.Type,
		Parameter:       params,
		FiringTriggerId: tag.FiringTriggerID,
	}

	updatedTag, err := wc.Client.UpdateTag(tCtx, path, tagInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update tag %s: %w", input.TagID, err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, AppendListEntryOutput{
		Success:          true,
		EntityType:       "tag",
		EntityID:         input.TagID,
		EntityName:       tag.Name,
		ListParameterKey: input.ListParameterKey,
		Action:           action,
		PreviousSize:     previousSize,
		CurrentSize:      len(updatedEntries),
		Entry:            flattenEntry(*input.Entry),
		Fingerprint:      updatedTag.Fingerprint,
	}, nil
}

func handleTagRemoveListEntry(ctx context.Context, input TagToolInput) (*mcp.CallToolResult, any, error) {
	if input.TagID == "" {
		return nil, nil, fmt.Errorf("tagId is required for remove_list_entry action")
	}
	if input.ListParameterKey == "" {
		return nil, nil, fmt.Errorf("listParameterKey is required for remove_list_entry action")
	}
	if len(input.MatchBy) == 0 {
		return nil, nil, fmt.Errorf("matchBy is required for remove_list_entry action — provide key=value pairs to match the entry to remove")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tag, err := wc.Client.GetTag(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tag %s: %w", input.TagID, err)
	}

	params := getTagParams(tag)
	if params == nil {
		return nil, nil, fmt.Errorf("tag %s (%s) has no parameters", input.TagID, tag.Name)
	}

	idx, existingEntries := findListParameter(params, input.ListParameterKey)
	if idx == -1 {
		return nil, nil, fmt.Errorf("tag %s (%s) does not have a list parameter with key %q", input.TagID, tag.Name, input.ListParameterKey)
	}

	previousSize := len(existingEntries)
	removed, remaining := removeListEntry(existingEntries, input.MatchBy)

	if len(removed) == 0 {
		return nil, RemoveListEntryOutput{
			Success:          true,
			EntityType:       "tag",
			EntityID:         input.TagID,
			EntityName:       tag.Name,
			ListParameterKey: input.ListParameterKey,
			Removed:          false,
			PreviousSize:     previousSize,
			CurrentSize:      previousSize,
			Fingerprint:      tag.Fingerprint,
			Message:          "No entries matched the given matchBy criteria",
		}, nil
	}

	// Update the list in params
	params[idx].List = remaining

	path := BuildTagPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)
	tagInput := &TagInput{
		Name:            tag.Name,
		Type:            tag.Type,
		Parameter:       params,
		FiringTriggerId: tag.FiringTriggerID,
	}

	updatedTag, err := wc.Client.UpdateTag(tCtx, path, tagInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update tag %s: %w", input.TagID, err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	// Collect removed entry fields for the response
	var removedEntry map[string]string
	if len(removed) > 0 {
		removedEntry = removed[0].Fields
	}

	return nil, RemoveListEntryOutput{
		Success:          true,
		EntityType:       "tag",
		EntityID:         input.TagID,
		EntityName:       tag.Name,
		ListParameterKey: input.ListParameterKey,
		Removed:          true,
		RemovedEntry:     removedEntry,
		PreviousSize:     previousSize,
		CurrentSize:      len(remaining),
		Fingerprint:      updatedTag.Fingerprint,
		Message:          fmt.Sprintf("%d entries removed from tag %s (%s)", len(removed), input.TagID, tag.Name),
	}, nil
}

func handleTagListEntries(ctx context.Context, input TagToolInput) (*mcp.CallToolResult, any, error) {
	if input.TagID == "" {
		return nil, nil, fmt.Errorf("tagId is required for list_entries action")
	}
	if input.ListParameterKey == "" {
		return nil, nil, fmt.Errorf("listParameterKey is required for list_entries action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tag, err := wc.Client.GetTag(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TagID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tag %s: %w", input.TagID, err)
	}

	params := getTagParams(tag)
	if params == nil {
		return nil, ListEntriesOutput{
			EntityType:       "tag",
			EntityID:         input.TagID,
			EntityName:       tag.Name,
			ListParameterKey: input.ListParameterKey,
			Total:            0,
			Entries:          []map[string]string{},
		}, nil
	}

	_, existingEntries := findListParameter(params, input.ListParameterKey)
	if existingEntries == nil {
		return nil, ListEntriesOutput{
			EntityType:       "tag",
			EntityID:         input.TagID,
			EntityName:       tag.Name,
			ListParameterKey: input.ListParameterKey,
			Total:            0,
			Entries:          []map[string]string{},
		}, nil
	}

	flat := flattenEntries(existingEntries)
	entries := make([]map[string]string, 0, len(flat))
	for _, le := range flat {
		entries = append(entries, le.Fields)
	}

	return nil, ListEntriesOutput{
		EntityType:       "tag",
		EntityID:         input.TagID,
		EntityName:       tag.Name,
		ListParameterKey: input.ListParameterKey,
		Total:            len(entries),
		Entries:          entries,
	}, nil
}

