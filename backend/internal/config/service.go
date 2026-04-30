package config

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service loads all config from system_config table.
// .env has only DATABASE_URL — everything else lives here.
type Service struct {
	db          *pgxpool.Pool
	encKey      []byte // 32-byte AES-256 key, derived from DB master key
	cache       map[string]entry
	mu          sync.RWMutex
	refreshRate time.Duration
}

type entry struct {
	value    string
	isSecret bool
}

func NewService(db *pgxpool.Pool, masterKey string) (*Service, error) {
	if len(masterKey) == 0 {
		return nil, fmt.Errorf("config: master encryption key is empty")
	}
	key := deriveKey(masterKey)
	s := &Service{
		db:          db,
		encKey:      key,
		cache:       make(map[string]entry),
		refreshRate: 5 * time.Minute,
	}
	if err := s.load(context.Background()); err != nil {
		return nil, fmt.Errorf("config: initial load failed: %w", err)
	}
	go s.refreshLoop()
	return s, nil
}

// Get returns a plain-text config value. Panics if key is missing.
func (s *Service) Get(key string) string {
	s.mu.RLock()
	e, ok := s.cache[key]
	s.mu.RUnlock()
	if !ok {
		return ""
	}
	return e.value
}

// GetSecret returns a decrypted secret value.
func (s *Service) GetSecret(key string) string {
	s.mu.RLock()
	e, ok := s.cache[key]
	s.mu.RUnlock()
	if !ok {
		return ""
	}
	if !e.isSecret {
		return e.value
	}
	plain, err := s.decrypt(e.value)
	if err != nil {
		slog.Error("config: failed to decrypt secret", "key", key, "err", err)
		return ""
	}
	return plain
}

// Set upserts a config value. Encrypts if isSecret is true.
func (s *Service) Set(ctx context.Context, key, value string, isSecret bool) error {
	stored := value
	if isSecret {
		enc, err := s.encrypt(value)
		if err != nil {
			return fmt.Errorf("config: encrypt %s: %w", key, err)
		}
		stored = enc
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO system_config (key, value, is_secret, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (key) DO UPDATE
		  SET value = EXCLUDED.value,
		      is_secret = EXCLUDED.is_secret,
		      updated_at = now()
	`, key, stored, isSecret)
	if err != nil {
		return fmt.Errorf("config: upsert %s: %w", key, err)
	}
	s.mu.Lock()
	s.cache[key] = entry{value: stored, isSecret: isSecret}
	s.mu.Unlock()
	return nil
}

// List returns all keys with masked secret values (for admin API).
func (s *Service) List(ctx context.Context) ([]ConfigEntry, error) {
	rows, err := s.db.Query(ctx, `
		SELECT key, value, is_secret, description, updated_at FROM system_config ORDER BY key
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ConfigEntry
	for rows.Next() {
		var e ConfigEntry
		if err := rows.Scan(&e.Key, &e.Value, &e.IsSecret, &e.Description, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if e.IsSecret {
			e.Value = "****"
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

type ConfigEntry struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	IsSecret    bool      `json:"is_secret"`
	Description *string   `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (s *Service) load(ctx context.Context) error {
	rows, err := s.db.Query(ctx, `SELECT key, value, is_secret FROM system_config`)
	if err != nil {
		return err
	}
	defer rows.Close()

	newCache := make(map[string]entry)
	for rows.Next() {
		var key, value string
		var isSecret bool
		if err := rows.Scan(&key, &value, &isSecret); err != nil {
			return err
		}
		newCache[key] = entry{value: value, isSecret: isSecret}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.cache = newCache
	s.mu.Unlock()
	return nil
}

func (s *Service) refreshLoop() {
	ticker := time.NewTicker(s.refreshRate)
	defer ticker.Stop()
	for range ticker.C {
		if err := s.load(context.Background()); err != nil {
			slog.Error("config: refresh failed", "err", err)
		}
	}
}

// encrypt uses AES-256-GCM. Returns base64(nonce+ciphertext).
func (s *Service) encrypt(plain string) (string, error) {
	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (s *Service) decrypt(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.encKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// deriveKey pads/truncates masterKey to exactly 32 bytes for AES-256.
func deriveKey(masterKey string) []byte {
	key := make([]byte, 32)
	copy(key, []byte(masterKey))
	return key
}
