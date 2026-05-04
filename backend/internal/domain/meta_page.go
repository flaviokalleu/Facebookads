package domain

import "time"

type MetaPage struct {
	ID                 string    `json:"id"`
	MetaID             string    `json:"meta_id"`
	BMID               *string   `json:"bm_id,omitempty"`
	UserID             string    `json:"user_id"`
	Name               string    `json:"name"`
	Category           string    `json:"category"`
	FanCount           int64     `json:"fan_count"`
	EncryptedPageToken string    `json:"-"`
	IGUserID           string    `json:"ig_user_id"`
	Raw                []byte    `json:"-"`
	SyncedAt           time.Time `json:"synced_at"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type MetaInstagramAccount struct {
	ID         string    `json:"id"`
	MetaID     string    `json:"meta_id"`
	BMID       *string   `json:"bm_id,omitempty"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	ProfilePic string    `json:"profile_pic"`
	Raw        []byte    `json:"-"`
	SyncedAt   time.Time `json:"synced_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
