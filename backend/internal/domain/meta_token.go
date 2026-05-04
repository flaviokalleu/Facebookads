package domain

import "time"

// MetaTokenType enumerates the kinds of Meta access tokens we persist.
const (
	MetaTokenTypeUser       = "user"
	MetaTokenTypeSystemUser = "system_user"
	MetaTokenTypePage       = "page"
)

type MetaToken struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	AppID          string     `json:"app_id"`
	MetaUserID     string     `json:"meta_user_id"`
	EncryptedToken string     `json:"-"`
	// PlainToken is populated by the repo *only* when reading via decrypt.
	// Never persisted directly.
	PlainToken string     `json:"-"`
	TokenType  string     `json:"token_type"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastRefresh time.Time `json:"last_refresh"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
