package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gtm-mcp-server/auth"
	"golang.org/x/oauth2"

	_ "modernc.org/sqlite"
)

type SQLiteTokenStore struct {
	db            *sql.DB
	encryptionKey []byte // AES-256 key for encrypting token data at rest
	cancel        context.CancelFunc
}

// NewSQLiteTokenStore creates a new SQLite token store with encryption.
// The encryptionKey should be 32 bytes (use auth.DeriveKey to generate from a secret).
// If encryptionKey is nil, tokens are stored in plaintext (not recommended for production).
func NewSQLiteTokenStore(dbPath string, encryptionKey []byte) (*SQLiteTokenStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := initSchema(db); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	store := &SQLiteTokenStore{
		db:            db,
		encryptionKey: encryptionKey,
		cancel:        cancel,
	}

	go store.cleanup(ctx)

	return store, nil
}

func initSchema(db *sql.DB) error {
	// Enable WAL mode for better concurrent read/write performance
	pragmas := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA busy_timeout=5000`,
		`PRAGMA synchronous=NORMAL`,
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return err
		}
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS tokens (
			access_token TEXT PRIMARY KEY,
			refresh_token TEXT,
			info TEXT,
			expires_at DATETIME,
			refresh_expires_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_refresh ON tokens(refresh_token)`,
		
		`CREATE TABLE IF NOT EXISTS states (
			state_value TEXT PRIMARY KEY,
			created_at DATETIME,
			info TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_states_created ON states(created_at)`,

		`CREATE TABLE IF NOT EXISTS clients (
			client_id TEXT PRIMARY KEY,
			created_at DATETIME,
			info TEXT
		)`,
	}
	
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteTokenStore) Close() error {
	s.cancel()
	return s.db.Close()
}

// encrypt wraps auth.Encrypt if a key is configured, otherwise returns plaintext.
func (s *SQLiteTokenStore) encrypt(plaintext string) (string, error) {
	if s.encryptionKey == nil {
		return plaintext, nil
	}
	return auth.Encrypt(plaintext, s.encryptionKey)
}

// decrypt wraps auth.Decrypt if a key is configured, otherwise returns as-is.
func (s *SQLiteTokenStore) decrypt(ciphertext string) (string, error) {
	if s.encryptionKey == nil {
		return ciphertext, nil
	}
	return auth.Decrypt(ciphertext, s.encryptionKey)
}

func (s *SQLiteTokenStore) StoreToken(info *auth.TokenInfo) error {
	b, err := json.Marshal(info)
	if err != nil {
		return err
	}

	// Encrypt the token info before storing
	encrypted, err := s.encrypt(string(b))
	if err != nil {
		return err
	}
	
	refreshExpiresAt := sql.NullTime{Time: info.RefreshExpiresAt, Valid: !info.RefreshExpiresAt.IsZero()}
	
	_, err = s.db.Exec(`
		INSERT INTO tokens (access_token, refresh_token, info, expires_at, refresh_expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(access_token) DO UPDATE SET
			refresh_token=excluded.refresh_token,
			info=excluded.info,
			expires_at=excluded.expires_at,
			refresh_expires_at=excluded.refresh_expires_at
	`, info.AccessToken, info.RefreshToken, encrypted, info.ExpiresAt, refreshExpiresAt)
	return err
}

func (s *SQLiteTokenStore) GetTokenByAccess(accessToken string) (*auth.TokenInfo, error) {
	var infoStr string
	var expiresAt time.Time
	err := s.db.QueryRow(`SELECT info, expires_at FROM tokens WHERE access_token = ?`, accessToken).Scan(&infoStr, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, auth.ErrTokenNotFound
	} else if err != nil {
		return nil, err
	}

	if time.Now().After(expiresAt) {
		return nil, auth.ErrTokenExpired
	}

	// Decrypt the token info
	decrypted, err := s.decrypt(infoStr)
	if err != nil {
		return nil, err
	}

	var info auth.TokenInfo
	if err := json.Unmarshal([]byte(decrypted), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *SQLiteTokenStore) GetTokenByAccessIncludeExpired(accessToken string) (*auth.TokenInfo, error) {
	var infoStr string
	err := s.db.QueryRow(`SELECT info FROM tokens WHERE access_token = ?`, accessToken).Scan(&infoStr)
	if err == sql.ErrNoRows {
		return nil, auth.ErrTokenNotFound
	} else if err != nil {
		return nil, err
	}

	decrypted, err := s.decrypt(infoStr)
	if err != nil {
		return nil, err
	}

	var info auth.TokenInfo
	if err := json.Unmarshal([]byte(decrypted), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *SQLiteTokenStore) GetTokenByRefresh(refreshToken string) (*auth.TokenInfo, error) {
	var infoStr string
	var refreshExpiresAt sql.NullTime
	err := s.db.QueryRow(`SELECT info, refresh_expires_at FROM tokens WHERE refresh_token = ?`, refreshToken).Scan(&infoStr, &refreshExpiresAt)
	if err == sql.ErrNoRows {
		return nil, auth.ErrTokenNotFound
	} else if err != nil {
		return nil, err
	}

	if refreshExpiresAt.Valid && time.Now().After(refreshExpiresAt.Time) {
		return nil, auth.ErrTokenExpired
	}

	decrypted, err := s.decrypt(infoStr)
	if err != nil {
		return nil, err
	}

	var info auth.TokenInfo
	if err := json.Unmarshal([]byte(decrypted), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *SQLiteTokenStore) DeleteToken(accessToken string) error {
	_, err := s.db.Exec(`DELETE FROM tokens WHERE access_token = ?`, accessToken)
	return err
}

// RotateToken atomically replaces an old token with a new one using a SQL transaction.
// Inserts the new token first, then deletes the old one — prevents data loss on crash.
func (s *SQLiteTokenStore) RotateToken(oldAccessToken string, newToken *auth.TokenInfo) error {
	b, err := json.Marshal(newToken)
	if err != nil {
		return fmt.Errorf("failed to marshal new token: %w", err)
	}

	encrypted, err := s.encrypt(string(b))
	if err != nil {
		return fmt.Errorf("failed to encrypt new token: %w", err)
	}

	refreshExpiresAt := sql.NullTime{Time: newToken.RefreshExpiresAt, Valid: !newToken.RefreshExpiresAt.IsZero()}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // no-op if committed

	// Insert new token first
	_, err = tx.Exec(`
		INSERT INTO tokens (access_token, refresh_token, info, expires_at, refresh_expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(access_token) DO UPDATE SET
			refresh_token=excluded.refresh_token,
			info=excluded.info,
			expires_at=excluded.expires_at,
			refresh_expires_at=excluded.refresh_expires_at
	`, newToken.AccessToken, newToken.RefreshToken, encrypted, newToken.ExpiresAt, refreshExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to store new token: %w", err)
	}

	// Then delete old token
	_, err = tx.Exec(`DELETE FROM tokens WHERE access_token = ?`, oldAccessToken)
	if err != nil {
		return fmt.Errorf("failed to delete old token: %w", err)
	}

	return tx.Commit()
}

func (s *SQLiteTokenStore) UpdateGoogleToken(accessToken string, googleToken *oauth2.Token) error {
	if googleToken == nil {
		return auth.ErrTokenNotFound
	}
	info, err := s.GetTokenByAccessIncludeExpired(accessToken)
	if err != nil {
		return err
	}
	info.GoogleToken = googleToken
	return s.StoreToken(info)
}

func (s *SQLiteTokenStore) StoreState(state *auth.AuthState) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO states (state_value, created_at, info)
		VALUES (?, ?, ?)
		ON CONFLICT(state_value) DO UPDATE SET info=excluded.info, created_at=excluded.created_at
	`, state.State, state.CreatedAt, string(b))
	return err
}

func (s *SQLiteTokenStore) GetState(stateValue string) (*auth.AuthState, error) {
	var infoStr string
	var createdAt time.Time
	err := s.db.QueryRow(`SELECT info, created_at FROM states WHERE state_value = ?`, stateValue).Scan(&infoStr, &createdAt)
	if err == sql.ErrNoRows {
		return nil, auth.ErrInvalidState
	} else if err != nil {
		return nil, err
	}

	if time.Since(createdAt) > 10*time.Minute {
		return nil, auth.ErrInvalidState
	}

	var state auth.AuthState
	if err := json.Unmarshal([]byte(infoStr), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *SQLiteTokenStore) ConsumeState(stateValue string) (*auth.AuthState, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var infoStr string
	var createdAt time.Time
	err = tx.QueryRow(`SELECT info, created_at FROM states WHERE state_value = ?`, stateValue).Scan(&infoStr, &createdAt)
	if err == sql.ErrNoRows {
		return nil, auth.ErrInvalidState
	} else if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`DELETE FROM states WHERE state_value = ?`, stateValue)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if time.Since(createdAt) > 10*time.Minute {
		return nil, auth.ErrInvalidState
	}

	var state auth.AuthState
	if err := json.Unmarshal([]byte(infoStr), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *SQLiteTokenStore) DeleteState(stateValue string) error {
	_, err := s.db.Exec(`DELETE FROM states WHERE state_value = ?`, stateValue)
	return err
}

func (s *SQLiteTokenStore) StoreClient(c *auth.ClientInfo) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		INSERT INTO clients (client_id, created_at, info)
		VALUES (?, ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET info=excluded.info, created_at=excluded.created_at
	`, c.ClientID, c.CreatedAt, string(b))
	return err
}

func (s *SQLiteTokenStore) GetClient(clientID string) (*auth.ClientInfo, error) {
	var infoStr string
	err := s.db.QueryRow(`SELECT info FROM clients WHERE client_id = ?`, clientID).Scan(&infoStr)
	if err == sql.ErrNoRows {
		return nil, auth.ErrClientNotFound
	} else if err != nil {
		return nil, err
	}

	var c auth.ClientInfo
	if err := json.Unmarshal([]byte(infoStr), &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *SQLiteTokenStore) DeleteClient(clientID string) error {
	_, err := s.db.Exec(`DELETE FROM clients WHERE client_id = ?`, clientID)
	return err
}

func (s *SQLiteTokenStore) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Delete old tokens
			s.db.Exec(`DELETE FROM tokens WHERE 
				expires_at < datetime('now', '-1 hours') AND 
				(refresh_expires_at IS NULL OR refresh_expires_at < datetime('now'))`)
			
			// Delete old states
			s.db.Exec(`DELETE FROM states WHERE created_at < datetime('now', '-10 minutes')`)
			
			// Evict old clients if > 1000
			s.db.Exec(`
				DELETE FROM clients WHERE client_id NOT IN (
					SELECT client_id FROM clients ORDER BY created_at DESC LIMIT 1000
				)
			`)

		case <-ctx.Done():
			return
		}
	}
}
