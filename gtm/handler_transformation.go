package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TransformationToolInput is the unified input for the transformation tool.
type TransformationToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,update,delete,revert,description:Operation to perform on transformations"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	// Fields for get/update/delete:
	TransformationID string `json:"transformationId,omitempty" jsonschema:"description:Transformation ID (required for get, update, delete)"`
	// Fields for create/update:
	Name           string `json:"name,omitempty" jsonschema:"description:Transformation name (required for create/update)"`
	Type           string `json:"type,omitempty" jsonschema:"description:Transformation type: tf_exclude_params, tf_allow_params, or tf_augment_event (required for create)"`
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:Transformation parameters as JSON array (optional)"`
	Notes          string `json:"notes,omitempty" jsonschema:"description:Transformation notes (optional)"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
}


func handleTransformationList(ctx context.Context, input TransformationToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "transformation_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	transformations, err := wc.Client.ListTransformations(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	out := ListTransformationsOutput{Transformations: transformations}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleTransformationGet(ctx context.Context, input TransformationToolInput) (*mcp.CallToolResult, any, error) {
	if input.TransformationID == "" {
		return nil, nil, fmt.Errorf("transformationId is required for get action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	t, err := wc.Client.GetTransformation(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TransformationID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetTransformationOutput{Transformation: *t}, nil
}

func handleTransformationCreate(ctx context.Context, input TransformationToolInput) (*mcp.CallToolResult, any, error) {
	if err := ValidateTransformationInput(input.Name, input.Type); err != nil {
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
			return nil, nil, fmt.Errorf("invalid parametersJson: %w", err)
		}
	}

	transformationInput := &TransformationInput{
		Name:      input.Name,
		Type:      input.Type,
		Parameter: params,
		Notes:     input.Notes,
	}

	t, err := wc.Client.CreateTransformation(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, transformationInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, CreateTransformationOutput{
		Success:        true,
		Transformation: *t,
		Message:        "Transformation created successfully",
	}, nil
}

func handleTransformationUpdate(ctx context.Context, input TransformationToolInput) (*mcp.CallToolResult, any, error) {
	if input.TransformationID == "" {
		return nil, nil, fmt.Errorf("transformationId is required for update action")
	}
	if err := ValidateTransformationInput(input.Name, input.Type); err != nil {
		return nil, nil, err
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildTransformationPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TransformationID)

	var params []Parameter
	if input.ParametersJSON != "" {
		if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
			return nil, nil, fmt.Errorf("invalid parametersJson: %w", err)
		}
	}

	transformationInput := &TransformationInput{
		Name:      input.Name,
		Type:      input.Type,
		Parameter: params,
		Notes:     input.Notes,
	}

	t, err := wc.Client.UpdateTransformation(tCtx, path, transformationInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, UpdateTransformationOutput{
		Success:        true,
		Transformation: *t,
		Message:        "Transformation updated successfully",
	}, nil
}

func handleTransformationDelete(ctx context.Context, input TransformationToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteTransformationOutput{
			Success: false,
			Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		}, nil
	}
	if input.TransformationID == "" {
		return nil, nil, fmt.Errorf("transformationId is required for delete action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildTransformationPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TransformationID)
	if err := wc.Client.DeleteTransformation(tCtx, path); err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DeleteTransformationOutput{
		Success: true,
		Message: fmt.Sprintf("Transformation %s deleted successfully", input.TransformationID),
	}, nil
}

func handleTransformationRevert(ctx context.Context, input TransformationToolInput) (*mcp.CallToolResult, any, error) {
	if input.TransformationID == "" {
		return nil, nil, fmt.Errorf("transformationId is required for revert action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildTransformationPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.TransformationID)
	call := wc.Client.Service.Accounts.Containers.Workspaces.Transformations.Revert(path).Context(tCtx)
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
		Message: fmt.Sprintf("Transformation %s reverted to latest published version", input.TransformationID),
		Entity:  resp.Transformation,
	}, nil
}
