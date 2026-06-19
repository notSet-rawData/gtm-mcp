package serviceauth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var jwtValidationClock = time.Now

const GoogleCertsURL = "https://www.googleapis.com/service_accounts/v1/jwk/_all@system.gserviceaccount.com"

const GoogleRSACertsURL = "https://www.googleapis.com/oauth2/v1/certs"

type SAJWTClaims struct {
	Issuer   string   `json:"iss"`
	Subject  string   `json:"sub"`
	Audience string   `json:"aud"`
	IssuedAt int64    `json:"iat"`
	Expires  int64    `json:"exp"`
	JTI      string   `json:"jti"` // JWT ID for replay protection
	Email    string   `json:"email,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
	KeyID     string `json:"kid"`
}

type Validator struct {
	mu         sync.RWMutex
	certCache  map[string]*rsa.PublicKey // key ID → public key
	certExpiry time.Time
	allowedSAs map[string]bool // set of allowed SA emails
	audience   string          // expected audience (MCP server base URL)

	jtiSeen map[string]time.Time
	jtiMu   sync.Mutex

	httpClient *http.Client
}

type ValidatorConfig struct {
	AllowedSAs []string

	Audience string

	HTTPClient *http.Client
}

func NewValidator(cfg ValidatorConfig) *Validator {
	allowed := make(map[string]bool, len(cfg.AllowedSAs))
	for _, sa := range cfg.AllowedSAs {
		allowed[strings.ToLower(sa)] = true
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &Validator{
		certCache:  make(map[string]*rsa.PublicKey),
		allowedSAs: allowed,
		audience:   cfg.Audience,
		jtiSeen:    make(map[string]time.Time),
		httpClient: client,
	}
}

func (v *Validator) ValidateJWT(ctx context.Context, tokenStr string) (*SAJWTClaims, error) {
	header, claims, sig, sigInput, err := parseJWTParts(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("jwt: malformed token: %w", err)
	}

	if header.Algorithm != "RS256" {
		return nil, fmt.Errorf("jwt: unsupported algorithm %q, expected RS256", header.Algorithm)
	}

	issuer := strings.ToLower(claims.Issuer)
	if issuer == "" {
		return nil, errors.New("jwt: missing issuer (iss)")
	}

	if len(v.allowedSAs) > 0 && !v.allowedSAs[issuer] {
		return nil, errors.New("jwt: issuer not authorized")
	}

	if claims.Subject != claims.Issuer {
		return nil, errors.New("jwt: iss and sub must match for SA tokens")
	}

	if v.audience != "" && claims.Audience != v.audience {
		return nil, fmt.Errorf("jwt: audience mismatch: expected %q, got %q", v.audience, claims.Audience)
	}

	now := jwtValidationClock()
	if claims.Expires == 0 {
		return nil, errors.New("jwt: missing exp claim")
	}
	if now.After(time.Unix(claims.Expires, 0)) {
		return nil, errors.New("jwt: token expired")
	}
	if now.Before(time.Unix(claims.IssuedAt, 0).Add(-30 * time.Second)) {
		return nil, errors.New("jwt: token not yet valid (iat in future)")
	}

	maxTTL := time.Unix(claims.Expires, 0).Sub(time.Unix(claims.IssuedAt, 0))
	if maxTTL > 1*time.Hour {
		return nil, fmt.Errorf("jwt: token lifetime %v exceeds maximum 1 hour", maxTTL)
	}

	if err := v.checkAndRecordJTI(claims.JTI, time.Unix(claims.Expires, 0)); err != nil {
		return nil, err
	}

	pubKey, err := v.getPublicKey(ctx, header.KeyID, issuer)
	if err != nil {
		return nil, fmt.Errorf("jwt: failed to get public key: %w", err)
	}

	if err := verifyRSASHA256(sigInput, sig, pubKey); err != nil {
		return nil, fmt.Errorf("jwt: invalid signature: %w", err)
	}

	return claims, nil
}

func (v *Validator) checkAndRecordJTI(jti string, expiry time.Time) error {
	if jti == "" {
		return errors.New("jwt: missing jti claim (required for replay protection)")
	}

	v.jtiMu.Lock()
	defer v.jtiMu.Unlock()

	now := jwtValidationClock()
	for k, exp := range v.jtiSeen {
		if now.After(exp) {
			delete(v.jtiSeen, k)
		}
	}

	if _, seen := v.jtiSeen[jti]; seen {
		return errors.New("jwt: token already used (replay detected)")
	}

	v.jtiSeen[jti] = expiry
	return nil
}

func (v *Validator) getPublicKey(ctx context.Context, keyID, email string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	if time.Now().Before(v.certExpiry) {
		if key, ok := v.certCache[keyID]; ok {
			v.mu.RUnlock()
			return key, nil
		}
	}
	v.mu.RUnlock()

	v.mu.Lock()
	defer v.mu.Unlock()

	if time.Now().Before(v.certExpiry) {
		if key, ok := v.certCache[keyID]; ok {
			return key, nil
		}
	}

	if err := v.refreshCerts(ctx, email); err != nil {
		return nil, err
	}

	key, ok := v.certCache[keyID]
	if !ok {
		return nil, fmt.Errorf("no public key found for key ID %q", keyID)
	}
	return key, nil
}

func (v *Validator) refreshCerts(ctx context.Context, email string) error {
	certURL := fmt.Sprintf("https://www.googleapis.com/service_accounts/v1/metadata/x509/%s", email)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, certURL, nil)
	if err != nil {
		return fmt.Errorf("failed to build cert request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch SA certs from %s: %w", certURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Google SA certs endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024)) // max 64KB
	if err != nil {
		return fmt.Errorf("failed to read SA certs response: %w", err)
	}

	var certs map[string]string
	if err := json.Unmarshal(body, &certs); err != nil {
		return fmt.Errorf("failed to parse SA certs response: %w", err)
	}

	newCache := make(map[string]*rsa.PublicKey, len(certs))
	for kid, certPEM := range certs {
		pubKey, err := parseX509Certificate([]byte(certPEM))
		if err != nil {
			continue
		}
		newCache[kid] = pubKey
	}

	v.certCache = newCache
	v.certExpiry = time.Now().Add(1 * time.Hour)

	return nil
}

func parseJWTParts(tokenStr string) (*jwtHeader, *SAJWTClaims, []byte, string, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, nil, nil, "", errors.New("invalid JWT format: expected 3 parts")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to decode header: %w", err)
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to decode claims: %w", err)
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to decode signature: %w", err)
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to parse header JSON: %w", err)
	}

	var claims SAJWTClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, nil, nil, "", fmt.Errorf("failed to parse claims JSON: %w", err)
	}

	sigInput := parts[0] + "." + parts[1]

	return &header, &claims, sig, sigInput, nil
}

func verifyRSASHA256(signingInput string, sig []byte, pubKey *rsa.PublicKey) error {
	h := sha256.New()
	h.Write([]byte(signingInput))
	digest := h.Sum(nil)
	return rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, digest, sig)
}
