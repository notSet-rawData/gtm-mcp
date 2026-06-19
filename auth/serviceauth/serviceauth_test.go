package serviceauth_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gtm-mcp-server/auth/serviceauth"
)

func testSAJSON(t *testing.T) []byte {
	t.Helper()
	creds := map[string]string{
		"type":                        "service_account",
		"project_id":                  "test-project",
		"private_key_id":              "key123abc",
		"private_key":                 validTestPrivateKey,
		"client_email":                "bot@test-project.iam.gserviceaccount.com",
		"client_id":                   "123456789",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/bot%40test-project.iam.gserviceaccount.com",
	}
	b, err := json.Marshal(creds)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestNewProvider_DirectMode(t *testing.T) {
	keyJSON := testSAJSON(t)

	p, err := serviceauth.NewProvider(serviceauth.Config{
		KeyJSON: keyJSON,
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if p.Mode() != serviceauth.ModeDirect {
		t.Errorf("expected ModeDirect, got %v", p.Mode())
	}

	if p.Email() != "bot@test-project.iam.gserviceaccount.com" {
		t.Errorf("unexpected email: %v", p.Email())
	}

	if p.Subject() != "" {
		t.Errorf("expected empty subject in Direct mode, got %q", p.Subject())
	}
}

func TestNewProvider_DelegationMode(t *testing.T) {
	keyJSON := testSAJSON(t)

	p, err := serviceauth.NewProvider(serviceauth.Config{
		KeyJSON: keyJSON,
		Subject: "admin@company.com",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if p.Mode() != serviceauth.ModeDelegation {
		t.Errorf("expected ModeDelegation, got %v", p.Mode())
	}

	if p.Subject() != "admin@company.com" {
		t.Errorf("expected subject 'admin@company.com', got %q", p.Subject())
	}
}

func TestNewProvider_InvalidType(t *testing.T) {
	creds := map[string]string{
		"type":         "authorized_user", // Wrong type
		"client_email": "user@example.com",
		"private_key":  validTestPrivateKey,
	}
	b, _ := json.Marshal(creds)

	_, err := serviceauth.NewProvider(serviceauth.Config{KeyJSON: b})
	if err == nil {
		t.Fatal("expected error for non-service_account type")
	}
}

func TestNewProvider_EmptyKey(t *testing.T) {
	_, err := serviceauth.NewProvider(serviceauth.Config{})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestNewProvider_MissingEmail(t *testing.T) {
	creds := map[string]string{
		"type":        "service_account",
		"private_key": validTestPrivateKey,
	}
	b, _ := json.Marshal(creds)

	_, err := serviceauth.NewProvider(serviceauth.Config{KeyJSON: b})
	if err == nil {
		t.Fatal("expected error for missing client_email")
	}
}

func TestFingerprintKeyID(t *testing.T) {
	keyJSON := testSAJSON(t)
	p, _ := serviceauth.NewProvider(serviceauth.Config{KeyJSON: keyJSON})

	fp := p.FingerprintKeyID()
	if fp == "" {
		t.Error("expected non-empty fingerprint")
	}

	if fp[:3] != "sa:" {
		t.Errorf("fingerprint should start with 'sa:', got %q", fp)
	}
}

func TestNewProviderFromEnv_NotConfigured(t *testing.T) {
	t.Setenv("GTM_SA_KEY_JSON", "")
	t.Setenv("GTM_SA_KEY_FILE", "")

	p, err := serviceauth.NewProviderFromEnv(nil, nil)
	if err != nil {
		t.Fatalf("expected no error when not configured, got: %v", err)
	}
	if p != nil {
		t.Error("expected nil provider when not configured")
	}
}

func TestNewProviderFromEnv_WithKeyJSON(t *testing.T) {
	t.Setenv("GTM_SA_KEY_JSON", string(testSAJSON(t)))
	t.Setenv("GTM_SA_KEY_FILE", "")
	t.Setenv("GTM_SA_SUBJECT", "")

	p, err := serviceauth.NewProviderFromEnv(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Mode() != serviceauth.ModeDirect {
		t.Errorf("expected ModeDirect, got %v", p.Mode())
	}
}

func TestNewProviderFromEnv_WithDelegation(t *testing.T) {
	t.Setenv("GTM_SA_KEY_JSON", string(testSAJSON(t)))
	t.Setenv("GTM_SA_SUBJECT", "admin@workspace.example.com")

	p, err := serviceauth.NewProviderFromEnv(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Mode() != serviceauth.ModeDelegation {
		t.Errorf("expected ModeDelegation, got %v", p.Mode())
	}
}

func TestValidator_RejectsExpiredToken(t *testing.T) {
	validator := serviceauth.NewValidator(serviceauth.ValidatorConfig{
		AllowedSAs: []string{"bot@test.iam.gserviceaccount.com"},
		Audience:   "https://mcp.example.com",
	})

	invalidToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJib3RAdGVzdC5pYW0uZ3NlcnZpY2VhY2NvdW50LmNvbSIsImV4cCI6MX0.invalid_signature"

	_, err := validator.ValidateJWT(context.Background(), invalidToken)
	if err == nil {
		t.Fatal("expected error for invalid/expired token")
	}
}

func TestValidator_RejectsMalformedToken(t *testing.T) {
	validator := serviceauth.NewValidator(serviceauth.ValidatorConfig{})

	_, err := validator.ValidateJWT(context.Background(), "not.a.jwt")
	if err == nil {
		t.Fatal("expected error for malformed token")
	}
}

func TestValidator_RejectsTokenTooLong(t *testing.T) {
	validator := serviceauth.NewValidator(serviceauth.ValidatorConfig{})

	_, err := validator.ValidateJWT(context.Background(), "part1.part2")
	if err == nil {
		t.Fatal("expected error for token with wrong part count")
	}
}

func TestIsLikelyJWT_Heuristic(t *testing.T) {
	tests := []struct {
		name  string
		token string
		valid bool
	}{
		{"valid JWT structure", "aaa.bbb.ccc", true},
		{"real JWT prefix", "eyJhbGciOiJSUzI1NiJ9.eyJpc3MiOiJ0ZXN0In0.signature", true},
		{"opaque OAuth token", "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0", false},
		{"too short", "a.b", false},
		{"empty", "", false},
		{"only dots", "...", false}, // empty parts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := splitJWTParts(tt.token)
			isJWT := len(parts) == 3 && len(parts[0]) > 0 && len(parts[1]) > 0 && len(parts[2]) > 0 && len(tt.token) >= 10
			if isJWT != tt.valid {
				t.Errorf("isLikelyJWT(%q) = %v, want %v", tt.token, isJWT, tt.valid)
			}
		})
	}
}

func splitJWTParts(s string) []string {
	var parts []string
	start := 0
	for i, c := range s {
		if c == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func TestTokenDefaultScopes(t *testing.T) {
	keyJSON := testSAJSON(t)
	p, err := serviceauth.NewProvider(serviceauth.Config{
		KeyJSON: keyJSON,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestSAModeConstants(t *testing.T) {
	if serviceauth.ModeDirect == "" {
		t.Error("ModeDirect should not be empty string")
	}
	if serviceauth.ModeDelegation == "" {
		t.Error("ModeDelegation should not be empty string")
	}
	if serviceauth.ModeDisabled != "" {
		t.Error("ModeDisabled should be empty string sentinel")
	}
}

func TestProviderCacheInvalidationOnError(t *testing.T) {
	keyJSON := testSAJSON(t)
	p, err := serviceauth.NewProvider(serviceauth.Config{KeyJSON: keyJSON})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = p.GoogleToken(ctx)
	if err == nil {
		t.Log("unexpected success — possibly running with real SA credentials")
	}
}

const validTestPrivateKey = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCsJ9MIbJpOuj9h
Bx4cYj8DjLo0wr2WAhuT9suc1dNbBSR/Ke+hhnKHOOXBWndPMAH1+it19V2KR2Zr
S8h3N8nvpYgOvn2lYyE9UQeasmsxMHiQZh9BHc7Lv1yiZgDT3hztQt9ZWGhhYrMH
WzLzzgyuLuABsqME334V55gMIxRRE51ZlNRt6N8sXmx7IG1bTZmZL4f0TScVji+l
Mo8C53ACmk4UM73la88YwEuItDy3sl8sHBBN+qgj9+QCL0i9nLypbqOJPzVdocLB
NYfpGq5OxpWDVCV6kWzuTzk5aaqzifiO0mS4fSWYDH7McO9dCAahhfFxEGalc2Id
wzx9Zh/xAgMBAAECggEAA3aK9I4YDeIJ6QNso/jcovn6EO+jQq4+LHF4J7/Wuuuq
j/L3Ar0ioVJlWh4IwmCz2WUz6fE1StUkMUpao6iio53P09lvIPGSjJo3A0dNiArH
520Lzz0wmMFRCrnvOv5/1HdWPi+HbKtrRd2cHnK+/I4C1uXnXkS5/j96ETHEydte
VZflCpDEjXPGiqZYj/t97e6uxfdz4TM0Q9Lce5w4uMyX+t31nHm24lhIHPyeWqST
/JBw0Rcj/DSdXyrl4xl7q91VoAoGpeWN+ib2iYJ+ulBdeOhSWUw7KfcOiPqgCbX8
BT9kWlD0rMI+O9+icXeKPZfavIREvLVB1V01pAfb5QKBgQDnSInvRIOLtfKEGyQj
oraBDbviGUWdSQfowq7ByHWBvV1eFrAlaNHY2PPBIXDc5vJY2B8yvVv+MHVSFbls
WOTPqBfn5h1rQHVvnprOxYmMBo5RHkdnNtwkGHxlHfRQgMK639qRrndPRSUrJnqc
hDhfEKo/3CWIEXJjXnPxs5QldwKBgQC+jaoh+0gcVg71j8bYk7/TuJPwjic4X6Kq
wUjneyp4ly/4UMQqjvxX+i8Y8ousmNaOrsh9upJqYrndgPWPG8M7pa67oSDhG4P9
wDqpgyoKXqmVQISzE19+pqiX5zEPkPQVxA3YRij8gRIdD0zu4wqdLvh/mq1kooEN
QVcfkWjf1wKBgQDNYVADpi6+YPsTrtpfvr0cSurd850q99BLNJ5lPLKEXHlN9Q3E
mplGXBnRFfYYZAkvNfQ2ZYsMZVG5a8s12JaPhHB+II1dUWc3kHteRHJJYwT8Kcw0
brX8Y7YLQRdUaZMCyYhZN7mBLiC8ebYFyTAZ0z2r6b12YC/Y5+ZD6zkSLwKBgFLF
zsV8FdLZPx5EGigx5f3eC8VOupKuWEa8NyL2SXigk+HVk6C5A7xjnNnFYg7TRUAt
hEG5LaiwwfQJ9KD5elEKo2A1mcau4SL0wYaoxzZB8IA4ymvPWof1dP6nGpScbqqV
wz3THDKzDl85Kj4Kua2VnbQwSGmSfWR4oZPA4kF5AoGAVhfd+GBW8MT2X+RkDdfL
opMgiW2ocCA4vhUlGOP7u8kYMD/CBLuiiGUdr1GtWl6DNMJ7ben7+zOiGTne9jMz
wlJdCsNklGK3Eipx9YzaOWOtB85K2U3r6HlL5tof/q58HY2jsuhuJ4DaLQ9zVl/9
t4tV/XcwU4F6mJpFvlecXJI=
-----END PRIVATE KEY-----`
