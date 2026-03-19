package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// VariableToolInput is the unified input for the variable tool.
type VariableToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,update,delete,revert,description:Operation to perform on variables"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	// Fields for get/update/delete:
	VariableID string `json:"variableId,omitempty" jsonschema:"description:Variable ID (required for get, update, delete)"`
	// Fields for create/update:
	Name           string `json:"name,omitempty" jsonschema:"description:Variable name (required for create/update)"`
	Type           string `json:"type,omitempty" jsonschema:"description:Variable type e.g. c (Constant), v (Data Layer), k (Cookie), jsm (Custom JavaScript), u (URL) (required for create/update)"`
	Parameter      []Parameter `json:"parameter,omitempty" jsonschema:"description:Variable parameters as array of objects. Each: {type, key, value}. Supports nested list/map."`
	ParametersJSON string      `json:"parametersJson,omitempty" jsonschema:"description:DEPRECATED: Variable parameters as JSON string. Use parameter array instead."`
	Notes          string `json:"notes,omitempty" jsonschema:"description:Variable notes (optional)"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
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
