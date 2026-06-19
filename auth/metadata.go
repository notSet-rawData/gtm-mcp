package auth

import (
	"encoding/json"
	"net/http"
)

type OAuthMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint,omitempty"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

func NewOAuthMetadata(baseURL string, serviceAccountEnabled bool) *OAuthMetadata {
	grantTypes := []string{"authorization_code", "refresh_token"}
	tokenAuthMethods := []string{"client_secret_post", "none"}

	if serviceAccountEnabled {
		grantTypes = append(grantTypes, "client_credentials")
		tokenAuthMethods = append(tokenAuthMethods, "private_key_jwt")
	}

	return &OAuthMetadata{
		Issuer:                            baseURL,
		AuthorizationEndpoint:             baseURL + "/authorize",
		TokenEndpoint:                     baseURL + "/token",
		RegistrationEndpoint:              baseURL + "/register",
		ScopesSupported:                   GoogleScopes,
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               grantTypes,
		TokenEndpointAuthMethodsSupported: tokenAuthMethods,
		CodeChallengeMethodsSupported:     []string{"S256"},
	}
}

func MetadataHandler(baseURL string, resolver *URLResolver, serviceAccountEnabled bool) http.HandlerFunc {
	staticMetadata := NewOAuthMetadata(baseURL, serviceAccountEnabled)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metadata := staticMetadata
		if resolver != nil {
			if resolved := resolver.Resolve(r); resolved != baseURL {
				metadata = NewOAuthMetadata(resolved, serviceAccountEnabled)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "public, max-age=3600")

		if err := json.NewEncoder(w).Encode(metadata); err != nil {
			http.Error(w, "Failed to encode metadata", http.StatusInternalServerError)
			return
		}
	}
}
