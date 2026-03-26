package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TriggerToolInput is the unified input for the trigger tool.
type TriggerToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,update,delete,revert,description:Operation to perform on triggers"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	// Fields for get/update/delete:
	TriggerID string `json:"triggerId,omitempty" jsonschema:"description:Trigger ID (required for get, update, delete)"`
	// Fields for create/update:
	Name                  string `json:"name,omitempty" jsonschema:"description:Trigger name (required for create/update)"`
	Type                  string `json:"type,omitempty" jsonschema:"description:Trigger type e.g. pageview, customEvent, linkClick, formSubmission, timer (required for create/update)"`
	FilterJSON            string `json:"filterJson,omitempty" jsonschema:"description:Filter conditions as JSON array for pageview triggers"`
	AutoEventFilterJSON   string `json:"autoEventFilterJson,omitempty" jsonschema:"description:Auto-event filter as JSON array for click/form triggers"`
	CustomEventFilterJSON string `json:"customEventFilterJson,omitempty" jsonschema:"description:Custom event filter as JSON array (REQUIRED for customEvent type)"`
	EventNameJSON         string `json:"eventNameJson,omitempty" jsonschema:"description:Event name as JSON object {type, value} for timer triggers"`
	ParameterJSON         string `json:"parameterJson,omitempty" jsonschema:"description:Trigger parameters as JSON array. For triggerGroup: [{key: triggerIds, type: list, list: [{type: triggerReference, value: triggerId}]}]"`
	Notes                 string `json:"notes,omitempty" jsonschema:"description:Trigger notes (optional)"`
	WaitForTagsJSON       string `json:"waitForTagsJson,omitempty" jsonschema:"description:Whether to wait for tags as JSON Parameter {type: boolean, value: true/false}. For link click/form submit triggers."`
	CheckValidationJSON   string `json:"checkValidationJson,omitempty" jsonschema:"description:Whether to check validation as JSON Parameter {type: boolean, value: true/false}. For link click/form submit triggers."`
	WaitForTagsTimeoutJSON string `json:"waitForTagsTimeoutJson,omitempty" jsonschema:"description:Max wait time in ms as JSON Parameter {type: integer, value: 2000}. For link click/form submit triggers."`
	ParentFolderID        string `json:"parentFolderId,omitempty" jsonschema:"description:Parent folder ID for organizational purposes (optional)"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
}


func handleTriggerList(ctx context.Context, input TriggerToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "trigger_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	triggers, err := wc.Client.ListTriggers(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	out := ListTriggersOutput{Triggers: triggers}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleTriggerGet(ctx context.Context, input TriggerToolInput) (*mcp.CallToolResult, any, error) {
	if input.TriggerID == "" {
		return nil, nil, fmt.Errorf("triggerId is required for get action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	trigger, err := wc.Client.GetTrigger(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TriggerID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetTriggerOutput{Trigger: *trigger}, nil
}

func handleTriggerCreate(ctx context.Context, input TriggerToolInput) (*mcp.CallToolResult, any, error) {
	if err := ValidateTriggerInput(input.Name, input.Type); err != nil {
		return nil, nil, err
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var filter []Condition
	if input.FilterJSON != "" {
		if err := json.Unmarshal([]byte(input.FilterJSON), &filter); err != nil {
			return nil, nil, fmt.Errorf("invalid filterJson: %w", err)
		}
	}

	var autoEventFilter []Condition
	if input.AutoEventFilterJSON != "" {
		if err := json.Unmarshal([]byte(input.AutoEventFilterJSON), &autoEventFilter); err != nil {
			return nil, nil, fmt.Errorf("invalid autoEventFilterJson: %w", err)
		}
	}

	var customEventFilter []Condition
	if input.CustomEventFilterJSON != "" {
		if err := json.Unmarshal([]byte(input.CustomEventFilterJSON), &customEventFilter); err != nil {
			return nil, nil, fmt.Errorf("invalid customEventFilterJson: %w", err)
		}
	}

	var eventName *Parameter
	if input.EventNameJSON != "" {
		eventName = &Parameter{}
		if err := json.Unmarshal([]byte(input.EventNameJSON), eventName); err != nil {
			return nil, nil, fmt.Errorf("invalid eventNameJson: %w", err)
		}
	}

	var waitForTags *Parameter
	if input.WaitForTagsJSON != "" {
		waitForTags = &Parameter{}
		if err := json.Unmarshal([]byte(input.WaitForTagsJSON), waitForTags); err != nil {
			return nil, nil, fmt.Errorf("invalid waitForTagsJson: %w", err)
		}
	}

	var checkValidation *Parameter
	if input.CheckValidationJSON != "" {
		checkValidation = &Parameter{}
		if err := json.Unmarshal([]byte(input.CheckValidationJSON), checkValidation); err != nil {
			return nil, nil, fmt.Errorf("invalid checkValidationJson: %w", err)
		}
	}

	var waitForTagsTimeout *Parameter
	if input.WaitForTagsTimeoutJSON != "" {
		waitForTagsTimeout = &Parameter{}
		if err := json.Unmarshal([]byte(input.WaitForTagsTimeoutJSON), waitForTagsTimeout); err != nil {
			return nil, nil, fmt.Errorf("invalid waitForTagsTimeoutJson: %w", err)
		}
	}

	triggerInput := &TriggerInput{
		Name:               input.Name,
		Type:               input.Type,
		Filter:             filter,
		AutoEventFilter:    autoEventFilter,
		CustomEventFilter:  customEventFilter,
		EventName:          eventName,
		Notes:              input.Notes,
		WaitForTags:        waitForTags,
		CheckValidation:    checkValidation,
		WaitForTagsTimeout: waitForTagsTimeout,
		ParentFolderID:     input.ParentFolderID,
	}

	trigger, err := wc.Client.CreateTrigger(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, triggerInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, CreateTriggerOutput{
		Success: true,
		Trigger: *trigger,
		Message: "Trigger created successfully",
	}, nil
}

func handleTriggerUpdate(ctx context.Context, input TriggerToolInput) (*mcp.CallToolResult, any, error) {
	if input.TriggerID == "" {
		return nil, nil, fmt.Errorf("triggerId is required for update action")
	}
	if err := ValidateTriggerInput(input.Name, input.Type); err != nil {
		return nil, nil, err
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildTriggerPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TriggerID)

	var filter []Condition
	if input.FilterJSON != "" {
		if err := json.Unmarshal([]byte(input.FilterJSON), &filter); err != nil {
			return nil, nil, fmt.Errorf("invalid filterJson: %w", err)
		}
	}

	var autoEventFilter []Condition
	if input.AutoEventFilterJSON != "" {
		if err := json.Unmarshal([]byte(input.AutoEventFilterJSON), &autoEventFilter); err != nil {
			return nil, nil, fmt.Errorf("invalid autoEventFilterJson: %w", err)
		}
	}

	var customEventFilter []Condition
	if input.CustomEventFilterJSON != "" {
		if err := json.Unmarshal([]byte(input.CustomEventFilterJSON), &customEventFilter); err != nil {
			return nil, nil, fmt.Errorf("invalid customEventFilterJson: %w", err)
		}
	}

	var params []Parameter
	if input.ParameterJSON != "" {
		if err := json.Unmarshal([]byte(input.ParameterJSON), &params); err != nil {
			return nil, nil, fmt.Errorf("invalid parameterJson: %w", err)
		}
	}

	var waitForTags *Parameter
	if input.WaitForTagsJSON != "" {
		waitForTags = &Parameter{}
		if err := json.Unmarshal([]byte(input.WaitForTagsJSON), waitForTags); err != nil {
			return nil, nil, fmt.Errorf("invalid waitForTagsJson: %w", err)
		}
	}

	var checkValidation *Parameter
	if input.CheckValidationJSON != "" {
		checkValidation = &Parameter{}
		if err := json.Unmarshal([]byte(input.CheckValidationJSON), checkValidation); err != nil {
			return nil, nil, fmt.Errorf("invalid checkValidationJson: %w", err)
		}
	}

	var waitForTagsTimeout *Parameter
	if input.WaitForTagsTimeoutJSON != "" {
		waitForTagsTimeout = &Parameter{}
		if err := json.Unmarshal([]byte(input.WaitForTagsTimeoutJSON), waitForTagsTimeout); err != nil {
			return nil, nil, fmt.Errorf("invalid waitForTagsTimeoutJson: %w", err)
		}
	}

	triggerInput := &TriggerInput{
		Name:               input.Name,
		Type:               input.Type,
		Filter:             filter,
		AutoEventFilter:    autoEventFilter,
		CustomEventFilter:  customEventFilter,
		Parameter:          params,
		Notes:              input.Notes,
		WaitForTags:        waitForTags,
		CheckValidation:    checkValidation,
		WaitForTagsTimeout: waitForTagsTimeout,
		ParentFolderID:     input.ParentFolderID,
	}

	trigger, err := wc.Client.UpdateTrigger(tCtx, path, triggerInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, UpdateTriggerOutput{
		Success: true,
		Trigger: *trigger,
		Message: "Trigger updated successfully",
	}, nil
}

func handleTriggerDelete(ctx context.Context, input TriggerToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteTriggerOutput{
			Success: false,
			Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		}, nil
	}
	if input.TriggerID == "" {
		return nil, nil, fmt.Errorf("triggerId is required for delete action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildTriggerPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TriggerID)
	if err := wc.Client.DeleteTrigger(tCtx, path); err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DeleteTriggerOutput{
		Success: true,
		Message: fmt.Sprintf("Trigger %s deleted successfully", input.TriggerID),
	}, nil
}

func handleTriggerRevert(ctx context.Context, input TriggerToolInput) (*mcp.CallToolResult, any, error) {
	if input.TriggerID == "" {
		return nil, nil, fmt.Errorf("triggerId is required for revert action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildTriggerPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TriggerID)
	call := wc.Client.Service.Accounts.Containers.Workspaces.Triggers.Revert(path).Context(tCtx)
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
		Message: fmt.Sprintf("Trigger %s reverted to latest published version", input.TriggerID),
		Entity:  resp.Trigger,
	}, nil
}
