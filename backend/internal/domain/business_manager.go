package domain

import "time"

type BusinessManager struct {
	ID                 string    `json:"id"`
	MetaID             string    `json:"meta_id"`
	UserID             string    `json:"user_id"`
	Name               string    `json:"name"`
	VerificationStatus string    `json:"verification_status"`
	TimezoneID         int       `json:"timezone_id"`
	Vertical           string    `json:"vertical"`
	Raw                []byte    `json:"-"`
	SyncedAt           time.Time `json:"synced_at"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
