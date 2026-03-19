package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetadataHandler_StaticBaseURL(t *testing.T) {
	handler := MetadataHandler("https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var meta OAuthMetadata
	if err := json.NewDecoder(w.Body).Decode(&meta); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if meta.Issuer != "https://mcp.notset.es" {
		t.Errorf("issuer = %q, want https://mcp.notset.es", meta.Issuer)
	}
	if meta.AuthorizationEndpoint != "https://mcp.notset.es/authorize" {
		t.Errorf("authorization_endpoint = %q", meta.AuthorizationEndpoint)
	}
	if meta.TokenEndpoint != "https://mcp.notset.es/token" {
		t.Errorf("token_endpoint = %q", meta.TokenEndpoint)
	}
	if meta.RegistrationEndpoint != "https://mcp.notset.es/register" {
		t.Errorf("registration_endpoint = %q", meta.RegistrationEndpoint)
	}
}

func TestMetadataHandler_DynamicResolver_TrustedHost(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", []string{"gtm-mcp:8080"})
	handler := MetadataHandler("https://mcp.notset.es", resolver)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	req.Host = "gtm-mcp:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var meta OAuthMetadata
	json.NewDecoder(w.Body).Decode(&meta)

	if meta.Issuer != "http://gtm-mcp:8080" {
		t.Errorf("issuer = %q, want http://gtm-mcp:8080", meta.Issuer)
	}
	if meta.AuthorizationEndpoint != "http://gtm-mcp:8080/authorize" {
		t.Errorf("authorization_endpoint = %q", meta.AuthorizationEndpoint)
	}
}

func TestMetadataHandler_DynamicResolver_UntrustedHost(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", nil)
	handler := MetadataHandler("https://mcp.notset.es", resolver)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	req.Host = "evil.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var meta OAuthMetadata
	json.NewDecoder(w.Body).Decode(&meta)

	// Should fall back to configured URL, not use evil.com
	if meta.Issuer != "https://mcp.notset.es" {
		t.Errorf("issuer = %q, want https://mcp.notset.es (should not use untrusted host)", meta.Issuer)
	}
}

func TestMetadataHandler_MethodNotAllowed(t *testing.T) {
	handler := MetadataHandler("https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodPost, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestMetadataHandler_IssuerHasNoTrailingSlash(t *testing.T) {
	handler := MetadataHandler("https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var meta OAuthMetadata
	json.NewDecoder(w.Body).Decode(&meta)

	// RFC 8414: issuer should NOT have a trailing slash
	if meta.Issuer != "https://mcp.notset.es" {
		t.Errorf("issuer = %q, should not have trailing slash", meta.Issuer)
	}
}

func TestMetadataHandler_CacheHeaders(t *testing.T) {
	handler := MetadataHandler("https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "public, max-age=3600" {
		t.Errorf("Cache-Control = %q", cc)
	}
	if cors := w.Header().Get("Access-Control-Allow-Origin"); cors != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q", cors)
	}
}
