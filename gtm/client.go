// Package gtm provides a client for the Google Tag Manager API.
package gtm

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	tagmanager "google.golang.org/api/tagmanager/v2"
)

var authHeaderRe = regexp.MustCompile(`(?i)(Authorization:\s*)Bearer\s+\S+`)

// loggingTransport wraps an http.RoundTripper and logs request/response bodies
// with sensitive headers redacted.
type loggingTransport struct {
	wrapped http.RoundTripper
	logger  *slog.Logger
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dump, _ := httputil.DumpRequestOut(req, true)
	redacted := authHeaderRe.ReplaceAllString(string(dump), "${1}Bearer [REDACTED]")
	
	if t.logger != nil {
		t.logger.Debug("HTTP REQUEST", "payload", redacted)
	}

	resp, err := t.wrapped.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Only log response status and headers — never log body (may contain tokens)
	respDump, _ := httputil.DumpResponse(resp, false) // false = no body
	if t.logger != nil {
		t.logger.Debug("HTTP RESPONSE", "status", resp.StatusCode, "headers", string(respDump))
	}

	return resp, nil
}

// Client wraps the Google Tag Manager API service.
type Client struct {
	Service    *tagmanager.Service
	HTTPClient *http.Client // OAuth2-authenticated HTTP client for direct API calls
}

// NewClient creates a GTM client from an OAuth2 token source.
// The token source should handle automatic refresh.
func NewClient(ctx context.Context, tokenSource oauth2.TokenSource, logger *slog.Logger) (*Client, error) {
	if tokenSource == nil {
		return nil, fmt.Errorf("token source is required")
	}

	// Create an OAuth2-authenticated HTTP client for direct API calls
	httpClient := oauth2.NewClient(ctx, tokenSource)

	opts := []option.ClientOption{option.WithTokenSource(tokenSource)}

	// Enable HTTP request/response logging when GTM_DEBUG is set
	if os.Getenv("GTM_DEBUG") != "" {
		baseURL := os.Getenv("BASE_URL")
		if baseURL != "" && !strings.Contains(baseURL, "localhost") && !strings.Contains(baseURL, "127.0.0.1") {
			if logger != nil {
				logger.Warn("GTM_DEBUG ignored in production", "BASE_URL", baseURL)
			}
		} else {
			if logger != nil {
				logger.Warn("GTM_DEBUG is enabled — HTTP bodies will be logged (headers redacted)")
			}
			httpClient.Transport = &loggingTransport{wrapped: httpClient.Transport, logger: logger}
			opts = []option.ClientOption{option.WithHTTPClient(httpClient)}
		}
	}

	service, err := tagmanager.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create tagmanager service: %w", err)
	}

	return &Client{Service: service, HTTPClient: httpClient}, nil
}
