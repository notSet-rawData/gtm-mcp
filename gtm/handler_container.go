package gtm

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

// ContainerToolInput is the unified input for the container tool.
type ContainerToolInput struct {
	Action    string `json:"action" jsonschema:"enum:list,create,delete,description:Operation to perform on containers"`
	AccountID string `json:"accountId" jsonschema:"description:The GTM account ID"`
	// Fields for list: only accountId needed
	// Fields for create:
	ContainerID       string   `json:"containerId,omitempty" jsonschema:"description:The GTM container ID (required for delete)"`
	Name              string   `json:"name,omitempty" jsonschema:"description:Container display name (required for create)"`
	UsageContext      []string `json:"usageContext,omitempty" jsonschema:"description:Usage context: web, android, ios, amp, server (required for create)"`
	Notes             string   `json:"notes,omitempty" jsonschema:"description:Container notes (optional, for create)"`
	DomainName        []string `json:"domainName,omitempty" jsonschema:"description:Domain names for the container (optional, for create)"`
	TaggingServerUrls []string `json:"taggingServerUrls,omitempty" jsonschema:"description:Server-side container URLs (optional, for server containers)"`
	// Fields for delete:
	Confirm bool `json:"confirm,omitempty" jsonschema:"description:Must be true to confirm deletion (required for delete)"`
}


func handleListContainers(ctx context.Context, input ContainerToolInput) (*mcp.CallToolResult, any, error) {
	if input.AccountID == "" {
		return nil, nil, fmt.Errorf("accountId is required")
	}

	client, err := resolveAccount(ctx, input.AccountID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	containers, err := client.ListContainers(tCtx, input.AccountID)
	if err != nil {
		return nil, nil, err
	}

	return nil, ListContainersOutput{Containers: containers}, nil
}

func handleCreateContainer(ctx context.Context, input ContainerToolInput) (*mcp.CallToolResult, any, error) {
	if input.AccountID == "" {
		return nil, nil, fmt.Errorf("accountId is required")
	}
	if input.Name == "" {
		return nil, nil, fmt.Errorf("name is required")
	}
	if len(input.UsageContext) == 0 {
		return nil, nil, fmt.Errorf("usageContext is required (valid values: web, android, ios, amp, server)")
	}
	validContexts := map[string]bool{"web": true, "android": true, "ios": true, "androidSdk5": true, "iosSdk5": true, "amp": true, "server": true}
	for _, uc := range input.UsageContext {
		if !validContexts[uc] {
			return nil, nil, fmt.Errorf("invalid usageContext '%s' (valid values: web, android, ios, amp, server)", uc)
		}
	}

	client, err := resolveAccount(ctx, input.AccountID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parent := fmt.Sprintf("accounts/%s", input.AccountID)
	container := &tagmanager.Container{
		Name:         input.Name,
		UsageContext: input.UsageContext,
		Notes:        input.Notes,
		DomainName:   input.DomainName,
	}

	if len(input.TaggingServerUrls) > 0 {
		container.TaggingServerUrls = input.TaggingServerUrls
	}

	created, err := client.Service.Accounts.Containers.Create(parent, container).Context(tCtx).Do()
	if err != nil {
		return nil, nil, mapGoogleError(err)
	}

	return nil, CreateContainerOutput{
		Success: true,
		Container: CreatedContainer{
			ContainerID:   created.ContainerId,
			Name:          created.Name,
			PublicID:      created.PublicId,
			UsageContext:  created.UsageContext,
			Path:          created.Path,
			TagManagerUrl: created.TagManagerUrl,
		},
		Message: "Container created successfully",
	}, nil
}

func handleDeleteContainer(ctx context.Context, input ContainerToolInput) (*mcp.CallToolResult, any, error) {
	if !input.Confirm {
		return nil, DeleteContainerOutput{
			Success: false,
			Message: "Deletion requires confirm: true. WARNING: This will permanently delete the container and all its contents (tags, triggers, variables, versions).",
		}, nil
	}

	if input.AccountID == "" || input.ContainerID == "" {
		return nil, nil, fmt.Errorf("accountId and containerId are required")
	}

	cc, err := resolveContainer(ctx, input.AccountID, input.ContainerID)
	if err != nil {
		return nil, nil, err
	}

	tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	path := cc.ContainerPath()
	if err := cc.Client.DeleteContainer(tCtx, path); err != nil {
		return nil, nil, err
	}

	return nil, DeleteContainerOutput{
		Success: true,
		Message: fmt.Sprintf("Container %s deleted successfully", input.ContainerID),
	}, nil
}
