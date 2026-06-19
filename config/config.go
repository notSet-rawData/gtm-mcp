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

type Config struct {
	Port    int
	BaseURL string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string

	JWTSecret string

	LogLevel string

	AccessTokenTTL time.Duration

	MaxRetries int

	AllowedHosts []string

	TrustedProxies []string

	GoogleScopes []string

	AllowedDCRDomains []string

	ServiceAccountKeyJSON  string
	ServiceAccountKeyFile  string
	ServiceAccountSubject  string
	ServiceAccountAudience string
	AllowedServiceAccounts []string
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	_ = godotenv.Overload(".env.local")

	cfg := &Config{
		Port:               getEnvInt("PORT", 8080),
		BaseURL:            getEnv("BASE_URL", "http://localhost:8080"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURI:  getEnv("GOOGLE_REDIRECT_URI", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		AccessTokenTTL:     getEnvDuration("ACCESS_TOKEN_TTL", 8*time.Hour),
		AllowedHosts:       getEnvList("ALLOWED_HOSTS"),
		TrustedProxies:     getEnvList("TRUSTED_PROXIES"),
		GoogleScopes:       getEnvList("GOOGLE_SCOPES"),
		AllowedDCRDomains:  getEnvList("ALLOWED_DCR_DOMAINS"),
		MaxRetries:         getEnvInt("MAX_RETRIES", 3),

		ServiceAccountKeyJSON:  getEnv("GTM_SA_KEY_JSON", ""),
		ServiceAccountKeyFile:  getEnv("GTM_SA_KEY_FILE", ""),
		ServiceAccountSubject:  getEnv("GTM_SA_SUBJECT", ""),
		ServiceAccountAudience: getEnv("GTM_SA_AUDIENCE", ""),
		AllowedServiceAccounts: getEnvList("GTM_ALLOWED_SA_EMAILS"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535 (got %d)", c.Port)
	}

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

func (c *Config) IsServiceAccountEnabled() bool {
	return c.ServiceAccountKeyJSON != "" || c.ServiceAccountKeyFile != ""
}

func (c *Config) ServiceAccountAudienceOrDefault() string {
	if c.ServiceAccountAudience != "" {
		return c.ServiceAccountAudience
	}
	return c.BaseURL
}

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
