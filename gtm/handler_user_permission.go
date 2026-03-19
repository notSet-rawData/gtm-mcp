package gtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// UserPermissionToolInput is the unified input for the user_permission tool.
type UserPermissionToolInput struct {
	Action    string `json:"action" jsonschema:"enum:list,get,create,update,delete,description:Operation to perform on user permissions"`
	AccountID string `json:"accountId" jsonschema:"description:The GTM account ID"`
	// Fields for get/update/delete:
	PermissionID string `json:"permissionId,omitempty" jsonschema:"description:Permission ID (required for get, update, delete)"`
	// Fields for create/update:
	EmailAddress        string `json:"emailAddress,omitempty" jsonschema:"description:User email address (required for create/update)"`
	AccountPermission   string `json:"accountPermission,omitempty" jsonschema:"description:Account-level permission: noAccess, read, edit, publish, admin"`
	ContainerAccessJSON string `json:"containerAccessJson,omitempty" jsonschema:"description:Container-level permissions as JSON array. Each entry: {containerId, permission}. Values: noAccess, read, edit, publish, approve"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
}


func handleUserPermissionList(ctx context.Context, input UserPermissionToolInput) (*mcp.CallToolResult, any, error) {
	if input.AccountID == "" {
		return nil, nil, fmt.Errorf("accountId is required")
	}

	client, err := resolveAccount(ctx, input.AccountID)
	if err != nil {
		return nil, nil, err
	}

	perms, err := client.ListUserPermissions(ctx, input.AccountID)
	if err != nil {
		return nil, nil, err
	}

	return nil, ListUserPermissionsOutput{Permissions: perms}, nil
}

func handleUserPermissionGet(ctx context.Context, input UserPermissionToolInput) (*mcp.CallToolResult, any, error) {
	if input.PermissionID == "" {
		return nil, nil, fmt.Errorf("permissionId is required for get action")
	}

	client, err := resolveAccount(ctx, input.AccountID)
	if err != nil {
		return nil, nil, err
	}

	perm, err := client.GetUserPermission(ctx, input.AccountID, input.PermissionID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetUserPermissionOutput{Permission: *perm}, nil
}

func handleUserPermissionCreate(ctx context.Context, input UserPermissionToolInput) (*mcp.CallToolResult, any, error) {
	if input.EmailAddress == "" {
		return nil, nil, fmt.Errorf("emailAddress is required")
	}

	client, err := resolveAccount(ctx, input.AccountID)
	if err != nil {
		return nil, nil, err
	}

	permInput := &UserPermissionInput{
		EmailAddress: input.EmailAddress,
	}

	if input.AccountPermission != "" {
		permInput.AccountAccess = &AccountAccess{
			Permission: input.AccountPermission,
		}
	}

	if input.ContainerAccessJSON != "" {
		var ca []ContainerAccess
		if err := json.Unmarshal([]byte(input.ContainerAccessJSON), &ca); err != nil {
			return nil, nil, fmt.Errorf("invalid containerAccessJson: %w", err)
		}
		permInput.ContainerAccess = ca
	}

	perm, err := client.CreateUserPermission(ctx, input.AccountID, permInput)
	if err != nil {
		return nil, nil, err
	}

	return nil, CreateUserPermissionOutput{
		Success:    true,
		Permission: *perm,
		Message:    fmt.Sprintf("User permission created for %s", input.EmailAddress),
	}, nil
}

func handleUserPermissionUpdate(ctx context.Context, input UserPermissionToolInput) (*mcp.CallToolResult, any, error) {
	if input.PermissionID == "" {
		return nil, nil, fmt.Errorf("permissionId is required for update action")
	}
	if input.EmailAddress == "" {
		return nil, nil, fmt.Errorf("emailAddress is required")
	}

	client, err := resolveAccount(ctx, input.AccountID)
	if err != nil {
		return nil, nil, err
	}

	path := BuildUserPermissionPath(input.AccountID, input.PermissionID)

	permInput := &UserPermissionInput{
		EmailAddress: input.EmailAddress,
	}

	if input.AccountPermission != "" {
		permInput.AccountAccess = &AccountAccess{
			Permission: input.AccountPermission,
		}
	}

	if input.ContainerAccessJSON != "" {
		var ca []ContainerAccess
		if err := json.Unmarshal([]byte(input.ContainerAccessJSON), &ca); err != nil {
			return nil, nil, fmt.Errorf("invalid containerAccessJson: %w", err)
		}
		permInput.ContainerAccess = ca
	}

	perm, err := client.UpdateUserPermission(ctx, path, permInput)
	if err != nil {
		return nil, nil, err
	}

	return nil, UpdateUserPermissionOutput{
		Success:    true,
		Permission: *perm,
		Message:    fmt.Sprintf("User permission updated for %s", input.EmailAddress),
	}, nil
}

func handleUserPermissionDelete(ctx context.Context, input UserPermissionToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteUserPermissionOutput{
			Success: false,
			Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental permission removals.",
		}, nil
	}
	if input.PermissionID == "" {
		return nil, nil, fmt.Errorf("permissionId is required for delete action")
	}

	client, err := resolveAccount(ctx, input.AccountID)
	if err != nil {
		return nil, nil, err
	}

	path := BuildUserPermissionPath(input.AccountID, input.PermissionID)
	if err := client.DeleteUserPermission(ctx, path); err != nil {
		return nil, nil, err
	}

	return nil, DeleteUserPermissionOutput{
		Success: true,
		Message: fmt.Sprintf("User permission %s deleted successfully", input.PermissionID),
	}, nil
}
