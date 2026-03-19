package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProtectedResourceMetadataHandler_StaticBaseURL(t *testing.T) {
	handler := ProtectedResourceMetadataHandler("https://mcp.notset.es", "https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var meta ProtectedResourceMetadata
	if err := json.NewDecoder(w.Body).Decode(&meta); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Resource should have trailing slash for root URLs (Gemini CLI compatibility)
	if meta.Resource != "https://mcp.notset.es/" {
		t.Errorf("resource = %q, want https://mcp.notset.es/ (with trailing slash)", meta.Resource)
	}
	if len(meta.AuthorizationServers) != 1 || meta.AuthorizationServers[0] != "https://mcp.notset.es" {
		t.Errorf("authorization_servers = %v", meta.AuthorizationServers)
	}
}

func TestProtectedResourceMetadataHandler_DynamicResolver_TrustedHost(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", []string{"gtm-mcp:8080"})
	handler := ProtectedResourceMetadataHandler("https://mcp.notset.es", "https://mcp.notset.es", resolver)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	req.Host = "gtm-mcp:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var meta ProtectedResourceMetadata
	json.NewDecoder(w.Body).Decode(&meta)

	if meta.Resource != "http://gtm-mcp:8080/" {
		t.Errorf("resource = %q, want http://gtm-mcp:8080/", meta.Resource)
	}
	if len(meta.AuthorizationServers) != 1 || meta.AuthorizationServers[0] != "http://gtm-mcp:8080" {
		t.Errorf("authorization_servers = %v", meta.AuthorizationServers)
	}
}

func TestProtectedResourceMetadataHandler_DynamicResolver_UntrustedHost(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", nil)
	handler := ProtectedResourceMetadataHandler("https://mcp.notset.es", "https://mcp.notset.es", resolver)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	req.Host = "evil.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var meta ProtectedResourceMetadata
	json.NewDecoder(w.Body).Decode(&meta)

	// Should fall back to configured URL
	if meta.Resource != "https://mcp.notset.es/" {
		t.Errorf("resource = %q, should use configured URL not evil.com", meta.Resource)
	}
}

func TestNormalizeResourceURL_RootURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://mcp.notset.es", "https://mcp.notset.es/"},
		{"https://mcp.notset.es/", "https://mcp.notset.es/"},
		{"http://localhost:8080", "http://localhost:8080/"},
		{"http://localhost:8080/", "http://localhost:8080/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeResourceURL(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeResourceURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeResourceURL_PathURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/mcp", "https://example.com/mcp"},
		{"https://example.com/mcp/", "https://example.com/mcp/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeResourceURL(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeResourceURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestProtectedResourceMetadataHandler_MethodNotAllowed(t *testing.T) {
	handler := ProtectedResourceMetadataHandler("https://mcp.notset.es", "https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodPost, "/.well-known/oauth-protected-resource", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestMetadataConsistency_IssuerMatchesAuthServer(t *testing.T) {
	// Verify that the issuer in auth metadata matches the authorization_servers
	// entry in protected resource metadata (without trailing slash)
	baseURL := "https://mcp.notset.es"

	authMeta := NewOAuthMetadata(baseURL)
	resMeta := NewProtectedResourceMetadata(baseURL, baseURL)

	if len(resMeta.AuthorizationServers) != 1 {
		t.Fatalf("expected 1 authorization server, got %d", len(resMeta.AuthorizationServers))
	}

	if authMeta.Issuer != resMeta.AuthorizationServers[0] {
		t.Errorf("issuer %q != authorization_servers[0] %q — these must match",
			authMeta.Issuer, resMeta.AuthorizationServers[0])
	}
}
