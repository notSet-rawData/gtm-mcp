package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// ContextKey is the type for context keys.
type ContextKey string

const (
	// TokenInfoKey is the context key for TokenInfo.
	TokenInfoKey ContextKey = "token_info"
	// GoogleTokenKey is the context key for the Google OAuth token.
	GoogleTokenKey ContextKey = "google_token"
	// TokenStoreKey is the context key for the token store.
	TokenStoreKey ContextKey = "token_store"
	// GoogleProviderKey is the context key for the Google OAuth provider.
	GoogleProviderKey ContextKey = "google_provider"
)

// Middleware creates HTTP middleware that validates bearer tokens.
// If a token is expired but has a valid refresh token, it will automatically
// refresh the token and continue the request transparently.
// If resolver is non-nil, 401 responses will use dynamically resolved URLs.
func Middleware(store TokenStore, google *GoogleProvider, logger *slog.Logger, baseURL string, accessTokenTTL time.Duration, resolver *URLResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Resolve the base URL for error responses
			effectiveURL := baseURL
			if resolver != nil {
				effectiveURL = resolver.Resolve(r)
			}

			// Extract bearer token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn("auth_failed", "reason", "missing_header")
				unauthorized(w, effectiveURL, "Missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				logger.Warn("auth_failed", "reason", "invalid_format")
				unauthorized(w, effectiveURL, "Invalid authorization header format")
				return
			}

			accessToken := parts[1]

			// Look up the token
			tokenInfo, err := store.GetTokenByAccess(accessToken)
			if err != nil {
				if err == ErrTokenExpired {
					// Attempt auto-refresh
					tokenInfo, err = tryAutoRefresh(r.Context(), store, google, logger, accessToken, baseURL, accessTokenTTL)
					if err != nil {
						unauthorized(w, effectiveURL, err.Error())
						return
					}
				} else {
					logger.Warn("auth_failed", "reason", "token_not_found", "token_prefix", truncateToken(accessToken))
					unauthorized(w, effectiveURL, "Invalid token")
					return
				}
			}

			// Add token info and dependencies to context
			ctx := context.WithValue(r.Context(), TokenInfoKey, tokenInfo)
			ctx = context.WithValue(ctx, GoogleTokenKey, tokenInfo.GoogleToken)
			ctx = context.WithValue(ctx, TokenStoreKey, store)
			ctx = context.WithValue(ctx, GoogleProviderKey, google)

			logger.Debug("authenticated request",
				"client_id", tokenInfo.ClientID,
			)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// tryAutoRefresh attempts to refresh an expired token transparently.
func tryAutoRefresh(ctx context.Context, store TokenStore, google *GoogleProvider, logger *slog.Logger, accessToken string, baseURL string, accessTokenTTL time.Duration) (*TokenInfo, error) {
	logger.Info("auth_token_expired", "token_prefix", truncateToken(accessToken), "action", "auto_refresh")

	// Get the expired token info (including refresh token)
	expiredToken, err := store.GetTokenByAccessIncludeExpired(accessToken)
	if err != nil {
		logger.Warn("auth_auto_refresh_failed", "reason", "token_not_found", "error", err)
		return nil, fmt.Errorf("Token expired")
	}

	// Check refresh token exists
	if expiredToken.RefreshToken == "" || expiredToken.GoogleToken == nil || expiredToken.GoogleToken.RefreshToken == "" {
		logger.Warn("auth_auto_refresh_failed", "reason", "no_refresh_token", "client_id", expiredToken.ClientID)
		return nil, fmt.Errorf("Token expired, no refresh token available")
	}

	// Check refresh token hasn't expired
	if !expiredToken.RefreshExpiresAt.IsZero() && time.Now().After(expiredToken.RefreshExpiresAt) {
		logger.Warn("auth_auto_refresh_failed", "reason", "refresh_token_expired", "client_id", expiredToken.ClientID)
		return nil, fmt.Errorf("Token expired, refresh token also expired")
	}

	// Refresh the Google token
	newGoogleToken, err := google.RefreshToken(ctx, expiredToken.GoogleToken.RefreshToken)
	if err != nil {
		logger.Warn("auth_auto_refresh_failed", "reason", "google_refresh_failed", "client_id", expiredToken.ClientID, "error", err)
		return nil, fmt.Errorf("Token expired, failed to refresh")
	}

	// Generate new tokens (rotation)
	newAccessToken, err := GenerateToken(32)
	if err != nil {
		logger.Warn("auth_auto_refresh_failed", "reason", "token_generation_failed", "error", err)
		return nil, fmt.Errorf("Token expired")
	}

	newRefreshToken, err := GenerateToken(32)
	if err != nil {
		logger.Warn("auth_auto_refresh_failed", "reason", "token_generation_failed", "error", err)
		return nil, fmt.Errorf("Token expired")
	}

	// Atomically rotate: store new then delete old in one operation
	newTokenInfo := &TokenInfo{
		AccessToken:      newAccessToken,
		RefreshToken:     newRefreshToken,
		ExpiresAt:        time.Now().Add(accessTokenTTL),
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		GoogleToken:      newGoogleToken,
		ClientID:         expiredToken.ClientID,
		CreatedAt:        time.Now(),
	}

	if err := store.RotateToken(accessToken, newTokenInfo); err != nil {
		logger.Warn("auth_auto_refresh_failed", "reason", "rotate_failed", "error", err)
		return nil, fmt.Errorf("Token expired")
	}

	logger.Info("auth_auto_refresh_success",
		"client_id", expiredToken.ClientID,
		"new_expiry", newTokenInfo.ExpiresAt,
	)

	return newTokenInfo, nil
}

// OptionalMiddleware allows unauthenticated requests but adds token info if present.
func OptionalMiddleware(store TokenStore, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				next.ServeHTTP(w, r)
				return
			}

			accessToken := parts[1]
			tokenInfo, err := store.GetTokenByAccess(accessToken)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), TokenInfoKey, tokenInfo)
			ctx = context.WithValue(ctx, GoogleTokenKey, tokenInfo.GoogleToken)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTokenInfo retrieves TokenInfo from context.
func GetTokenInfo(ctx context.Context) *TokenInfo {
	if info, ok := ctx.Value(TokenInfoKey).(*TokenInfo); ok {
		return info
	}
	return nil
}

// GetGoogleToken retrieves the Google OAuth token from context.
func GetGoogleToken(ctx context.Context) *oauth2.Token {
	if token, ok := ctx.Value(GoogleTokenKey).(*oauth2.Token); ok {
		return token
	}
	return nil
}

// GetTokenStore retrieves the TokenStore from context.
func GetTokenStore(ctx context.Context) TokenStore {
	if store, ok := ctx.Value(TokenStoreKey).(TokenStore); ok {
		return store
	}
	return nil
}

// GetGoogleProvider retrieves the GoogleProvider from context.
func GetGoogleProvider(ctx context.Context) *GoogleProvider {
	if provider, ok := ctx.Value(GoogleProviderKey).(*GoogleProvider); ok {
		return provider
	}
	return nil
}

// unauthorized sends a 401 response with WWW-Authenticate header per RFC 9728
func unauthorized(w http.ResponseWriter, baseURL, message string) {
	// Build WWW-Authenticate header with resource_metadata per RFC 9728
	resourceMetadataURL := baseURL + "/.well-known/oauth-protected-resource"

	authHeader := fmt.Sprintf(`Bearer resource_metadata="%s"`, resourceMetadataURL)

	w.Header().Set("WWW-Authenticate", authHeader)
	w.Header().Set("Content-Type", "application/json")

	if strings.Contains(strings.ToLower(message), "expired") {
		w.Header().Set("Retry-After", "0")
	}

	w.WriteHeader(http.StatusUnauthorized)

	resp := map[string]string{
		"error":                  "unauthorized",
		"error_description":      message,
		"authorization_endpoint": baseURL + "/authorize",
		"token_endpoint":         baseURL + "/token",
	}
	json.NewEncoder(w).Encode(resp)
}

// truncateToken returns the first 8 characters of a token for safe logging.
func truncateToken(token string) string {
	if len(token) <= 8 {
		return token + "..."
	}
	return token[:8] + "..."
}
