package domain

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserToken struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	AdAccountID    string     `json:"ad_account_id"`
	EncryptedToken string     `json:"-"`
	TokenExpiry    *time.Time `json:"token_expiry,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
