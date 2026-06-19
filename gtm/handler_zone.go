package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

type ZoneToolInput struct {
	Action          string `json:"action" jsonschema:"enum:list,get,create,update,delete,revert,description:Operation to perform on zones"`
	AccountID       string `json:"accountId" jsonschema:"description:The GTM account ID"`
	ContainerID     string `json:"containerId" jsonschema:"description:The GTM container ID"`
	WorkspaceID     string `json:"workspaceId" jsonschema:"description:The GTM workspace ID"`
	ZoneID          string `json:"zoneId,omitempty" jsonschema:"description:Zone ID (required for get, update, delete, revert)"`
	Name            string `json:"name,omitempty" jsonschema:"description:Zone name (required for create/update)"`
	ChildContainer  string `json:"childContainerJson,omitempty" jsonschema:"description:Child containers as JSON array (optional)"`
	BoundaryJSON    string `json:"boundaryJson,omitempty" jsonschema:"description:Zone boundary conditions as JSON array (optional)"`
	TypeRestriction string `json:"typeRestrictionJson,omitempty" jsonschema:"description:Type restriction as JSON object (optional)"`
	Notes           string `json:"notes,omitempty" jsonschema:"description:Zone notes (optional)"`
	Confirm         bool   `json:"confirm,omitempty" jsonschema:"description:Must be true for delete (safety guard)"`
	Fingerprint     string `json:"fingerprint,omitempty" jsonschema:"description:Fingerprint for optimistic concurrency control (optional for revert)"`
}

type ZoneInfo struct {
	ZoneID        string `json:"zoneId"`
	Name          string `json:"name"`
	Path          string `json:"path,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	TagManagerUrl string `json:"tagManagerUrl,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

type ListZonesOutput struct {
	Zones []ZoneInfo `json:"zones"`
}

type GetZoneOutput struct {
	Zone interface{} `json:"zone"`
}

type CreateZoneOutput struct {
	Success bool     `json:"success"`
	Zone    ZoneInfo `json:"zone"`
	Message string   `json:"message"`
}

type UpdateZoneOutput struct {
	Success bool     `json:"success"`
	Zone    ZoneInfo `json:"zone"`
	Message string   `json:"message"`
}

type DeleteZoneOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func handleZoneList(ctx context.Context, input ZoneToolInput) (*mcp.CallToolResult, any, error) {
	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	cacheKey := WorkspaceCacheKey(wc.AccountID, wc.ContainerID, wc.WorkspaceID, "zone_list")
	if cached, ok := globalCache.Get(cacheKey); ok {
		return nil, cached, nil
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := wc.WorkspacePath()
	resp, err := wc.Client.Service.Accounts.Containers.Workspaces.Zones.List(parent).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	zones := make([]ZoneInfo, 0)
	if resp.Zone != nil {
		for _, z := range resp.Zone {
			zones = append(zones, ZoneInfo{
				ZoneID:        z.ZoneId,
				Name:          z.Name,
				Path:          z.Path,
				Fingerprint:   z.Fingerprint,
				TagManagerUrl: z.TagManagerUrl,
				Notes:         z.Notes,
			})
		}
	}

	out := ListZonesOutput{Zones: zones}
	globalCache.Set(cacheKey, out)
	return nil, out, nil
}

func handleZoneGet(ctx context.Context, input ZoneToolInput) (*mcp.CallToolResult, any, error) {
	if input.ZoneID == "" {
		return nil, nil, fmt.Errorf("zoneId is required for get action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/zones/%s", wc.WorkspacePath(), input.ZoneID)
	zone, err := wc.Client.Service.Accounts.Containers.Workspaces.Zones.Get(path).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	return nil, GetZoneOutput{Zone: zone}, nil
}

func handleZoneCreate(ctx context.Context, input ZoneToolInput) (*mcp.CallToolResult, any, error) {
	if input.Name == "" {
		return nil, nil, fmt.Errorf("name is required for create action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := wc.WorkspacePath()
	zone := &tagmanager.Zone{
		Name:  input.Name,
		Notes: input.Notes,
	}

	created, err := wc.Client.Service.Accounts.Containers.Workspaces.Zones.Create(parent, zone).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, CreateZoneOutput{
		Success: true,
		Zone: ZoneInfo{
			ZoneID:        created.ZoneId,
			Name:          created.Name,
			Path:          created.Path,
			Fingerprint:   created.Fingerprint,
			TagManagerUrl: created.TagManagerUrl,
			Notes:         created.Notes,
		},
		Message: fmt.Sprintf("Zone '%s' created successfully", created.Name),
	}, nil
}

func handleZoneUpdate(ctx context.Context, input ZoneToolInput) (*mcp.CallToolResult, any, error) {
	if input.ZoneID == "" {
		return nil, nil, fmt.Errorf("zoneId is required for update action")
	}
	if input.Name == "" {
		return nil, nil, fmt.Errorf("name is required for update action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/zones/%s", wc.WorkspacePath(), input.ZoneID)
	zone := &tagmanager.Zone{
		Name:  input.Name,
		Notes: input.Notes,
	}

	updated, err := wc.Client.Service.Accounts.Containers.Workspaces.Zones.Update(path, zone).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, UpdateZoneOutput{
		Success: true,
		Zone: ZoneInfo{
			ZoneID:        updated.ZoneId,
			Name:          updated.Name,
			Path:          updated.Path,
			Fingerprint:   updated.Fingerprint,
			TagManagerUrl: updated.TagManagerUrl,
			Notes:         updated.Notes,
		},
		Message: fmt.Sprintf("Zone '%s' updated successfully", updated.Name),
	}, nil
}

func handleZoneDelete(ctx context.Context, input ZoneToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteZoneOutput{
			Success: false,
			Message: "Deletion requires confirm: true. This is a safety guard to prevent accidental deletions.",
		}, nil
	}
	if input.ZoneID == "" {
		return nil, nil, fmt.Errorf("zoneId is required for delete action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/zones/%s", wc.WorkspacePath(), input.ZoneID)
	if err := wc.Client.Service.Accounts.Containers.Workspaces.Zones.Delete(path).Context(tCtx).Do(); err != nil {
		return nil, nil, mapGoogleError(err)
	}

	globalCache.InvalidateWorkspace(wc.AccountID, wc.ContainerID, wc.WorkspaceID)

	return nil, DeleteZoneOutput{
		Success: true,
		Message: fmt.Sprintf("Zone %s deleted successfully", input.ZoneID),
	}, nil
}

func handleZoneRevert(ctx context.Context, input ZoneToolInput) (*mcp.CallToolResult, any, error) {
	if input.ZoneID == "" {
		return nil, nil, fmt.Errorf("zoneId is required for revert action")
	}

	wc, err := resolveWorkspace(ctx, input.AccountID, input.ContainerID, input.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("%s/zones/%s", wc.WorkspacePath(), input.ZoneID)
	call := wc.Client.Service.Accounts.Containers.Workspaces.Zones.Revert(path).Context(tCtx)
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
		Message: fmt.Sprintf("Zone %s reverted to latest published version", input.ZoneID),
		Entity:  resp.Zone,
	}, nil
}
