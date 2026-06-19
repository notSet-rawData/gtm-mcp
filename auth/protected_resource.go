package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
)

type ProtectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported,omitempty"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}

func NewProtectedResourceMetadata(baseURL, resourceURL string) *ProtectedResourceMetadata {
	return &ProtectedResourceMetadata{
		Resource:               normalizeResourceURL(resourceURL),
		AuthorizationServers:   []string{baseURL},
		ScopesSupported:        GoogleScopes,
		BearerMethodsSupported: []string{"header"},
	}
}

func normalizeResourceURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	return parsed.String()
}

func ProtectedResourceMetadataHandler(baseURL, resourceURL string, resolver *URLResolver) http.HandlerFunc {
	staticMetadata := NewProtectedResourceMetadata(baseURL, resourceURL)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		metadata := staticMetadata
		if resolver != nil {
			if resolved := resolver.Resolve(r); resolved != baseURL {
				metadata = NewProtectedResourceMetadata(resolved, resolved)
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
