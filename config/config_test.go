package config

import (
	"os"
	"testing"
)

func TestValidate_Port(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"valid port 8080", 8080, false},
		{"valid port 1", 1, false},
		{"valid port 65535", 65535, false},
		{"port 0", 0, true},
		{"port negative", -1, true},
		{"port too high", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Port: tt.port, BaseURL: "http://localhost:8080"}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidate_BaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{"valid http", "http://localhost:8080", false},
		{"valid https", "https://gtm.example.com", false},
		{"empty (allowed)", "", false},
		{"no scheme", "localhost:8080", true},
		{"ftp scheme", "ftp://files.example.com", true},
		{"no host", "http://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{Port: 8080, BaseURL: tt.baseURL}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateAuth(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		jwtSecret    string
		wantErr      bool
	}{
		{"valid", "id", "secret", "this-is-a-jwt-secret-with-32-chars!", false},
		{"empty client ID", "", "secret", "this-is-a-jwt-secret-with-32-chars!", true},
		{"empty client secret", "id", "", "this-is-a-jwt-secret-with-32-chars!", true},
		{"empty JWT secret", "id", "secret", "", true},
		{"JWT secret too short", "id", "secret", "short", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GoogleClientID:     tt.clientID,
				GoogleClientSecret: tt.clientSecret,
				JWTSecret:          tt.jwtSecret,
			}
			err := cfg.ValidateAuth()
			if (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestGetEnvInt(t *testing.T) {
	got := getEnvInt("NONEXISTENT_VAR_12345", 42)
	if got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	os.Setenv("TEST_PORT_VAR", "3000")
	defer os.Unsetenv("TEST_PORT_VAR")

	got = getEnvInt("TEST_PORT_VAR", 42)
	if got != 3000 {
		t.Fatalf("expected 3000, got %d", got)
	}

	os.Setenv("TEST_PORT_BAD", "not-a-number")
	defer os.Unsetenv("TEST_PORT_BAD")

	got = getEnvInt("TEST_PORT_BAD", 42)
	if got != 42 {
		t.Fatalf("expected 42 on invalid, got %d", got)
	}
}
