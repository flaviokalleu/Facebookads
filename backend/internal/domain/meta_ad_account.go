package domain

import "time"

const (
	AccessKindOwned    = "owned"
	AccessKindClient   = "client"
	AccessKindPersonal = "personal"
)

type MetaAdAccount struct {
	ID            string    `json:"id"`
	MetaID        string    `json:"meta_id"`
	BMID          *string   `json:"bm_id,omitempty"`
	UserID        string    `json:"user_id"`
	Name          string    `json:"name"`
	Currency      string    `json:"currency"`
	TimezoneName  string    `json:"timezone_name"`
	AccountStatus int       `json:"account_status"`
	DisableReason int       `json:"disable_reason"`
	SpendCap      float64   `json:"spend_cap"`
	AmountSpent   float64   `json:"amount_spent"`
	Balance       float64   `json:"balance"`
	AccessKind    string    `json:"access_kind"`
	Raw           []byte    `json:"-"`
	SyncedAt      time.Time `json:"synced_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
