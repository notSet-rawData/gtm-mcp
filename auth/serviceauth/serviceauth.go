package serviceauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

type ServiceAccountMode string

const (
	ModeDisabled ServiceAccountMode = ""

	ModeDirect ServiceAccountMode = "direct"

	ModeDelegation ServiceAccountMode = "delegation"
)

type SACredentials struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

type Provider struct {
	mu      sync.Mutex
	mode    ServiceAccountMode
	creds   *SACredentials
	subject string // Email to impersonate (delegation mode only)
	scopes  []string
	logger  *slog.Logger

	tokenSource oauth2.TokenSource
}

type Config struct {
	KeyJSON []byte

	KeyFile string

	Subject string

	Scopes []string

	Logger *slog.Logger
}

func NewProvider(cfg Config) (*Provider, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	keyJSON := cfg.KeyJSON
	if len(keyJSON) == 0 && cfg.KeyFile != "" {
		var err error
		keyJSON, err = os.ReadFile(cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("serviceauth: failed to read key file %q: %w", cfg.KeyFile, err)
		}
	}

	if len(keyJSON) == 0 {
		return nil, errors.New("serviceauth: no key provided (set KeyJSON or KeyFile)")
	}

	var creds SACredentials
	if err := json.Unmarshal(keyJSON, &creds); err != nil {
		return nil, fmt.Errorf("serviceauth: invalid key JSON: %w", err)
	}

	if creds.Type != "service_account" {
		return nil, fmt.Errorf("serviceauth: key type must be 'service_account', got %q", creds.Type)
	}

	if creds.ClientEmail == "" {
		return nil, errors.New("serviceauth: key missing client_email")
	}

	if creds.PrivateKey == "" {
		return nil, errors.New("serviceauth: key missing private_key")
	}

	mode := ModeDirect
	if cfg.Subject != "" {
		mode = ModeDelegation
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = defaultGTMScopes()
	}

	p := &Provider{
		mode:    mode,
		creds:   &creds,
		subject: cfg.Subject,
		scopes:  scopes,
		logger:  logger,
	}

	logger.Info("serviceauth: provider initialized",
		"mode", string(mode),
		"service_account", creds.ClientEmail,
		"subject", cfg.Subject,
		"scopes_count", len(scopes),
	)

	return p, nil
}

func NewProviderFromEnv(scopes []string, logger *slog.Logger) (*Provider, error) {
	keyJSON := os.Getenv("GTM_SA_KEY_JSON")
	keyFile := os.Getenv("GTM_SA_KEY_FILE")
	subject := os.Getenv("GTM_SA_SUBJECT")

	if keyJSON == "" && keyFile == "" {
		return nil, nil
	}

	var keyBytes []byte
	if keyJSON != "" {
		keyBytes = []byte(keyJSON)
	}

	return NewProvider(Config{
		KeyJSON: keyBytes,
		KeyFile: keyFile,
		Subject: subject,
		Scopes:  scopes,
		Logger:  logger,
	})
}

func (p *Provider) Mode() ServiceAccountMode {
	return p.mode
}

func (p *Provider) Email() string {
	return p.creds.ClientEmail
}

func (p *Provider) Subject() string {
	return p.subject
}

func (p *Provider) GoogleToken(ctx context.Context) (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tokenSource == nil {
		ts, err := p.buildTokenSource(ctx)
		if err != nil {
			return nil, err
		}
		p.tokenSource = ts
	}

	token, err := p.tokenSource.Token()
	if err != nil {
		p.tokenSource = nil
		return nil, fmt.Errorf("serviceauth: failed to get token: %w", err)
	}

	return token, nil
}

func (p *Provider) buildTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	jwtConfig := &jwt.Config{
		Email:      p.creds.ClientEmail,
		PrivateKey: []byte(p.creds.PrivateKey),
		Scopes:     p.scopes,
		TokenURL:   p.creds.TokenURI,
	}

	if jwtConfig.TokenURL == "" {
		jwtConfig.TokenURL = google.JWTTokenURL
	}

	if p.mode == ModeDelegation {
		jwtConfig.Subject = p.subject
	}

	baseSource := jwtConfig.TokenSource(ctx)
	return oauth2.ReuseTokenSource(nil, baseSource), nil
}

func (p *Provider) Validate(ctx context.Context) error {
	_, err := p.GoogleToken(ctx)
	if err != nil {
		return fmt.Errorf("serviceauth: credential validation failed for %s: %w", p.creds.ClientEmail, err)
	}
	p.logger.Info("serviceauth: credentials validated successfully",
		"service_account", p.creds.ClientEmail,
		"mode", string(p.mode),
	)
	return nil
}

func (p *Provider) FingerprintKeyID() string {
	keyID := p.creds.PrivateKeyID
	if len(keyID) > 8 {
		keyID = keyID[:8]
	}
	return fmt.Sprintf("sa:%s:%s", p.creds.ClientEmail, keyID)
}

type SATokenInfo struct {
	ServiceAccountEmail string

	Subject string

	Mode ServiceAccountMode

	Scopes []string

	Fingerprint string
}

func GenerateServiceAccountBearerToken(keyJSON []byte, audience string, ttl time.Duration) (string, error) {
	var creds SACredentials
	if err := json.Unmarshal(keyJSON, &creds); err != nil {
		return "", fmt.Errorf("invalid SA key JSON: %w", err)
	}

	now := time.Now()
	claims := map[string]interface{}{
		"iss": creds.ClientEmail,
		"sub": creds.ClientEmail,
		"aud": audience,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
		"jti": generateJTI(),
	}

	header := map[string]interface{}{
		"alg": "RS256",
		"typ": "JWT",
		"kid": creds.PrivateKeyID,
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := headerB64 + "." + claimsB64

	privKey, err := parseRSAPrivateKey([]byte(creds.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(unsigned))
	digest := h.Sum(nil)

	sig, err := rsa.SignPKCS1v15(rand.Reader, privKey, 0, digest)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func defaultGTMScopes() []string {
	return []string{
		"https://www.googleapis.com/auth/tagmanager.readonly",
		"https://www.googleapis.com/auth/tagmanager.delete.containers",
		"https://www.googleapis.com/auth/tagmanager.edit.containers",
		"https://www.googleapis.com/auth/tagmanager.edit.containerversions",
		"https://www.googleapis.com/auth/tagmanager.publish",
		"https://www.googleapis.com/auth/tagmanager.manage.users",
	}
}

func generateJTI() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func parseRSAPrivateKey(pemKey []byte) (*rsa.PrivateKey, error) {
	pemStr := string(pemKey)
	pemStr = strings.ReplaceAll(pemStr, "\\n", "\n")

	jwtConfig := &jwt.Config{
		Email:      "validate@example.com",
		PrivateKey: []byte(pemStr),
		Scopes:     []string{"https://www.googleapis.com/auth/tagmanager.readonly"},
		TokenURL:   google.JWTTokenURL,
	}

	ctx := context.Background()
	ts := jwtConfig.TokenSource(ctx)
	if ts == nil {
		return nil, errors.New("failed to parse RSA private key")
	}

	return parseRSAKeyFromPEM([]byte(pemStr))
}
