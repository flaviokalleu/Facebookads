package domain

import (
	"encoding/json"
	"time"
)

// AIAction status enum
const (
	AIActionStatusPending  = "pending"
	AIActionStatusApproved = "approved"
	AIActionStatusExecuted = "executed"
	AIActionStatusRejected = "rejected"
	AIActionStatusFailed   = "failed"
	AIActionStatusReverted = "reverted"
)

// AIAction action_type enum
const (
	AIActionTypePauseAd        = "pause_ad"
	AIActionTypePauseAdSet     = "pause_adset"
	AIActionTypeScaleBudget    = "scale_budget"
	AIActionTypeRotateCreative = "rotate_creative"
	AIActionTypeDuplicateAdSet = "duplicate_adset"
	AIActionTypeCreateCampaign = "create_campaign"
	AIActionTypeAlert          = "alert"
)

// AIAction target_kind enum
const (
	AIActionTargetAd       = "ad"
	AIActionTargetAdSet    = "adset"
	AIActionTargetCampaign = "campaign"
)

// AIAction source enum
const (
	AIActionSourceRules    = "rules"
	AIActionSourceDeepSeek = "deepseek"
)

// AIAction mode enum
const (
	AIActionModeAuto    = "auto"
	AIActionModePropose = "propose"
)

type AIAction struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	AccountMetaID  string          `json:"account_meta_id"`
	ActionType     string          `json:"action_type"`
	TargetMetaID   string          `json:"target_meta_id"`
	TargetKind     string          `json:"target_kind"`
	Reason         string          `json:"reason"`
	MetricSnapshot json.RawMessage `json:"metric_snapshot,omitempty"`
	ProposedChange json.RawMessage `json:"proposed_change,omitempty"`
	Source         string          `json:"source"`
	Mode           string          `json:"mode"`
	Status         string          `json:"status"`
	MetaResponse   json.RawMessage `json:"meta_response,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	DecidedAt      *time.Time      `json:"decided_at,omitempty"`
	ExecutedAt     *time.Time      `json:"executed_at,omitempty"`
}

// AISafetyRule overrides a default threshold for a user (or for one account).
type AISafetyRule struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	AccountMetaID *string   `json:"account_meta_id,omitempty"`
	RuleKey       string    `json:"rule_key"`
	RuleValue     float64   `json:"rule_value"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
