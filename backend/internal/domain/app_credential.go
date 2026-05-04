package domain

import "time"

type AppCredential struct {
	ID                  string    `json:"id"`
	UserID              string    `json:"user_id"`
	AppID               string    `json:"app_id"`
	EncryptedAppSecret  string    `json:"-"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
