package auth

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleProvider struct {
	config *oauth2.Config
}

var GoogleScopes = []string{
	"https://www.googleapis.com/auth/tagmanager.readonly",
	"https://www.googleapis.com/auth/tagmanager.delete.containers",
	"https://www.googleapis.com/auth/tagmanager.edit.containers",
	"https://www.googleapis.com/auth/tagmanager.edit.containerversions",
	"https://www.googleapis.com/auth/tagmanager.publish",
	"https://www.googleapis.com/auth/tagmanager.manage.users",
}

func NewGoogleProvider(clientID, clientSecret, redirectURI string, customScopes ...string) *GoogleProvider {
	scopes := GoogleScopes
	if len(customScopes) > 0 {
		scopes = customScopes
	}
	return &GoogleProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Scopes:       scopes,
			Endpoint:     google.Endpoint,
		},
	}
}

func (p *GoogleProvider) AuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	opts = append(opts, oauth2.AccessTypeOffline)
	opts = append(opts, oauth2.ApprovalForce)

	return p.config.AuthCodeURL(state, opts...)
}

func (p *GoogleProvider) Exchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	token, err := p.config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	return token, nil
}

func (p *GoogleProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	tokenSource := p.config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	token, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return token, nil
}

func (p *GoogleProvider) Client(ctx context.Context, token *oauth2.Token) *oauth2.Config {
	return p.config
}

func (p *GoogleProvider) Config() *oauth2.Config {
	return p.config
}

type AutoRefreshTokenSource struct {
	mu          sync.Mutex
	store       TokenStore
	accessToken string // Our token (to identify the record in store)
	config      *oauth2.Config
	current     *oauth2.Token
}

func NewAutoRefreshTokenSource(store TokenStore, accessToken string, config *oauth2.Config, token *oauth2.Token) *AutoRefreshTokenSource {
	return &AutoRefreshTokenSource{
		store:       store,
		accessToken: accessToken,
		config:      config,
		current:     token,
	}
}

func (s *AutoRefreshTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current.Valid() {
		return s.current, nil
	}

	slog.Info("Google token expired, refreshing...")

	tokenSource := s.config.TokenSource(context.Background(), s.current)
	newToken, err := tokenSource.Token()
	if err != nil {
		slog.Error("Failed to refresh Google token", "error", err)
		return nil, fmt.Errorf("failed to refresh Google token: %w", err)
	}

	slog.Info("Google token refreshed successfully", "new_expiry", newToken.Expiry)

	s.current = newToken

	if s.store != nil && s.accessToken != "" {
		if err := s.store.UpdateGoogleToken(s.accessToken, newToken); err != nil {
			slog.Warn("failed to update Google token in store", "error", err)
		}
	}

	return newToken, nil
}
