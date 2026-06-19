package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type BuiltInVariableToolInput struct {
	Action      string   `json:"action" jsonschema:"enum:list,enable,disable,revert,description:Operation to perform on built-in variables"`
	AccountID   string   `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string   `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string   `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	Types       []string `json:"types,omitempty" jsonschema:"description:Built-in variable types to enable/disable (e.g. eventName, clientName, requestPath, pageUrl, event)"`
	Confirm     bool     `json:"confirm,omitempty" jsonschema:"description:Must be true for disable (safety guard)"`
}

func handleBuiltInVariableList(ctx context.Context, input BuiltInVariableToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "built_in_variable_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	vars, err := wc.Client.ListBuiltInVariables(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	out := ListBuiltInVariablesOutput{BuiltInVariables: vars}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleBuiltInVariableEnable(ctx context.Context, input BuiltInVariableToolInput) (*mcp.CallToolResult, any, error) {
	if len(input.Types) == 0 {
		return nil, nil, fmt.Errorf("at least one built-in variable type is required")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	vars, err := wc.Client.EnableBuiltInVariables(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.Types)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, EnableBuiltInVariablesOutput{
		Success:          true,
		BuiltInVariables: vars,
		Message:          "Built-in variables enabled successfully",
	}, nil
}

func handleBuiltInVariableDisable(ctx context.Context, input BuiltInVariableToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DisableBuiltInVariablesOutput{
			Success: false,
			Message: "Disabling requires confirm: true. This is a safety guard to prevent accidental changes.",
		}, nil
	}
	if len(input.Types) == 0 {
		return nil, nil, fmt.Errorf("at least one built-in variable type is required")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := wc.Client.DisableBuiltInVariables(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.Types); err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DisableBuiltInVariablesOutput{
		Success: true,
		Message: "Built-in variables disabled successfully",
	}, nil
}

func handleBuiltInVariableRevert(ctx context.Context, input BuiltInVariableToolInput) (*mcp.CallToolResult, any, error) {
	if len(input.Types) == 0 {
		return nil, nil, fmt.Errorf("at least one built-in variable type is required for revert action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := wc.WorkspacePath()
	resp, err := wc.Client.Service.Accounts.Containers.Workspaces.BuiltInVariables.Revert(path).Type(input.Types[0]).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, RevertOutput{
		Success: true,
		Message: fmt.Sprintf("Built-in variable reverted (enabled=%v)", resp.Enabled),
		Entity:  resp,
	}, nil
}
