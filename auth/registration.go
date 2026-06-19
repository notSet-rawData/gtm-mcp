package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ClientRegistrationRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
}

type ClientRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name,omitempty"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

func (s *Server) RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.registrationError(w, "invalid_request", "Invalid JSON")
		return
	}

	if len(req.RedirectURIs) == 0 {
		s.registrationError(w, "invalid_redirect_uri", "At least one redirect_uri required")
		return
	}

	for _, uri := range req.RedirectURIs {
		if !isValidDCRRedirectURI(uri) {
			s.registrationError(w, "invalid_redirect_uri", "Invalid redirect_uri: "+uri)
			return
		}
	}

	if len(s.allowedDCRDomains) > 0 {
		for _, uri := range req.RedirectURIs {
			parsed, _ := url.Parse(uri)
			host := strings.Split(parsed.Host, ":")[0] // Strip port
			if host == "localhost" || host == "127.0.0.1" {
				continue // Always allow localhost for development
			}
			if !s.allowedDCRDomains[host] {
				s.logger.Warn("dcr_domain_rejected", "domain", host, "uri", uri)
				s.registrationError(w, "invalid_redirect_uri", "Domain not in allowlist: "+host)
				return
			}
		}
	}

	clientID, err := GenerateToken(16)
	if err != nil {
		s.logger.Error("failed to generate client_id", "error", err)
		s.registrationError(w, "server_error", "Internal server error")
		return
	}

	resp := ClientRegistrationResponse{
		ClientID:                clientID,
		ClientSecretExpiresAt:   0, // Does not expire
		RedirectURIs:            req.RedirectURIs,
		ClientName:              req.ClientName,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none", // Public client
	}

	clientInfo := &ClientInfo{
		ClientID:                clientID,
		RedirectURIs:            req.RedirectURIs,
		ClientName:              req.ClientName,
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
		CreatedAt:               time.Now(),
	}

	if err := s.store.StoreClient(clientInfo); err != nil {
		s.logger.Error("failed to store registered client", "error", err)
		s.registrationError(w, "server_error", "Internal server error")
		return
	}

	s.logger.Info("client registered", "client_id", clientID, "client_name", req.ClientName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) registrationError(w http.ResponseWriter, errCode, errDesc string) {
	resp := map[string]string{
		"error":             errCode,
		"error_description": errDesc,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(resp)
}

func isValidDCRRedirectURI(uri string) bool {
	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}

	if parsed.Scheme == "" {
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

	host := strings.Split(parsed.Host, ":")[0] // Remove port if present
	if host == "localhost" || host == "127.0.0.1" {
		return parsed.Scheme == "http" || parsed.Scheme == "https"
	}

	return parsed.Scheme == "https"
}
