package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// GtagConfigToolInput is the unified input for the gtag_config tool.
type GtagConfigToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,update,delete,description:Operation to perform on Google tag configs"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	// Fields for get/update/delete:
	GtagConfigID string `json:"gtagConfigId,omitempty" jsonschema:"description:Google tag config ID (required for get, update, delete)"`
	// Fields for create/update:
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:Gtag config parameters as JSON array (optional)"`
	Type           string `json:"type,omitempty" jsonschema:"description:Gtag config type (optional for create)"`
	// Fields for delete:
	Confirm     bool   `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for update)"`
}

type GtagConfigInfo struct {
	GtagConfigID string `json:"gtagConfigId"`
	Type         string `json:"type,omitempty"`
	Path         string `json:"path,omitempty"`
	Fingerprint  string `json:"fingerprint,omitempty"`
}

type ListGtagConfigsOutput struct {
	GtagConfigs []GtagConfigInfo `json:"gtagConfigs"`
}

type GetGtagConfigOutput struct {
	GtagConfig interface{} `json:"gtagConfig"`
}

type CreateGtagConfigOutput struct {
	Success    bool           `json:"success"`
	GtagConfig GtagConfigInfo `json:"gtagConfig"`
	Message    string         `json:"message"`
}

type UpdateGtagConfigOutput struct {
	Success    bool           `json:"success"`
	GtagConfig GtagConfigInfo `json:"gtagConfig"`
	Message    string         `json:"message"`
}

type DeleteGtagConfigOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}


func handleGtagConfigList(ctx context.Context, input GtagConfigToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "gtag_config_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := wc.WorkspacePath()
	resp, err := wc.Client.Service.Accounts.Containers.Workspaces.GtagConfig.List(parent).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	configs := make([]GtagConfigInfo, 0)
	if resp.GtagConfig != nil {
		for _, gc := range resp.GtagConfig {
			configs = append(configs, GtagConfigInfo{
				GtagConfigID: gc.GtagConfigId,
				Type:         gc.Type,
				Path:         gc.Path,
				Fingerprint:  gc.Fingerprint,
			})
		}
	}

	out := ListGtagConfigsOutput{GtagConfigs: configs}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleGtagConfigGet(ctx context.Context, input GtagConfigToolInput) (*mcp.CallToolResult, any, error) {
	if input.GtagConfigID == "" {
		return nil, nil, fmt.Errorf("gtagConfigId is required for get action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/gtag_config/%s", wc.WorkspacePath(), input.GtagConfigID)
	gc, err := wc.Client.Service.Accounts.Containers.Workspaces.GtagConfig.Get(path).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	return nil, GetGtagConfigOutput{GtagConfig: gc}, nil
}

func handleGtagConfigCreate(ctx context.Context, input GtagConfigToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := wc.WorkspacePath()
	gc := &tagmanager.GtagConfig{
		Type: input.Type,
	}

	created, err := wc.Client.Service.Accounts.Containers.Workspaces.GtagConfig.Create(parent, gc).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, CreateGtagConfigOutput{
		Success: true,
		GtagConfig: GtagConfigInfo{
			GtagConfigID: created.GtagConfigId,
			Type:         created.Type,
			Path:         created.Path,
			Fingerprint:  created.Fingerprint,
		},
		Message: fmt.Sprintf("Gtag config created successfully (ID: %s)", created.GtagConfigId),
	}, nil
}

func handleGtagConfigUpdate(ctx context.Context, input GtagConfigToolInput) (*mcp.CallToolResult, any, error) {
	if input.GtagConfigID == "" {
		return nil, nil, fmt.Errorf("gtagConfigId is required for update action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/gtag_config/%s", wc.WorkspacePath(), input.GtagConfigID)

	gc := &tagmanager.GtagConfig{
		Type: input.Type,
	}

	call := wc.Client.Service.Accounts.Containers.Workspaces.GtagConfig.Update(path, gc).Context(tCtx)
	if input.Fingerprint != "" {
		call = call.Fingerprint(input.Fingerprint)
	}

	updated, err := call.Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, UpdateGtagConfigOutput{
		Success: true,
		GtagConfig: GtagConfigInfo{
			GtagConfigID: updated.GtagConfigId,
			Type:         updated.Type,
			Path:         updated.Path,
			Fingerprint:  updated.Fingerprint,
		},
		Message: fmt.Sprintf("Gtag config %s updated successfully", input.GtagConfigID),
	}, nil
}

func handleGtagConfigDelete(ctx context.Context, input GtagConfigToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteGtagConfigOutput{
			Success: false,
			Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		}, nil
	}
	if input.GtagConfigID == "" {
		return nil, nil, fmt.Errorf("gtagConfigId is required for delete action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/gtag_config/%s", wc.WorkspacePath(), input.GtagConfigID)
	if err := wc.Client.Service.Accounts.Containers.Workspaces.GtagConfig.Delete(path).Context(tCtx).Do(); err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DeleteGtagConfigOutput{
		Success: true,
		Message: fmt.Sprintf("Gtag config %s deleted successfully", input.GtagConfigID),
	}, nil
}
