package gtm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gtm-mcp-server/auth"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterTools(server *mcp.Server) {
	registerGateway(server)

	RegisterResources(server)

	RegisterPrompts(server)
}

func getTokenInfo(ctx context.Context) interface{} {
	return auth.GetTokenInfo(ctx)
}

type clientEntry struct {
	client    *Client
	expiresAt time.Time
}

var clientPool sync.Map

func init() {
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			now := time.Now()
			clientPool.Range(func(key, value any) bool {
				if entry, ok := value.(clientEntry); ok {
					if now.After(entry.expiresAt) {
						clientPool.Delete(key)
					}
				}
				return true
			})
		}
	}()
}

func getClient(ctx context.Context) (*Client, error) {
	tokenInfo := auth.GetTokenInfo(ctx)
	if tokenInfo == nil || tokenInfo.GoogleToken == nil {
		return nil, fmt.Errorf("not authenticated - please authenticate with Google first")
	}

	if value, ok := clientPool.Load(tokenInfo.AccessToken); ok {
		if entry, ok := value.(clientEntry); ok && time.Now().Before(entry.expiresAt) {
			return entry.client, nil
		}
		clientPool.Delete(tokenInfo.AccessToken)
	}

	store := auth.GetTokenStore(ctx)
	google := auth.GetGoogleProvider(ctx)

	var tokenSource = auth.NewAutoRefreshTokenSource(
		store,
		tokenInfo.AccessToken,
		google.Config(),
		tokenInfo.GoogleToken,
	)

	client, err := NewClient(ctx, tokenSource, slog.Default())
	if err != nil {
		return nil, err
	}

	clientPool.Store(tokenInfo.AccessToken, clientEntry{
		client:    client,
		expiresAt: tokenInfo.ExpiresAt,
	})

	return client, nil
}
