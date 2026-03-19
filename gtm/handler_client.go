package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ClientToolInput is the unified input for the client tool (server-side containers).
type ClientToolInput struct {
	Action      string `json:"action" jsonschema:"enum:list,get,create,update,delete,revert,description:Operation to perform on clients"`
	AccountID   string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	// Fields for get/update/delete:
	ClientID string `json:"clientId,omitempty" jsonschema:"description:Client ID (required for get, update, delete)"`
	// Fields for create/update:
	Name           string `json:"name,omitempty" jsonschema:"description:Client name (required for create/update)"`
	Type           string `json:"type,omitempty" jsonschema:"description:Client type e.g. __ga4 (GA4), __googtag (Google tag) (required for create/update)"`
	Priority       int64  `json:"priority,omitempty" jsonschema:"description:Client priority (optional, higher runs first)"`
	ParametersJSON string `json:"parametersJson,omitempty" jsonschema:"description:Client parameters as JSON array (optional)"`
	Notes          string `json:"notes,omitempty" jsonschema:"description:Client notes (optional)"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
}


func handleClientList(ctx context.Context, input ClientToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "client_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	clients, err := wc.Client.ListClients(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	out := ListClientsOutput{Clients: clients}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleClientGet(ctx context.Context, input ClientToolInput) (*mcp.CallToolResult, any, error) {
	if input.ClientID == "" {
		return nil, nil, fmt.Errorf("clientId is required for get action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cl, err := wc.Client.GetClient(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ClientID)
	if err != nil {
		return nil, nil, err
	}

	return nil, GetClientOutput{Client: *cl}, nil
}

func handleClientCreate(ctx context.Context, input ClientToolInput) (*mcp.CallToolResult, any, error) {
	if err := ValidateClientInput(input.Name, input.Type); err != nil {
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

	clientInput := &ClientInput{
		Name:      input.Name,
		Type:      input.Type,
		Priority:  input.Priority,
		Parameter: params,
		Notes:     input.Notes,
	}

	cl, err := wc.Client.CreateClient(tCtx, wc.AccountID, wc.ContainerID, wc.WorkspaceID, clientInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, CreateClientOutput{
		Success: true,
		Client:  *cl,
		Message: "Client created successfully",
	}, nil
}

func handleClientUpdate(ctx context.Context, input ClientToolInput) (*mcp.CallToolResult, any, error) {
	if input.ClientID == "" {
		return nil, nil, fmt.Errorf("clientId is required for update action")
	}
	if err := ValidateClientInput(input.Name, input.Type); err != nil {
		return nil, nil, err
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildClientPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ClientID)

	var params []Parameter
	if input.ParametersJSON != "" {
		if err := json.Unmarshal([]byte(input.ParametersJSON), &params); err != nil {
			return nil, nil, fmt.Errorf("invalid parametersJson: %w", err)
		}
	}

	clientInput := &ClientInput{
		Name:      input.Name,
		Type:      input.Type,
		Priority:  input.Priority,
		Parameter: params,
		Notes:     input.Notes,
	}

	cl, err := wc.Client.UpdateClient(tCtx, path, clientInput)
	if err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, UpdateClientOutput{
		Success: true,
		Client:  *cl,
		Message: "Client updated successfully",
	}, nil
}

func handleClientDelete(ctx context.Context, input ClientToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteClientOutput{
			Success: false,
			Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		}, nil
	}
	if input.ClientID == "" {
		return nil, nil, fmt.Errorf("clientId is required for delete action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildClientPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ClientID)
	if err := wc.Client.DeleteClient(tCtx, path); err != nil {
		return nil, nil, err
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DeleteClientOutput{
		Success: true,
		Message: fmt.Sprintf("Client %s deleted successfully", input.ClientID),
	}, nil
}

func handleClientRevert(ctx context.Context, input ClientToolInput) (*mcp.CallToolResult, any, error) {
	if input.ClientID == "" {
		return nil, nil, fmt.Errorf("clientId is required for revert action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := BuildClientPath(wc.AccountID, wc.ContainerID, wc.WorkspaceID, input.ClientID)
	call := wc.Client.Service.Accounts.Containers.Workspaces.Clients.Revert(path).Context(tCtx)
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
		Message: fmt.Sprintf("Client %s reverted to latest published version", input.ClientID),
		Entity:  resp.Client,
	}, nil
}
