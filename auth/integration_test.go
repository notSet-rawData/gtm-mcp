package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIntegration_MetadataAndMiddleware401_Consistency(t *testing.T) {
	baseURL := "https://mcp.notset.es"
	store := newMockTokenStore()
	logger := testLogger()

	metaHandler := MetadataHandler(baseURL, nil, false)
	mw := Middleware(store, nil, nil, nil, logger, baseURL, 1*time.Hour, nil)
	protectedHandler := mw(dummyHandler)

	metaReq := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	metaW := httptest.NewRecorder()
	metaHandler.ServeHTTP(metaW, metaReq)

	var meta OAuthMetadata
	json.NewDecoder(metaW.Body).Decode(&meta)

	authReq := httptest.NewRequest(http.MethodGet, "/", nil)
	authW := httptest.NewRecorder()
	protectedHandler.ServeHTTP(authW, authReq)

	if authW.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", authW.Code)
	}

	var errResp map[string]string
	json.NewDecoder(authW.Body).Decode(&errResp)

	if errResp["authorization_endpoint"] != meta.AuthorizationEndpoint {
		t.Errorf("401 authorization_endpoint %q != metadata %q",
			errResp["authorization_endpoint"], meta.AuthorizationEndpoint)
	}
	if errResp["token_endpoint"] != meta.TokenEndpoint {
		t.Errorf("401 token_endpoint %q != metadata %q",
			errResp["token_endpoint"], meta.TokenEndpoint)
	}
}

func TestIntegration_MetadataAndMiddleware401_ConsistencyWithResolver(t *testing.T) {
	baseURL := "https://mcp.notset.es"
	resolver := NewURLResolver(baseURL, []string{"gtm-mcp:8080"})
	store := newMockTokenStore()
	logger := testLogger()

	metaHandler := MetadataHandler(baseURL, resolver, false)
	resHandler := ProtectedResourceMetadataHandler(baseURL, baseURL, resolver)
	mw := Middleware(store, nil, nil, nil, logger, baseURL, 1*time.Hour, resolver)
	protectedHandler := mw(dummyHandler)

	host := "gtm-mcp:8080"

	metaReq := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	metaReq.Host = host
	metaW := httptest.NewRecorder()
	metaHandler.ServeHTTP(metaW, metaReq)

	var meta OAuthMetadata
	json.NewDecoder(metaW.Body).Decode(&meta)

	if meta.Issuer != "http://gtm-mcp:8080" {
		t.Errorf("issuer = %q, want http://gtm-mcp:8080", meta.Issuer)
	}

	resReq := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
	resReq.Host = host
	resW := httptest.NewRecorder()
	resHandler.ServeHTTP(resW, resReq)

	var resMeta ProtectedResourceMetadata
	json.NewDecoder(resW.Body).Decode(&resMeta)

	if resMeta.Resource != "http://gtm-mcp:8080/" {
		t.Errorf("resource = %q, want http://gtm-mcp:8080/", resMeta.Resource)
	}
	if len(resMeta.AuthorizationServers) != 1 || resMeta.AuthorizationServers[0] != "http://gtm-mcp:8080" {
		t.Errorf("authorization_servers = %v, want [http://gtm-mcp:8080]", resMeta.AuthorizationServers)
	}

	if meta.Issuer != resMeta.AuthorizationServers[0] {
		t.Errorf("issuer %q != authorization_servers[0] %q", meta.Issuer, resMeta.AuthorizationServers[0])
	}

	authReq := httptest.NewRequest(http.MethodGet, "/", nil)
	authReq.Host = host
	authW := httptest.NewRecorder()
	protectedHandler.ServeHTTP(authW, authReq)

	var errResp map[string]string
	json.NewDecoder(authW.Body).Decode(&errResp)

	if errResp["authorization_endpoint"] != meta.AuthorizationEndpoint {
		t.Errorf("401 authorization_endpoint %q != metadata %q",
			errResp["authorization_endpoint"], meta.AuthorizationEndpoint)
	}
	if errResp["token_endpoint"] != meta.TokenEndpoint {
		t.Errorf("401 token_endpoint %q != metadata %q",
			errResp["token_endpoint"], meta.TokenEndpoint)
	}

	wwwAuth := authW.Header().Get("WWW-Authenticate")
	expectedResourceMeta := `resource_metadata="http://gtm-mcp:8080/.well-known/oauth-protected-resource"`
	if !containsSubstr(wwwAuth, expectedResourceMeta) {
		t.Errorf("WWW-Authenticate = %q, want it to contain %q", wwwAuth, expectedResourceMeta)
	}
}

func TestIntegration_UntrustedHostDoesNotLeakIntoResponses(t *testing.T) {
	baseURL := "https://mcp.notset.es"
	resolver := NewURLResolver(baseURL, nil)
	store := newMockTokenStore()
	logger := testLogger()

	handlers := map[string]http.Handler{
		"metadata":           MetadataHandler(baseURL, resolver, false),
		"protected_resource": ProtectedResourceMetadataHandler(baseURL, baseURL, resolver),
		"middleware_401":     Middleware(store, nil, nil, nil, logger, baseURL, 1*time.Hour, resolver)(dummyHandler),
	}

	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = "evil.com"
			req.Header.Set("X-Forwarded-Proto", "https")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			body := w.Body.String()
			if containsSubstr(body, "evil.com") {
				t.Errorf("response body contains untrusted host 'evil.com': %s", body)
			}

			for key, values := range w.Header() {
				for _, v := range values {
					if containsSubstr(v, "evil.com") {
						t.Errorf("response header %s contains untrusted host: %s", key, v)
					}
				}
			}
		})
	}
}

func TestIntegration_ResourceMetadata_GeminiCLICompatibility(t *testing.T) {
	tests := []struct {
		name         string
		baseURL      string
		wantResource string
	}{
		{
			name:         "root URL without slash",
			baseURL:      "https://mcp.notset.es",
			wantResource: "https://mcp.notset.es/",
		},
		{
			name:         "root URL with slash",
			baseURL:      "https://mcp.notset.es/",
			wantResource: "https://mcp.notset.es/",
		},
		{
			name:         "localhost with port",
			baseURL:      "http://localhost:8080",
			wantResource: "http://localhost:8080/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ProtectedResourceMetadataHandler(tt.baseURL, tt.baseURL, nil)

			req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			var meta ProtectedResourceMetadata
			json.NewDecoder(w.Body).Decode(&meta)

			if meta.Resource != tt.wantResource {
				t.Errorf("resource = %q, want %q", meta.Resource, tt.wantResource)
			}
		})
	}
}
