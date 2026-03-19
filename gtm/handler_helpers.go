package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// resolveParameters returns parameters from either direct array or JSON string.
// Priority: direct array > JSON string > nil.
// This supports both formats that Claude might send:
//   - "parameter": [{...}] (array, preferred)
//   - "parametersJson": "[{...}]" (JSON string, backward compat)
func resolveParameters(direct []Parameter, jsonStr string) ([]Parameter, error) {
	if len(direct) > 0 {
		return direct, nil
	}
	if jsonStr != "" {
		var params []Parameter
		if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
			return nil, fmt.Errorf("invalid parametersJson: %w", err)
		}
		return params, nil
	}
	return nil, nil
}

// WorkspaceContext holds a validated workspace path and an authenticated GTM client.
type WorkspaceContext struct {
	Client      *Client
	AccountID   string
	ContainerID string
	WorkspaceID string
}

// ContainerContext holds a validated container path and an authenticated GTM client.
type ContainerContext struct {
	Client      *Client
	AccountID   string
	ContainerID string
}

// resolveWorkspace validates the workspace path IDs and creates an authenticated GTM client.
// Use this in any tool handler that operates at the workspace level.
func resolveWorkspace(ctx context.Context, accountID, containerID, workspaceID string) (*WorkspaceContext, error) {
	if err := ValidateWorkspacePath(accountID, containerID, workspaceID); err != nil {
		return nil, err
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	return &WorkspaceContext{
		Client:      client,
		AccountID:   accountID,
		ContainerID: containerID,
		WorkspaceID: workspaceID,
	}, nil
}

// resolveContainer validates the container path IDs and creates an authenticated GTM client.
// Use this in any tool handler that operates at the container level.
func resolveContainer(ctx context.Context, accountID, containerID string) (*ContainerContext, error) {
	if err := ValidateContainerPath(accountID, containerID); err != nil {
		return nil, err
	}

	client, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	return &ContainerContext{
		Client:      client,
		AccountID:   accountID,
		ContainerID: containerID,
	}, nil
}

// resolveAccount validates the account ID and creates an authenticated GTM client.
// Use this in any tool handler that operates at the account level.
func resolveAccount(ctx context.Context, accountID string) (*Client, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	return getClient(ctx)
}

// WorkspacePath returns the formatted workspace path string.
func (wc *WorkspaceContext) WorkspacePath() string {
	return BuildWorkspacePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID)
}

// ContainerPath returns the formatted container path string.
func (cc *ContainerContext) ContainerPath() string {
	return BuildContainerPath(cc.AccountID, cc.ContainerID)
}

// toolConfig holds optional configuration for tool registration.
type toolConfig struct {
	cached     bool // if true, cache the response for read-only operations
	invalidate bool // if true, invalidate workspace cache after write operations
}

// ToolOption modifies tool behavior during registration.
type ToolOption func(*toolConfig)

// WithCache marks a tool as cacheable (for read-only operations).
// Responses will be cached for 30s with workspace-scoped keys.
func WithCache() ToolOption {
	return func(c *toolConfig) { c.cached = true }
}

// WithCacheInvalidation marks a tool that should invalidate the workspace cache
// after execution (for write operations like create/update/delete).
func WithCacheInvalidation() ToolOption {
	return func(c *toolConfig) { c.invalidate = true }
}

// WorkspaceToolHandler is a simplified handler that already has a WorkspaceContext and timeout applied
type WorkspaceToolHandler[I any, O any] func(ctx context.Context, wc *WorkspaceContext, input I) (O, error)

// ContainerToolHandler is a simplified handler that already has a ContainerContext and timeout applied
type ContainerToolHandler[I any, O any] func(ctx context.Context, cc *ContainerContext, input I) (O, error)

// AccountToolHandler is a simplified handler that already has an Account Client and timeout applied
type AccountToolHandler[I any, O any] func(ctx context.Context, client *Client, input I) (O, error)

// RegisterWorkspaceTool creates a tool that extracts Account, Container, and Workspace IDs from the input,
// resolves the WorkspaceContext, applies a 30s timeout, and calls the handler.
// Use WithCache() for read-only tools and WithCacheInvalidation() for write tools.
func RegisterWorkspaceTool[I any, O any](
	server *mcp.Server,
	name string,
	description string,
	extractIDs func(I) (accountID, containerID, workspaceID string),
	handler WorkspaceToolHandler[I, O],
	opts ...ToolOption,
) {
	cfg := &toolConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	mcpHandler := func(ctx context.Context, req *mcp.CallToolRequest, input I) (*mcp.CallToolResult, O, error) {
		var zero O
		aID, cID, wID := extractIDs(input)

		// Check cache for read-only tools
		if cfg.cached {
			cacheKey := WorkspaceCacheKey(aID, cID, wID, name)
			if cached, ok := globalCache.Get(cacheKey); ok {
				return nil, cached.(O), nil
			}
		}

		wc, err := resolveWorkspace(ctx, aID, cID, wID)
		if err != nil {
			return nil, zero, err
		}

		// Apply 30s timeout
		tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		out, err := handler(tCtx, wc, input)
		if err != nil {
			return nil, zero, err
		}

		// Cache the result for read-only tools
		if cfg.cached {
			cacheKey := WorkspaceCacheKey(aID, cID, wID, name)
			globalCache.Set(cacheKey, out)
		}

		// Invalidate cache for write tools
		if cfg.invalidate {
			globalCache.InvalidateWorkspace(aID, cID, wID)
		}

		return nil, out, nil
	}

	tool := &mcp.Tool{
		Name:        name,
		Description: description,
	}

	mcp.AddTool(server, tool, mcpHandler)
}

// RegisterContainerTool does the same at the Container level.
func RegisterContainerTool[I any, O any](
	server *mcp.Server,
	name string,
	description string,
	extractIDs func(I) (accountID, containerID string),
	handler ContainerToolHandler[I, O],
	opts ...ToolOption,
) {
	mcpHandler := func(ctx context.Context, req *mcp.CallToolRequest, input I) (*mcp.CallToolResult, O, error) {
		var zero O
		aID, cID := extractIDs(input)
		cc, err := resolveContainer(ctx, aID, cID)
		if err != nil {
			return nil, zero, err
		}

		tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		out, err := handler(tCtx, cc, input)
		if err != nil {
			return nil, zero, err
		}
		return nil, out, nil
	}

	tool := &mcp.Tool{
		Name:        name,
		Description: description,
	}

	mcp.AddTool(server, tool, mcpHandler)
}

// RegisterAccountTool does the same at the Account level.
func RegisterAccountTool[I any, O any](
	server *mcp.Server,
	name string,
	description string,
	extractID func(I) string,
	handler AccountToolHandler[I, O],
	opts ...ToolOption,
) {
	mcpHandler := func(ctx context.Context, req *mcp.CallToolRequest, input I) (*mcp.CallToolResult, O, error) {
		var zero O
		aID := extractID(input)
		client, err := resolveAccount(ctx, aID)
		if err != nil {
			return nil, zero, err
		}

		tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		out, err := handler(tCtx, client, input)
		if err != nil {
			return nil, zero, err
		}
		return nil, out, nil
	}

	tool := &mcp.Tool{
		Name:        name,
		Description: description,
	}

	mcp.AddTool(server, tool, mcpHandler)
}
