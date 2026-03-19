package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// WorkspaceToolInput is the unified input for the workspace tool.
type WorkspaceToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,create,status,description:Operation to perform on workspaces"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	// Fields for create:
	WorkspaceID string `json:"workspaceId,omitempty" jsonschema:"description:The GTM workspace ID (required for status)"`
	Name        string `json:"name,omitempty" jsonschema:"description:Workspace name (required for create)"`
	Description string `json:"description,omitempty" jsonschema:"description:Workspace description (optional, for create)"`
}


func handleListWorkspaces(ctx context.Context, input WorkspaceToolInput) (*mcp.CallToolResult, any, error) {
	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	workspaces, err := cc.Client.ListWorkspaces(tCtx, cc.AccountID, cc.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	return nil, ListWorkspacesOutput{Workspaces: workspaces}, nil
}

func handleCreateWorkspace(ctx context.Context, input WorkspaceToolInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}

	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := cc.ContainerPath()
	workspace := &tagmanager.Workspace{
		Name:        input.Name,
		Description: input.Description,
	}

	created, err := cc.Client.Service.Accounts.Containers.Workspaces.Create(parent, workspace).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	return nil, CreateWorkspaceOutput{
		Success: true,
		Workspace: CreatedWorkspace{
			WorkspaceID:   created.WorkspaceId,
			Name:          created.Name,
			Description:   created.Description,
			Path:          created.Path,
			TagManagerUrl: created.TagManagerUrl,
		},
		Message: "Workspace created successfully",
	}, nil
}

func handleGetWorkspaceStatus(ctx context.Context, input WorkspaceToolInput) (*mcp.CallToolResult, any, error) {
	if input.WorkspaceID == "" {
		return nil, nil, fmt.Errorf("workspaceId is required for status action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	status, err := wc.Client.GetWorkspaceStatus(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetWorkspaceStatusOutput{Status: *status}, nil
}
