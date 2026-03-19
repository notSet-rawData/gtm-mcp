package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the GTM MCP Server.
type Config struct {
	// Server configuration
	Port    int
	BaseURL string

	// Google OAuth configuration
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string

	// JWT configuration
	JWTSecret string

	// Logging
	LogLevel string

	// Token configuration
	AccessTokenTTL time.Duration
	
	// API Retry configuration
	MaxRetries int

	// AllowedHosts lists additional trusted hostnames for dynamic base URL resolution.
	// Enables Docker-to-Docker contexts where the server is reached via internal aliases.
	AllowedHosts []string

	// TrustedProxies lists IP addresses/CIDRs of trusted reverse proxies.
	// Only trust X-Forwarded-For from these sources. Empty = trust RemoteAddr only.
	TrustedProxies []string

	// GoogleScopes configures which GTM API scopes to request.
	// Default: all GTM scopes (edit, readonly, publish, delete).
	GoogleScopes []string

	// AllowedDCRDomains restricts which domains can register via DCR.
	// Empty = accept any valid HTTPS domain (less secure).
	AllowedDCRDomains []string
}

// Load reads configuration from environment variables.
// It first attempts to load from .env file if present, then .env.local for overrides.
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()
	// Load .env.local for local development overrides (takes precedence)
	_ = godotenv.Overload(".env.local")

	cfg := &Config{
		Port:              getEnvInt("PORT", 8080),
		BaseURL:           getEnv("BASE_URL", "http://localhost:8080"),
		GoogleClientID:    getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURI: getEnv("GOOGLE_REDIRECT_URI", ""),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		AccessTokenTTL:    getEnvDuration("ACCESS_TOKEN_TTL", 8*time.Hour),
		AllowedHosts:      getEnvList("ALLOWED_HOSTS"),
		TrustedProxies:    getEnvList("TRUSTED_PROXIES"),
		GoogleScopes:      getEnvList("GOOGLE_SCOPES"),
		AllowedDCRDomains: getEnvList("ALLOWED_DCR_DOMAINS"),
		MaxRetries:        getEnvInt("MAX_RETRIES", 3),
	}

	// Validate structural config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks structural configuration constraints (format, range, etc).
// This runs at startup regardless of whether auth is configured.
func (c *Config) Validate() error {
	// Validate PORT range
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535 (got %d)", c.Port)
	}

	// Validate BASE_URL format
	if c.BaseURL != "" {
		parsed, err := url.Parse(c.BaseURL)
		if err != nil {
			return fmt.Errorf("BASE_URL is not a valid URL: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("BASE_URL must use http:// or https:// scheme (got %q)", parsed.Scheme)
		}
		if parsed.Host == "" {
			return fmt.Errorf("BASE_URL must include a host")
		}
	}

	return nil
}

// ValidateAuth checks if OAuth credentials are configured.
func (c *Config) ValidateAuth() error {
	if c.GoogleClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID is required for authentication")
	}
	if c.GoogleClientSecret == "" {
		return fmt.Errorf("GOOGLE_CLIENT_SECRET is required for authentication")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required for authentication")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters (got %d)", len(c.JWTSecret))
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getEnvList(key string) []string {
	if value := os.Getenv(key); value != "" {
		var hosts []string
		for _, h := range strings.Split(value, ",") {
			if h = strings.TrimSpace(h); h != "" {
				hosts = append(hosts, h)
			}
		}
		return hosts
	}
	return nil
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
