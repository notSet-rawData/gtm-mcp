package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gtm-mcp-server/auth/serviceauth"

	"golang.org/x/oauth2"
)

type ContextKey string

const (
	TokenInfoKey      ContextKey = "token_info"
	GoogleTokenKey    ContextKey = "google_token"
	TokenStoreKey     ContextKey = "token_store"
	GoogleProviderKey ContextKey = "google_provider"
	AuthMethodKey     ContextKey = "auth_method"
	SATokenInfoKey    ContextKey = "sa_token_info"
)

func Middleware(
	store TokenStore,
	google *GoogleProvider,
	saProvider *serviceauth.Provider,
	saValidator *serviceauth.Validator,
	logger *slog.Logger,
	baseURL string,
	accessTokenTTL time.Duration,
	resolver *URLResolver,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			effectiveURL := baseURL
			if resolver != nil {
				effectiveURL = resolver.Resolve(r)
			}

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

			if tokenInfo, err := tryOAuthToken(r.Context(), store, google, logger, accessToken, baseURL, accessTokenTTL); err == nil {
				ctx := context.WithValue(r.Context(), TokenInfoKey, tokenInfo)
				ctx = context.WithValue(ctx, GoogleTokenKey, tokenInfo.GoogleToken)
				ctx = context.WithValue(ctx, TokenStoreKey, store)
				ctx = context.WithValue(ctx, GoogleProviderKey, google)
				ctx = context.WithValue(ctx, AuthMethodKey, AuthMethodOAuth)

				logger.Debug("authenticated request",
					"method", string(AuthMethodOAuth),
					"client_id", tokenInfo.ClientID,
				)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if saProvider != nil && saValidator != nil && isLikelyJWT(accessToken) {
				if saInfo, googleToken, err := tryServiceAccountToken(r.Context(), saProvider, saValidator, logger, accessToken); err == nil {
					ctx := context.WithValue(r.Context(), GoogleTokenKey, googleToken)
					ctx = context.WithValue(ctx, AuthMethodKey, AuthMethodServiceAccount)
					ctx = context.WithValue(ctx, SATokenInfoKey, saInfo)
					ctx = context.WithValue(ctx, TokenStoreKey, store)
					ctx = context.WithValue(ctx, GoogleProviderKey, google)

					logger.Info("authenticated request",
						"method", string(AuthMethodServiceAccount),
						"service_account", saInfo.ServiceAccountEmail,
						"mode", string(saInfo.Mode),
					)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			logger.Warn("auth_failed",
				"reason", "invalid_token",
				"token_prefix", truncateToken(accessToken),
			)
			unauthorized(w, effectiveURL, "Invalid token")
		})
	}
}

func tryAutoRefresh(ctx context.Context, store TokenStore, google *GoogleProvider, logger *slog.Logger, accessToken string, baseURL string, accessTokenTTL time.Duration) (*TokenInfo, error) {
	logger.Info("auth_token_expired", "token_prefix", truncateToken(accessToken), "action", "auto_refresh")

	expiredToken, err := store.GetTokenByAccessIncludeExpired(accessToken)
	if err != nil {
		logger.Warn("auth_auto_refresh_failed", "reason", "token_not_found", "error", err)
		return nil, fmt.Errorf("Token expired")
	}

	if expiredToken.RefreshToken == "" || expiredToken.GoogleToken == nil || expiredToken.GoogleToken.RefreshToken == "" {
		logger.Warn("auth_auto_refresh_failed", "reason", "no_refresh_token", "client_id", expiredToken.ClientID)
		return nil, fmt.Errorf("Token expired, no refresh token available")
	}

	if !expiredToken.RefreshExpiresAt.IsZero() && time.Now().After(expiredToken.RefreshExpiresAt) {
		logger.Warn("auth_auto_refresh_failed", "reason", "refresh_token_expired", "client_id", expiredToken.ClientID)
		return nil, fmt.Errorf("Token expired, refresh token also expired")
	}

	newGoogleToken, err := google.RefreshToken(ctx, expiredToken.GoogleToken.RefreshToken)
	if err != nil {
		logger.Warn("auth_auto_refresh_failed", "reason", "google_refresh_failed", "client_id", expiredToken.ClientID, "error", err)
		return nil, fmt.Errorf("Token expired, failed to refresh")
	}

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

func tryOAuthToken(ctx context.Context, store TokenStore, google *GoogleProvider, logger *slog.Logger, accessToken string, baseURL string, accessTokenTTL time.Duration) (*TokenInfo, error) {
	tokenInfo, err := store.GetTokenByAccess(accessToken)
	if err == nil {
		return tokenInfo, nil
	}
	if err == ErrTokenExpired {
		return tryAutoRefresh(ctx, store, google, logger, accessToken, baseURL, accessTokenTTL)
	}
	return nil, err
}

func tryServiceAccountToken(
	ctx context.Context,
	provider *serviceauth.Provider,
	validator *serviceauth.Validator,
	logger *slog.Logger,
	tokenStr string,
) (*serviceauth.SATokenInfo, *oauth2.Token, error) {
	claims, err := validator.ValidateJWT(ctx, tokenStr)
	if err != nil {
		logger.Debug("sa_jwt_validation_failed", "reason", err.Error())
		return nil, nil, err
	}

	googleToken, err := provider.GoogleToken(ctx)
	if err != nil {
		logger.Warn("sa_google_token_failed",
			"service_account", provider.Email(),
			"error", err,
		)
		return nil, nil, fmt.Errorf("service account token fetch failed: %w", err)
	}

	saInfo := &serviceauth.SATokenInfo{
		ServiceAccountEmail: claims.Issuer,
		Subject:             claims.Subject,
		Mode:                provider.Mode(),
		Fingerprint:         provider.FingerprintKeyID(),
	}

	return saInfo, googleToken, nil
}

func isLikelyJWT(s string) bool {
	if len(s) < 10 {
		return false
	}
	parts := strings.Split(s, ".")
	return len(parts) == 3 && len(parts[0]) > 0 && len(parts[1]) > 0 && len(parts[2]) > 0
}

func GetAuthMethod(ctx context.Context) AuthMethod {
	if method, ok := ctx.Value(AuthMethodKey).(AuthMethod); ok {
		return method
	}
	return ""
}

func GetSATokenInfo(ctx context.Context) *serviceauth.SATokenInfo {
	if info, ok := ctx.Value(SATokenInfoKey).(*serviceauth.SATokenInfo); ok {
		return info
	}
	return nil
}

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

func GetTokenInfo(ctx context.Context) *TokenInfo {
	if info, ok := ctx.Value(TokenInfoKey).(*TokenInfo); ok {
		return info
	}
	return nil
}

func GetGoogleToken(ctx context.Context) *oauth2.Token {
	if token, ok := ctx.Value(GoogleTokenKey).(*oauth2.Token); ok {
		return token
	}
	return nil
}

func GetTokenStore(ctx context.Context) TokenStore {
	if store, ok := ctx.Value(TokenStoreKey).(TokenStore); ok {
		return store
	}
	return nil
}

func GetGoogleProvider(ctx context.Context) *GoogleProvider {
	if provider, ok := ctx.Value(GoogleProviderKey).(*GoogleProvider); ok {
		return provider
	}
	return nil
}

func unauthorized(w http.ResponseWriter, baseURL, message string) {
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

func truncateToken(token string) string {
	if len(token) <= 8 {
		return token + "..."
	}
	return token[:8] + "..."
}
