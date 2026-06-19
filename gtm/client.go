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

	respDump, _ := httputil.DumpResponse(resp, false) // false = no body
	if t.logger != nil {
		t.logger.Debug("HTTP RESPONSE", "status", resp.StatusCode, "headers", string(respDump))
	}

	return resp, nil
}

type Client struct {
	Service    *tagmanager.Service
	HTTPClient *http.Client // OAuth2-authenticated HTTP client for direct API calls
}

func NewClient(ctx context.Context, tokenSource oauth2.TokenSource, logger *slog.Logger) (*Client, error) {
	if tokenSource == nil {
		return nil, fmt.Errorf("token source is required")
	}

	httpClient := oauth2.NewClient(ctx, tokenSource)

	opts := []option.ClientOption{option.WithTokenSource(tokenSource)}

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
