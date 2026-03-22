package main

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gtm-mcp-server/auth"
	"gtm-mcp-server/config"
	"gtm-mcp-server/gtm"
	"gtm-mcp-server/middleware"
	"gtm-mcp-server/store"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	_ "modernc.org/sqlite"
)

//go:embed mcp-logo.png
var mcpLogoPNG []byte

const (
	serverName    = "gtm-mcp-server"
	serverVersion = "1.0.0"
)

func main() {
	// Set up structured logging to stderr (stdout is reserved for MCP in stdio mode)
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Adjust log level
	if cfg.LogLevel == "debug" {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
	}

	// Parse flags
	stdioMode := flag.Bool("stdio", false, "Run in stdio mode (no HTTP server, no auth required)")
	flag.Parse()

	// Create MCP server
	logoDataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(mcpLogoPNG)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Title:   "NotSet GTM MCP Server",
		Version: serverVersion,
		Icons: []mcp.Icon{
			{
				Source:   logoDataURI,
				MIMEType: "image/png",
			},
		},
	}, &mcp.ServerOptions{
		Instructions: `This server exposes a SINGLE gateway tool called "gtm". ALL operations go through this one tool.

Usage: call tool "gtm" with {"resource": "<resource>", "action": "<action>", "args": {<params>}}

IMPORTANT BEHAVIORS:
  - UPDATE operations are PARTIAL: omitted fields are preserved from the existing entity. You only need to send fields you want to change.
  - To ADD entries to a list parameter (e.g. RegEx table rows), first GET the entity to see current params, then send the FULL updated parameter array.
  - Community template tags use container-specific type IDs like "cvt_CONTAINERID_NNN". Use templates_ref to discover them. NEVER guess type names.
  - Parameters use a nested structure: {"type": "template|boolean|integer|list|map", "key": "...", "value": "...", "list": [...], "map": [...]}

Resources & actions:
  account → list
  container → list, create, delete
  workspace → list, create, status
  tag → list, get, create, update, delete, revert
  trigger → list, get, create, update, delete, revert
  variable → list, get, create, update, delete, revert
  folder → list, get, create, update, delete, move, audit, revert
  template → list, get, create, update, delete, import, revert
  built_in_variable → list, enable, disable, revert
  client → list, get, create, update, delete, revert
  transformation → list, get, create, update, delete, revert
  environment → list, get, create, update, delete
  user_permission → list, get, create, update, delete
  version → list, get, create, publish, compare, find_by_date, set_latest, export
  destination → list, get, link
  zone → list, get, create, update, delete, revert
  gtag_config → list, get, create, update, delete
  templates_ref → tag_templates, trigger_templates
  ping → (no action needed)
  auth_status → (no action needed)`,
	})

	// Add middleware (order matters: compat first, then logging, then audit, then transport mode)
	server.AddReceivingMiddleware(middleware.NewToolCompatMiddleware(logger))
	server.AddReceivingMiddleware(middleware.NewLoggingMiddleware(logger))
	server.AddReceivingMiddleware(middleware.NewAuditMiddleware(logger))

	// Transport mode middleware: injects the correct TransportMode into the context.
	// Uses the stdioMode flag (captured by closure) to determine the mode.
	isStdio := *stdioMode
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if isStdio {
				ctx = gtm.WithTransportMode(ctx, gtm.TransportStdio)
			} else {
				ctx = gtm.WithTransportMode(ctx, gtm.TransportHTTP)
			}
			return next(ctx, method, req)
		}
	})

	// Register tools
	registerTools(server)

	// Stdio mode: direct MCP over stdin/stdout, no auth
	if *stdioMode {
		logger.Info("starting in stdio mode (no auth)")
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			logger.Error("stdio server error", "error", err)
			os.Exit(1)
		}
		return
	}

	// Create HTTP handler for MCP
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Health check endpoint (no auth required)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": serverName,
			"version": serverVersion,
		})
	})

	// Readiness check — verifies backend dependencies
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status := map[string]any{
			"service": serverName,
			"version": serverVersion,
		}

		// Quick SQLite check: try to open and ping the database
		db, err := sql.Open("sqlite", "data/gtm-tokens.db")
		if err != nil {
			status["status"] = "degraded"
			status["token_store"] = "error: " + err.Error()
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			pingErr := db.Ping()
			_ = db.Close()
			if pingErr != nil {
				status["status"] = "degraded"
				status["token_store"] = "error: " + pingErr.Error()
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				status["status"] = "ready"
				status["token_store"] = "ok"
				w.WriteHeader(http.StatusOK)
			}
		}

		json.NewEncoder(w).Encode(status)
	})

	// URL resolver for dynamic base URL resolution in Docker-to-Docker contexts.
	// Only resolves dynamically for hosts in the allowlist; falls back to cfg.BaseURL.
	var urlResolver *auth.URLResolver
	if len(cfg.AllowedHosts) > 0 {
		urlResolver = auth.NewURLResolver(cfg.BaseURL, cfg.AllowedHosts)
		logger.Info("dynamic URL resolution enabled", "allowed_hosts", cfg.AllowedHosts)
	}

	// OAuth metadata endpoints (always served, no auth required)
	// RFC 9728: Protected Resource Metadata - tells clients where to find the authorization server
	mux.HandleFunc("GET /.well-known/oauth-protected-resource",
		auth.ProtectedResourceMetadataHandler(cfg.BaseURL, cfg.BaseURL, urlResolver))

	// RFC 8414: Authorization Server Metadata - tells clients about OAuth endpoints
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", auth.MetadataHandler(cfg.BaseURL, urlResolver))

	// Check if OAuth is configured
	var authServer *auth.Server
	var tokenStore auth.TokenStore
	oauthConfigured := cfg.ValidateAuth() == nil

	// Rate limiters for public endpoints
	oauthLimiter := middleware.NewRateLimiter(10, 20)  // 10 req/s, burst 20
	registerLimiter := middleware.NewRateLimiter(2, 5) // 2 req/s, burst 5
	mcpLimiter := middleware.NewRateLimiter(30, 50)    // 30 req/s, burst 50

	if oauthConfigured {
		// Set up OAuth with encrypted token store
		encryptionKey := auth.DeriveKey(cfg.JWTSecret)
		var err error
		tokenStore, err = store.NewSQLiteTokenStore("data/gtm-tokens.db", encryptionKey)
		if err != nil {
			logger.Error("failed to initialize sqlite token store", "error", err)
			os.Exit(1)
		}

		googleProvider := auth.NewGoogleProvider(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.BaseURL+"/oauth/callback",
			cfg.GoogleScopes...,
		)
		authServer = auth.NewServer(cfg.BaseURL, googleProvider, tokenStore, logger, cfg.AccessTokenTTL, cfg.AllowedDCRDomains...)

		// OAuth endpoints with rate limiting and body size limits
		mux.HandleFunc("GET /authorize", oauthLimiter.MiddlewareFunc(authServer.AuthorizeHandler))
		mux.HandleFunc("GET /oauth/callback", oauthLimiter.MiddlewareFunc(authServer.CallbackHandler))
		mux.HandleFunc("POST /token", oauthLimiter.MiddlewareFunc(middleware.MaxBytesMiddleware(1<<20, authServer.TokenHandler)))
		mux.HandleFunc("POST /register", registerLimiter.MiddlewareFunc(middleware.MaxBytesMiddleware(1<<20, authServer.RegistrationHandler)))

		// MCP endpoint with REQUIRED auth middleware, rate limiting, and body size limit
		// Returns 401 if no valid Bearer token - triggers Claude's OAuth flow
		authMiddleware := auth.Middleware(tokenStore, googleProvider, logger, cfg.BaseURL, cfg.AccessTokenTTL, urlResolver)
		mux.Handle("/", mcpLimiter.Middleware(authMiddleware(maxBytesHandler(5<<20, mcpHandler))))

		logger.Info("OAuth configured",
			"authorize_endpoint", cfg.BaseURL+"/authorize",
			"token_endpoint", cfg.BaseURL+"/token",
			"callback_endpoint", cfg.BaseURL+"/oauth/callback",
			"register_endpoint", cfg.BaseURL+"/register",
			"protected_resource_metadata", cfg.BaseURL+"/.well-known/oauth-protected-resource",
			"authorization_server_metadata", cfg.BaseURL+"/.well-known/oauth-authorization-server",
		)
	} else {
		logger.Warn("OAuth not configured, running without authentication", "error", cfg.ValidateAuth())

		// Register OAuth endpoints that return proper errors
		oauthNotConfiguredHandler := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"error":             "server_error",
				"error_description": "OAuth is not configured on this server.",
			})
		}
		mux.HandleFunc("GET /authorize", oauthLimiter.MiddlewareFunc(oauthNotConfiguredHandler))
		mux.HandleFunc("GET /oauth/callback", oauthLimiter.MiddlewareFunc(oauthNotConfiguredHandler))
		mux.HandleFunc("POST /token", oauthLimiter.MiddlewareFunc(oauthNotConfiguredHandler))
		mux.HandleFunc("POST /register", registerLimiter.MiddlewareFunc(oauthNotConfiguredHandler))

		// MCP endpoint without auth (still apply rate limit and body size limit)
		mux.Handle("/", mcpLimiter.Middleware(maxBytesHandler(5<<20, mcpHandler)))
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%d", cfg.Port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0, // Disabled for SSE streams
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start server
	go func() {
		logger.Info("starting GTM MCP server",
			"port", cfg.Port,
			"base_url", cfg.BaseURL,
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("shutting down server")

	// Give outstanding requests 10 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("server stopped")
}

// registerTools adds MCP tools to the server.
func registerTools(server *mcp.Server) {
	// All tools (including ping and auth_status) are registered via the gateway
	gtm.RegisterTools(server)
}

// maxBytesHandler wraps an http.Handler with a request body size limit.
func maxBytesHandler(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}
