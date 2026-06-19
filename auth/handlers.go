package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gtm-mcp-server/auth/serviceauth"
)

type Server struct {
	baseURL           string
	google            *GoogleProvider
	store             TokenStore
	logger            *slog.Logger
	accessTokenTTL    time.Duration
	allowedDCRDomains map[string]bool
	saProvider        *serviceauth.Provider
	saValidator       *serviceauth.Validator
}

func NewServer(baseURL string, google *GoogleProvider, store TokenStore, logger *slog.Logger, accessTokenTTL time.Duration, allowedDCRDomains ...string) *Server {
	dcrDomains := make(map[string]bool)
	for _, d := range allowedDCRDomains {
		dcrDomains[d] = true
	}
	return &Server{
		baseURL:           baseURL,
		google:            google,
		store:             store,
		logger:            logger,
		accessTokenTTL:    accessTokenTTL,
		allowedDCRDomains: dcrDomains,
	}
}

func (s *Server) WithServiceAccount(provider *serviceauth.Provider, validator *serviceauth.Validator) *Server {
	s.saProvider = provider
	s.saValidator = validator
	return s
}

func (s *Server) AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	responseType := r.URL.Query().Get("response_type")
	state := r.URL.Query().Get("state")
	codeChallenge := r.URL.Query().Get("code_challenge")
	codeChallengeMethod := r.URL.Query().Get("code_challenge_method")
	resource := r.URL.Query().Get("resource") // RFC 9728: resource indicator

	if responseType != "code" {
		s.errorResponse(w, "unsupported_response_type", "Only 'code' response type is supported")
		return
	}

	if state == "" {
		s.errorResponse(w, "invalid_request", "State parameter is required")
		return
	}

	if clientID != "" {
		if client, err := s.store.GetClient(clientID); err == nil {
			validRedirect := false
			for _, uri := range client.RedirectURIs {
				if uri == redirectURI {
					validRedirect = true
					break
				}
			}
			if !validRedirect {
				s.errorResponse(w, "invalid_request", "redirect_uri does not match registered URIs")
				return
			}
		} else {
			if !isValidRedirectURI(redirectURI) {
				s.errorResponse(w, "invalid_request", "Invalid redirect_uri")
				return
			}
		}
	} else if !isValidRedirectURI(redirectURI) {
		s.errorResponse(w, "invalid_request", "Invalid redirect_uri")
		return
	}

	if codeChallenge == "" || codeChallengeMethod != "S256" {
		s.errorResponse(w, "invalid_request", "PKCE with S256 is required")
		return
	}

	googleState, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate state", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	authState := &AuthState{
		State:        googleState,
		ClientState:  state,
		CodeVerifier: codeChallenge, // Store the challenge, we'll verify later
		RedirectURI:  redirectURI,
		ClientID:     clientID,
		Resource:     resource, // Store resource for audience binding
		CreatedAt:    time.Now(),
	}

	if err := s.store.StoreState(authState); err != nil {
		s.logger.Error("failed to store state", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	googleAuthURL := s.google.AuthCodeURL(authState.State)

	s.logger.Info("redirecting to Google OAuth",
		"client_id", clientID,
		"redirect_uri", redirectURI,
	)

	http.Redirect(w, r, googleAuthURL, http.StatusFound)
}

func (s *Server) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if errCode := r.URL.Query().Get("error"); errCode != "" {
		errDesc := r.URL.Query().Get("error_description")
		s.logger.Error("Google OAuth error", "error", errCode, "description", errDesc)
		s.errorResponse(w, errCode, errDesc)
		return
	}

	code := r.URL.Query().Get("code")
	googleState := r.URL.Query().Get("state")

	if code == "" || googleState == "" {
		s.errorResponse(w, "invalid_request", "Missing code or state")
		return
	}

	authState, err := s.store.GetState(googleState)
	if err != nil {
		s.logger.Error("failed to get state", "error", err)
		s.errorResponse(w, "invalid_request", "Invalid or expired state")
		return
	}

	claudeState := authState.ClientState

	_ = s.store.DeleteState(googleState)

	googleToken, err := s.google.Exchange(r.Context(), code)
	if err != nil {
		s.logger.Error("failed to exchange code with Google", "error", err)
		s.errorResponse(w, "server_error", "Failed to exchange authorization code")
		return
	}

	ourCode, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate code", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	tempToken := &TokenInfo{
		AccessToken: ourCode, // Temporary: using code as key
		GoogleToken: googleToken,
		ClientID:    authState.ClientID,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(5 * time.Minute), // Code expires in 5 min
	}

	codeState := &AuthState{
		State:        ourCode,
		CodeVerifier: authState.CodeVerifier,
		RedirectURI:  authState.RedirectURI,
		ClientID:     authState.ClientID,
		Resource:     authState.Resource, // Preserve resource for token endpoint
		CreatedAt:    time.Now(),
	}

	if err := s.store.StoreState(codeState); err != nil {
		s.logger.Error("failed to store code state", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	if err := s.store.StoreToken(tempToken); err != nil {
		s.logger.Error("failed to store temp token", "error", err)
		s.errorResponse(w, "server_error", "Internal server error")
		return
	}

	redirectURL, _ := url.Parse(authState.RedirectURI)
	q := redirectURL.Query()
	q.Set("code", ourCode)
	q.Set("state", claudeState)
	redirectURL.RawQuery = q.Encode()

	s.logger.Info("OAuth callback successful, redirecting to Claude",
		"redirect_uri", authState.RedirectURI,
	)

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (s *Server) TokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.tokenError(w, "invalid_request", "Failed to parse request")
		return
	}

	grantType := r.FormValue("grant_type")

	switch grantType {
	case "authorization_code":
		s.handleAuthorizationCodeGrant(w, r)
	case "refresh_token":
		s.handleRefreshTokenGrant(w, r)
	case "client_credentials":
		s.handleClientCredentialsGrant(w, r)
	default:
		s.tokenError(w, "unsupported_grant_type", "Unsupported grant type")
	}
}

func (s *Server) handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	codeVerifier := r.FormValue("code_verifier")
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")

	if code == "" {
		s.tokenError(w, "invalid_request", "Missing code")
		return
	}

	codeState, err := s.store.ConsumeState(code)
	if err != nil {
		s.logger.Error("failed to get code state", "error", err)
		s.tokenError(w, "invalid_grant", "Invalid or expired code")
		return
	}

	if clientID != "" && clientID != codeState.ClientID {
		s.logger.Error("client_id mismatch", "expected", codeState.ClientID, "got", clientID)
		s.tokenError(w, "invalid_grant", "client_id does not match")
		return
	}
	if redirectURI != "" && redirectURI != codeState.RedirectURI {
		s.logger.Error("redirect_uri mismatch", "expected", codeState.RedirectURI, "got", redirectURI)
		s.tokenError(w, "invalid_grant", "redirect_uri does not match")
		return
	}

	if codeVerifier == "" {
		s.tokenError(w, "invalid_request", "Missing code_verifier")
		return
	}

	h := sha256.Sum256([]byte(codeVerifier))
	calculatedChallenge := base64.RawURLEncoding.EncodeToString(h[:])

	if calculatedChallenge != codeState.CodeVerifier {
		s.logger.Error("PKCE verification failed", "client_id", codeState.ClientID)
		s.tokenError(w, "invalid_grant", "PKCE verification failed")
		return
	}

	tempToken, err := s.store.GetTokenByAccess(code)
	if err != nil {
		s.logger.Error("failed to get temp token", "error", err)
		s.tokenError(w, "invalid_grant", "Invalid or expired code")
		return
	}

	_ = s.store.DeleteToken(code)

	accessToken, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate access token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	refreshToken, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate refresh token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	tokenInfo := &TokenInfo{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresAt:        time.Now().Add(s.accessTokenTTL),
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		GoogleToken:      tempToken.GoogleToken,
		ClientID:         codeState.ClientID,
		CreatedAt:        time.Now(),
	}

	if err := s.store.StoreToken(tokenInfo); err != nil {
		s.logger.Error("failed to store token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	s.logger.Info("issued access token", "client_id", codeState.ClientID)

	s.tokenResponse(w, accessToken, refreshToken, int(s.accessTokenTTL.Seconds()))
}

func (s *Server) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.FormValue("refresh_token")

	if refreshToken == "" {
		s.tokenError(w, "invalid_request", "Missing refresh_token")
		return
	}

	tokenInfo, err := s.store.GetTokenByRefresh(refreshToken)
	if err != nil {
		s.logger.Error("failed to get token by refresh", "error", err)
		s.tokenError(w, "invalid_grant", "Invalid refresh token")
		return
	}

	if tokenInfo.GoogleToken.Expiry.Before(time.Now()) {
		newGoogleToken, err := s.google.RefreshToken(r.Context(), tokenInfo.GoogleToken.RefreshToken)
		if err != nil {
			s.logger.Error("failed to refresh Google token", "error", err)
			s.tokenError(w, "invalid_grant", "Failed to refresh upstream token")
			return
		}
		tokenInfo.GoogleToken = newGoogleToken
	}

	newAccessToken, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate access token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	newRefreshToken, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("failed to generate refresh token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	newTokenInfo := &TokenInfo{
		AccessToken:      newAccessToken,
		RefreshToken:     newRefreshToken,
		ExpiresAt:        time.Now().Add(s.accessTokenTTL),
		RefreshExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		GoogleToken:      tokenInfo.GoogleToken,
		ClientID:         tokenInfo.ClientID,
		CreatedAt:        time.Now(),
	}

	if err := s.store.RotateToken(tokenInfo.AccessToken, newTokenInfo); err != nil {
		s.logger.Error("failed to rotate token", "error", err)
		s.tokenError(w, "server_error", "Internal server error")
		return
	}

	s.logger.Info("refreshed access token", "client_id", tokenInfo.ClientID)

	s.tokenResponse(w, newAccessToken, newRefreshToken, int(s.accessTokenTTL.Seconds()))
}

func (s *Server) tokenResponse(w http.ResponseWriter, accessToken, refreshToken string, expiresIn int) {
	resp := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    expiresIn,
		"refresh_token": refreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	json.NewEncoder(w).Encode(resp)
}

func (s *Server) tokenError(w http.ResponseWriter, errCode, errDesc string) {
	resp := map[string]string{
		"error":             errCode,
		"error_description": errDesc,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) errorResponse(w http.ResponseWriter, errCode, errDesc string) {
	http.Error(w, fmt.Sprintf("%s: %s", errCode, errDesc), http.StatusBadRequest)
}

var validRedirectHosts = map[string]struct {
	scheme     string
	pathPrefix string
}{
	"claude.ai":           {"https", "/api/mcp/auth_callback"},
	"claude.com":          {"https", "/api/mcp/auth_callback"},
	"chatgpt.com":         {"https", "/connector_platform_oauth_redirect"},
	"platform.openai.com": {"https", "/apps-manage/oauth"},
	"www.cursor.com":      {"https", "/agents/mcp/oauth/callback"},
	"cursor.com":          {"https", "/agents/mcp/oauth/callback"},
	"api2.cursor.sh":      {"https", "/agents/mcp/oauth/callback"},
}

var validRedirectCustomSchemes = []string{
	"cursor://anysphere.cursor-mcp/oauth/callback",
}

func isValidRedirectURI(uri string) bool {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme == "" {
		return false
	}

	for _, prefix := range validRedirectCustomSchemes {
		if strings.HasPrefix(uri, prefix) {
			return true
		}
	}

	if parsed.Host == "" {
		return false
	}

	hostname := parsed.Hostname()

	if hostname == "localhost" || hostname == "127.0.0.1" {
		return parsed.Scheme == "http" || parsed.Scheme == "https"
	}

	if rule, ok := validRedirectHosts[hostname]; ok {
		return parsed.Scheme == rule.scheme && strings.HasPrefix(parsed.Path, rule.pathPrefix)
	}

	return false
}

func (s *Server) handleClientCredentialsGrant(w http.ResponseWriter, r *http.Request) {
	if s.saProvider == nil || s.saValidator == nil {
		s.tokenError(w, "unsupported_grant_type",
			"client_credentials grant requires Service Account configuration")
		return
	}

	assertionType := r.FormValue("client_assertion_type")
	const expectedAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"
	if assertionType != expectedAssertionType {
		s.tokenError(w, "invalid_client",
			fmt.Sprintf("client_assertion_type must be %q", expectedAssertionType))
		return
	}

	clientAssertion := r.FormValue("client_assertion")
	if clientAssertion == "" {
		s.tokenError(w, "invalid_client", "client_assertion is required")
		return
	}

	claims, err := s.saValidator.ValidateJWT(r.Context(), clientAssertion)
	if err != nil {
		s.logger.Warn("sa_client_creds_failed",
			"reason", "jwt_validation",
			"error", err.Error(),
		)
		s.tokenError(w, "invalid_client", "Invalid client assertion")
		return
	}

	googleToken, err := s.saProvider.GoogleToken(r.Context())
	if err != nil {
		s.logger.Error("sa_google_token_failed",
			"service_account", s.saProvider.Email(),
			"error", err,
		)
		s.tokenError(w, "server_error", "Failed to obtain service account token")
		return
	}

	mcpToken, err := GenerateToken(32)
	if err != nil {
		s.logger.Error("sa_token_generation_failed", "error", err)
		s.tokenError(w, "server_error", "Failed to generate token")
		return
	}

	_ = r.FormValue("scope") // Accept scope param for RFC compliance

	now := time.Now()
	ttl := s.accessTokenTTL
	if ttl > 1*time.Hour {
		ttl = 1 * time.Hour
	}

	tokenInfo := &TokenInfo{
		AccessToken:  mcpToken,
		RefreshToken: "",            // No refresh token for SA — SA always gets fresh tokens via JWT
		ClientID:     claims.Issuer, // SA email as client identifier
		ExpiresAt:    now.Add(ttl),
		GoogleToken:  googleToken,
		CreatedAt:    now,
	}

	if err := s.store.StoreToken(tokenInfo); err != nil {
		s.logger.Error("sa_token_store_failed", "error", err)
		s.tokenError(w, "server_error", "Failed to store token")
		return
	}

	s.logger.Info("sa_client_credentials_issued",
		"service_account", claims.Issuer,
		"mode", string(s.saProvider.Mode()),
		"token_ttl", ttl.String(),
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": mcpToken,
		"token_type":   "Bearer",
		"expires_in":   int(ttl.Seconds()),
	})
}
