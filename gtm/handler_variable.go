package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// VariableToolInput is the unified input for the variable tool.
type VariableToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,update,delete,revert,add_lookup_entry,remove_lookup_entry,list_lookup_entries,description:Operation to perform on variables"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	// Fields for get/update/delete:
	VariableID string `json:"variableId,omitempty" jsonschema:"description:Variable ID (required for get, update, delete, and lookup operations)"`
	// Fields for create/update:
	Name           string `json:"name,omitempty" jsonschema:"description:Variable name (required for create/update)"`
	Type           string `json:"type,omitempty" jsonschema:"description:Variable type e.g. c (Constant), v (Data Layer), k (Cookie), jsm (Custom JavaScript), u (URL) (required for create/update)"`
	Parameter      []Parameter `json:"parameter,omitempty" jsonschema:"description:Variable parameters as array of objects. Each: {type, key, value}. Supports nested list/map."`
	ParametersJSON string      `json:"parametersJson,omitempty" jsonschema:"description:DEPRECATED: Variable parameters as JSON string. Use parameter array instead."`
	Notes          string `json:"notes,omitempty" jsonschema:"description:Variable notes (optional)"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
	// Fields for lookup table operations:
	Entries  []LookupEntry `json:"entries,omitempty" jsonschema:"description:Entries to add to the lookup/regex table. Each: {pattern, output, extra}. Required for add_lookup_entry."`
	Patterns []string      `json:"patterns,omitempty" jsonschema:"description:Regex patterns to remove from the lookup table. Required for remove_lookup_entry."`
}


func handleVariableList(ctx context.Context, input VariableToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "variable_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	variables, err := wc.Client.ListVariables(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	out := ListVariablesOutput{Variables: variables}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleVariableGet(ctx context.Context, input VariableToolInput) (*mcp.CallToolResult, any, error) {
	if input.VariableID == "" {
		return nil, nil, fmt.Errorf("variableId is required for get action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	variable, err := wc.Client.GetVariable(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetVariableOutput{Variable: *variable}, nil
}

func handleVariableCreate(ctx context.Context, input VariableToolInput) (*mcp.CallToolResult, any, error) {
	if err := ValidateVariableInput(input.Name, input.Type); err != nil {
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

	variableInput := &VariableInput{
		Name:      input.Name,
		Type:      input.Type,
		Parameter: params,
		Notes:     input.Notes,
	}

	variable, err := wc.Client.CreateVariable(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, variableInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, CreateVariableOutput{
		Success:  true,
		Variable: *variable,
		Message:  "Variable created successfully",
	}, nil
}

func handleVariableUpdate(ctx context.Context, input VariableToolInput) (*mcp.CallToolResult, any, error) {
	if input.VariableID == "" {
		return nil, nil, fmt.Errorf("variableId is required for update action")
	}
	if input.Name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}
	if input.Type == "" {
		return nil, nil, fmt.Errorf("type is required")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildVariablePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)

	params, err := resolveParameters(input.Parameter, input.ParametersJSON)
	if err != nil {
		return nil, nil, err
	}

	variableInput := &VariableInput{
		Name:      input.Name,
		Type:      input.Type,
		Parameter: params,
		Notes:     input.Notes,
	}

	variable, err := wc.Client.UpdateVariable(tCtx, path, variableInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, UpdateVariableOutput{
		Success:  true,
		Variable: *variable,
		Message:  "Variable updated successfully",
	}, nil
}

func handleVariableDelete(ctx context.Context, input VariableToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteVariableOutput{
			Success: false,
			Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		}, nil
	}
	if input.VariableID == "" {
		return nil, nil, fmt.Errorf("variableId is required for delete action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildVariablePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)
	if err := wc.Client.DeleteVariable(tCtx, path); err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DeleteVariableOutput{
		Success: true,
		Message: fmt.Sprintf("Variable %s deleted successfully", input.VariableID),
	}, nil
}

func handleVariableRevert(ctx context.Context, input VariableToolInput) (*mcp.CallToolResult, any, error) {
	if input.VariableID == "" {
		return nil, nil, fmt.Errorf("variableId is required for revert action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildVariablePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)
	call := wc.Client.Service.Accounts.Containers.Workspaces.Variables.Revert(path).Context(tCtx)
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
		Message: fmt.Sprintf("Variable %s reverted to latest published version", input.VariableID),
		Entity:  resp.Variable,
	}, nil
}

// --- Lookup table operations ---

// AddLookupEntryOutput is returned by the add_lookup_entry action.
type AddLookupEntryOutput struct {
	Success    bool          `json:"success"`
	Message    string        `json:"message"`
	Added      []LookupEntry `json:"added,omitempty"`
	Duplicates []LookupEntry `json:"duplicates,omitempty"`
	TotalAfter int           `json:"totalAfter"`
}

// RemoveLookupEntryOutput is returned by the remove_lookup_entry action.
type RemoveLookupEntryOutput struct {
	Success    bool          `json:"success"`
	Message    string        `json:"message"`
	Removed    []LookupEntry `json:"removed,omitempty"`
	TotalAfter int           `json:"totalAfter"`
}

// ListLookupEntriesOutput is returned by the list_lookup_entries action.
type ListLookupEntriesOutput struct {
	VariableID   string        `json:"variableId"`
	VariableName string        `json:"variableName"`
	Entries      []LookupEntry `json:"entries"`
	Total        int           `json:"total"`
}

func handleVariableAddLookupEntry(ctx context.Context, input VariableToolInput) (*mcp.CallToolResult, any, error) {
	if input.VariableID == "" {
		return nil, nil, fmt.Errorf("variableId is required for add_lookup_entry action")
	}
	if len(input.Entries) == 0 {
		return nil, nil, fmt.Errorf("entries is required for add_lookup_entry action (at least one entry)")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// GET current variable
	variable, err := wc.Client.GetVariable(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get variable %s: %w", input.VariableID, err)
	}

	// Convert API params to our Parameter type
	params := getVariableParams(variable)

	// Find the regex table
	idx, existingEntries := findRegexTable(params)
	if idx == -1 {
		return nil, nil, fmt.Errorf("variable %s does not have a regexTable parameter — this action only works on RegEx Table variables", input.VariableID)
	}

	// Merge entries
	added, duplicates, merged := mergeEntries(existingEntries, input.Entries)

	if len(added) == 0 {
		return nil, AddLookupEntryOutput{
			Success:    true,
			Message:    fmt.Sprintf("No new entries added — all %d entries already exist in the table", len(duplicates)),
			Duplicates: duplicates,
			TotalAfter: len(existingEntries),
		}, nil
	}

	// Update the regex table in params
	params[idx].List = merged

	// UPDATE variable
	path := BuildVariablePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)
	varInput := &VariableInput{
		Name:      variable.Name,
		Type:      variable.Type,
		Parameter: params,
	}

	_, err = wc.Client.UpdateVariable(tCtx, path, varInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update variable %s: %w", input.VariableID, err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	msg := fmt.Sprintf("%d entries added to variable %s (%s)", len(added), input.VariableID, variable.Name)
	if len(duplicates) > 0 {
		msg += fmt.Sprintf(". %d duplicate patterns skipped.", len(duplicates))
	}

	return nil, AddLookupEntryOutput{
		Success:    true,
		Message:    msg,
		Added:      added,
		Duplicates: duplicates,
		TotalAfter: len(merged),
	}, nil
}

func handleVariableRemoveLookupEntry(ctx context.Context, input VariableToolInput) (*mcp.CallToolResult, any, error) {
	if input.VariableID == "" {
		return nil, nil, fmt.Errorf("variableId is required for remove_lookup_entry action")
	}
	if len(input.Patterns) == 0 {
		return nil, nil, fmt.Errorf("patterns is required for remove_lookup_entry action (at least one pattern)")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// GET current variable
	variable, err := wc.Client.GetVariable(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get variable %s: %w", input.VariableID, err)
	}

	params := getVariableParams(variable)

	idx, existingEntries := findRegexTable(params)
	if idx == -1 {
		return nil, nil, fmt.Errorf("variable %s does not have a regexTable parameter", input.VariableID)
	}

	removed, remaining := removeEntries(existingEntries, input.Patterns)

	if len(removed) == 0 {
		return nil, RemoveLookupEntryOutput{
			Success:    true,
			Message:    fmt.Sprintf("No entries removed — none of the %d patterns matched", len(input.Patterns)),
			TotalAfter: len(existingEntries),
		}, nil
	}

	// Update the regex table in params
	params[idx].List = remaining

	path := BuildVariablePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)
	varInput := &VariableInput{
		Name:      variable.Name,
		Type:      variable.Type,
		Parameter: params,
	}

	_, err = wc.Client.UpdateVariable(tCtx, path, varInput)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update variable %s: %w", input.VariableID, err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, RemoveLookupEntryOutput{
		Success:    true,
		Message:    fmt.Sprintf("%d entries removed from variable %s (%s)", len(removed), input.VariableID, variable.Name),
		Removed:    removed,
		TotalAfter: len(remaining),
	}, nil
}

func handleVariableListLookupEntries(ctx context.Context, input VariableToolInput) (*mcp.CallToolResult, any, error) {
	if input.VariableID == "" {
		return nil, nil, fmt.Errorf("variableId is required for list_lookup_entries action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	variable, err := wc.Client.GetVariable(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.VariableID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get variable %s: %w", input.VariableID, err)
	}

	params := getVariableParams(variable)

	_, existingEntries := findRegexTable(params)
	if existingEntries == nil {
		return nil, ListLookupEntriesOutput{
			VariableID:   input.VariableID,
			VariableName: variable.Name,
			Entries:      []LookupEntry{},
			Total:        0,
		}, nil
	}

	entries := make([]LookupEntry, 0, len(existingEntries))
	for _, entry := range existingEntries {
		entries = append(entries, parseRegexEntry(entry))
	}

	return nil, ListLookupEntriesOutput{
		VariableID:   input.VariableID,
		VariableName: variable.Name,
		Entries:      entries,
		Total:        len(entries),
	}, nil
}

// getVariableParams extracts the Parameter slice from a Variable.
// Variable.Parameter is typed as `any` but stores []*tagmanager.Parameter.
func getVariableParams(v *Variable) []Parameter {
	apiParams, ok := v.Parameter.([]*tagmanager.Parameter)
	if !ok || len(apiParams) == 0 {
		return nil
	}
	result := make([]Parameter, 0, len(apiParams))
	for _, p := range apiParams {
		result = append(result, apiParamToParameter(p))
	}
	return result
}

// apiParamToParameter converts a Google API *tagmanager.Parameter to our Parameter type.
func apiParamToParameter(p *tagmanager.Parameter) Parameter {
	result := Parameter{
		Type:  p.Type,
		Key:   p.Key,
		Value: p.Value,
	}
	for _, child := range p.List {
		result.List = append(result.List, apiParamToParameter(child))
	}
	for _, child := range p.Map {
		result.Map = append(result.Map, apiParamToParameter(child))
	}
	return result
}
