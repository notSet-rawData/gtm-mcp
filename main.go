// Package main constitutes the entrypoint for the NotSet GTM MCP Community Edition.
// Yes, we open-sourced the gateway logic. Try not to break the MCP Protocol while looking at it.
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
	"gtm-mcp-server/auth/serviceauth"
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
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	if cfg.LogLevel == "debug" {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
	}

	stdioMode := flag.Bool("stdio", false, "Run in stdio mode (no HTTP server, no auth required)")
	flag.Parse()

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
  workspace → list, create, status, delete
  tag → list, get, create, update, delete, revert, append_list_entry, remove_list_entry, list_entries
  trigger → list, get, create, update, delete, revert
  variable → list, get, create, update, delete, revert, add_lookup_entry, remove_lookup_entry, list_lookup_entries, append_list_entry, remove_list_entry, list_entries
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

	server.AddReceivingMiddleware(middleware.NewToolCompatMiddleware(logger))
	server.AddReceivingMiddleware(middleware.NewLoggingMiddleware(logger))
	server.AddReceivingMiddleware(middleware.NewAuditMiddleware(logger))

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

	registerTools(server)

	if *stdioMode {
		logger.Info("starting in stdio mode (no auth)")
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			logger.Error("stdio server error", "error", err)
			os.Exit(1)
		}
		return
	}

	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": serverName,
			"version": serverVersion,
		})
	})

	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		status := map[string]any{
			"service": serverName,
			"version": serverVersion,
		}

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

	var urlResolver *auth.URLResolver
	if len(cfg.AllowedHosts) > 0 {
		urlResolver = auth.NewURLResolver(cfg.BaseURL, cfg.AllowedHosts)
		logger.Info("dynamic URL resolution enabled", "allowed_hosts", cfg.AllowedHosts)
	}

	mux.HandleFunc("GET /.well-known/oauth-protected-resource",
		auth.ProtectedResourceMetadataHandler(cfg.BaseURL, cfg.BaseURL, urlResolver))

	mux.HandleFunc("GET /.well-known/oauth-authorization-server", auth.MetadataHandler(cfg.BaseURL, urlResolver, cfg.IsServiceAccountEnabled()))

	var authServer *auth.Server
	var tokenStore auth.TokenStore
	oauthConfigured := cfg.ValidateAuth() == nil

	oauthLimiter := middleware.NewRateLimiter(10, 20)  // 10 req/s, burst 20
	registerLimiter := middleware.NewRateLimiter(2, 5) // 2 req/s, burst 5
	mcpLimiter := middleware.NewRateLimiter(30, 50)    // 30 req/s, burst 50

	if oauthConfigured {
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

		var saProvider *serviceauth.Provider
		var saValidator *serviceauth.Validator

		if cfg.IsServiceAccountEnabled() {
			var saErr error
			saProvider, saErr = serviceauth.NewProviderFromEnv(cfg.GoogleScopes, logger)
			if saErr != nil {
				logger.Error("service account initialization failed",
					"error", saErr,
					"hint", "Check GTM_SA_KEY_JSON or GTM_SA_KEY_FILE environment variable",
				)
				os.Exit(1)
			}

			if saProvider != nil {
				if valErr := saProvider.Validate(context.Background()); valErr != nil {
					logger.Error("service account credential validation failed",
						"error", valErr,
						"service_account", saProvider.Email(),
					)
					os.Exit(1)
				}

				saValidator = serviceauth.NewValidator(serviceauth.ValidatorConfig{
					AllowedSAs: cfg.AllowedServiceAccounts,
					Audience:   cfg.ServiceAccountAudienceOrDefault(),
				})

				authServer.WithServiceAccount(saProvider, saValidator)

				logger.Info("service account authentication enabled",
					"service_account", saProvider.Email(),
					"mode", string(saProvider.Mode()),
					"subject", saProvider.Subject(),
					"audience", cfg.ServiceAccountAudienceOrDefault(),
					"allowed_sa_count", len(cfg.AllowedServiceAccounts),
				)
			}
		}

		mux.HandleFunc("GET /authorize", oauthLimiter.MiddlewareFunc(authServer.AuthorizeHandler))
		mux.HandleFunc("GET /oauth/callback", oauthLimiter.MiddlewareFunc(authServer.CallbackHandler))
		mux.HandleFunc("POST /token", oauthLimiter.MiddlewareFunc(middleware.MaxBytesMiddleware(1<<20, authServer.TokenHandler)))
		mux.HandleFunc("POST /register", registerLimiter.MiddlewareFunc(middleware.MaxBytesMiddleware(1<<20, authServer.RegistrationHandler)))

		authMiddleware := auth.Middleware(tokenStore, googleProvider, saProvider, saValidator, logger, cfg.BaseURL, cfg.AccessTokenTTL, urlResolver)
		mux.Handle("/", mcpLimiter.Middleware(authMiddleware(maxBytesHandler(5<<20, mcpHandler))))

		logger.Info("OAuth configured",
			"authorize_endpoint", cfg.BaseURL+"/authorize",
			"token_endpoint", cfg.BaseURL+"/token",
			"callback_endpoint", cfg.BaseURL+"/oauth/callback",
			"register_endpoint", cfg.BaseURL+"/register",
			"protected_resource_metadata", cfg.BaseURL+"/.well-known/oauth-protected-resource",
			"authorization_server_metadata", cfg.BaseURL+"/.well-known/oauth-authorization-server",
			"service_account_enabled", cfg.IsServiceAccountEnabled(),
		)
	} else {
		logger.Warn("OAuth not configured, running without authentication", "error", cfg.ValidateAuth())

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

		mux.Handle("/", mcpLimiter.Middleware(maxBytesHandler(5<<20, mcpHandler)))
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0, // Disabled for SSE streams
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	<-ctx.Done()
	logger.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("server stopped")
}

func registerTools(server *mcp.Server) {
	gtm.RegisterTools(server)
}

func maxBytesHandler(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}
