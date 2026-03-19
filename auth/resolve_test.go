package auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestURLResolver_Resolve_ConfiguredHost(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "mcp.notset.es"
	req.Header.Set("X-Forwarded-Proto", "https")

	got := resolver.Resolve(req)
	if got != "https://mcp.notset.es" {
		t.Errorf("expected https://mcp.notset.es, got %s", got)
	}
}

func TestURLResolver_Resolve_AllowedHost(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", []string{"gtm-mcp:8080"})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "gtm-mcp:8080"

	got := resolver.Resolve(req)
	if got != "http://gtm-mcp:8080" {
		t.Errorf("expected http://gtm-mcp:8080, got %s", got)
	}
}

func TestURLResolver_Resolve_UntrustedHost_FallsBack(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "evil.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	got := resolver.Resolve(req)
	if got != "https://mcp.notset.es" {
		t.Errorf("expected fallback to configured URL, got %s", got)
	}
}

func TestURLResolver_Resolve_EmptyHost_FallsBack(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = ""

	got := resolver.Resolve(req)
	if got != "https://mcp.notset.es" {
		t.Errorf("expected fallback to configured URL, got %s", got)
	}
}

func TestURLResolver_Resolve_SchemeFromTLS(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "mcp.notset.es"
	req.TLS = &tls.ConnectionState{}

	got := resolver.Resolve(req)
	if got != "https://mcp.notset.es" {
		t.Errorf("expected https from TLS, got %s", got)
	}
}

func TestURLResolver_Resolve_SchemeFromForwardedProto(t *testing.T) {
	resolver := NewURLResolver("http://localhost:8080", nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:8080"
	req.Header.Set("X-Forwarded-Proto", "https")

	got := resolver.Resolve(req)
	if got != "https://localhost:8080" {
		t.Errorf("expected https from X-Forwarded-Proto, got %s", got)
	}
}

func TestURLResolver_Resolve_HTTPByDefault(t *testing.T) {
	resolver := NewURLResolver("http://localhost:8080", nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:8080"

	got := resolver.Resolve(req)
	if got != "http://localhost:8080" {
		t.Errorf("expected http by default, got %s", got)
	}
}

func TestURLResolver_Resolve_MultipleAllowedHosts(t *testing.T) {
	resolver := NewURLResolver("https://mcp.notset.es", []string{"gtm-mcp:8080", "localhost:8080"})

	tests := []struct {
		host     string
		proto    string
		expected string
	}{
		{"mcp.notset.es", "https", "https://mcp.notset.es"},
		{"gtm-mcp:8080", "", "http://gtm-mcp:8080"},
		{"localhost:8080", "", "http://localhost:8080"},
		{"evil.com", "https", "https://mcp.notset.es"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tt.host
			if tt.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.proto)
			}

			got := resolver.Resolve(req)
			if got != tt.expected {
				t.Errorf("host=%q proto=%q: expected %s, got %s", tt.host, tt.proto, tt.expected, got)
			}
		})
	}
}
