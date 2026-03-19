package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EnvironmentToolInput is the unified input for the environment tool.
type EnvironmentToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,update,delete,description:Operation to perform on environments"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	// Fields for get/update/delete:
	EnvironmentID string `json:"environmentId,omitempty" jsonschema:"description:Environment ID (required for get, update, delete)"`
	// Fields for create/update:
	Name               string `json:"name,omitempty" jsonschema:"description:Environment name (required for create/update)"`
	Description        string `json:"description,omitempty" jsonschema:"description:Environment description (optional)"`
	ContainerVersionID string `json:"containerVersionId,omitempty" jsonschema:"description:Container version ID to point this environment at (optional)"`
	EnableDebug        bool   `json:"enableDebug,omitempty" jsonschema:"description:Enable debug mode (optional)"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
}


func handleEnvironmentList(ctx context.Context, input EnvironmentToolInput) (*mcp.CallToolResult, any, error) {
	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	envs, err := cc.Client.ListEnvironments(tCtx, cc.AccountID, cc.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	return nil, ListEnvironmentsOutput{Environments: envs}, nil
}

func handleEnvironmentGet(ctx context.Context, input EnvironmentToolInput) (*mcp.CallToolResult, any, error) {
	if input.EnvironmentID == "" {
		return nil, nil, fmt.Errorf("environmentId is required for get action")
	}

	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	env, err := cc.Client.GetEnvironment(tCtx, cc.AccountID, cc.ContainerID, input.EnvironmentID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetEnvironmentOutput{Environment: *env}, nil
}

func handleEnvironmentCreate(ctx context.Context, input EnvironmentToolInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return nil, nil, fmt.Errorf("environment name is required")
	}

	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	envInput := &EnvironmentInput{
		Name:               input.Name,
		Description:        input.Description,
		ContainerVersionID: input.ContainerVersionID,
		EnableDebug:        input.EnableDebug,
	}

	env, err := cc.Client.CreateEnvironment(tCtx, cc.AccountID, cc.ContainerID, envInput)
	if err != nil {
		return nil, nil, err
	}

	return nil, CreateEnvironmentOutput{
		Success:     true,
		Environment: *env,
		Message:     fmt.Sprintf("Environment '%s' created successfully", input.Name),
	}, nil
}

func handleEnvironmentUpdate(ctx context.Context, input EnvironmentToolInput) (*mcp.CallToolResult, any, error) {
	if input.EnvironmentID == "" {
		return nil, nil, fmt.Errorf("environmentId is required for update action")
	}
	if input.Name == "" {
		return nil, nil, fmt.Errorf("environment name is required")
	}

	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildEnvironmentPath(cc.AccountID, cc.ContainerID, input.EnvironmentID)

	envInput := &EnvironmentInput{
		Name:               input.Name,
		Description:        input.Description,
		ContainerVersionID: input.ContainerVersionID,
		EnableDebug:        input.EnableDebug,
	}

	env, err := cc.Client.UpdateEnvironment(tCtx, path, envInput)
	if err != nil {
		return nil, nil, err
	}

	return nil, UpdateEnvironmentOutput{
		Success:     true,
		Environment: *env,
		Message:     fmt.Sprintf("Environment '%s' updated successfully", input.Name),
	}, nil
}

func handleEnvironmentDelete(ctx context.Context, input EnvironmentToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteEnvironmentOutput{
			Success: false,
			Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		}, nil
	}
	if input.EnvironmentID == "" {
		return nil, nil, fmt.Errorf("environmentId is required for delete action")
	}

	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildEnvironmentPath(cc.AccountID, cc.ContainerID, input.EnvironmentID)
	if err := cc.Client.DeleteEnvironment(tCtx, path); err != nil {
		return nil, nil, err
	}

	return nil, DeleteEnvironmentOutput{
		Success: true,
		Message: fmt.Sprintf("Environment %s deleted successfully", input.EnvironmentID),
	}, nil
}
