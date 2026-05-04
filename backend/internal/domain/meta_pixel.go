package domain

import "time"

type MetaPixel struct {
	ID        string     `json:"id"`
	MetaID    string     `json:"meta_id"`
	BMID      *string    `json:"bm_id,omitempty"`
	AccountID *string    `json:"account_id,omitempty"`
	UserID    string     `json:"user_id"`
	Name      string     `json:"name"`
	LastFired *time.Time `json:"last_fired,omitempty"`
	IsActive  bool       `json:"is_active"`
	Raw       []byte     `json:"-"`
	SyncedAt  time.Time  `json:"synced_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}
