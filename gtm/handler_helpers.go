package gtm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

type WorkspaceContext struct {
	Client      *Client
	AccountID   string
	ContainerID string
	WorkspaceID string
}

type ContainerContext struct {
	Client      *Client
	AccountID   string
	ContainerID string
}

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

func resolveAccount(ctx context.Context, accountID string) (*Client, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account ID is required")
	}

	return getClient(ctx)
}

func (wc *WorkspaceContext) WorkspacePath() string {
	return BuildWorkspacePath(wc.AccountID, wc.ContainerID, wc.WorkspaceID)
}

func (cc *ContainerContext) ContainerPath() string {
	return BuildContainerPath(cc.AccountID, cc.ContainerID)
}

type toolConfig struct {
	cached     bool // if true, cache the response for read-only operations
	invalidate bool // if true, invalidate workspace cache after write operations
}

type ToolOption func(*toolConfig)

func WithCache() ToolOption {
	return func(c *toolConfig) { c.cached = true }
}

func WithCacheInvalidation() ToolOption {
	return func(c *toolConfig) { c.invalidate = true }
}

type WorkspaceToolHandler[I any, O any] func(ctx context.Context, wc *WorkspaceContext, input I) (O, error)

type ContainerToolHandler[I any, O any] func(ctx context.Context, cc *ContainerContext, input I) (O, error)

type AccountToolHandler[I any, O any] func(ctx context.Context, client *Client, input I) (O, error)

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

		tCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		out, err := handler(tCtx, wc, input)
		if err != nil {
			return nil, zero, err
		}

		if cfg.cached {
			cacheKey := WorkspaceCacheKey(aID, cID, wID, name)
			globalCache.Set(cacheKey, out)
		}

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
