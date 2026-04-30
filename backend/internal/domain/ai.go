package domain

import "time"

type AnomalySeverity string
type AnomalyType string

const (
	SeverityHigh   AnomalySeverity = "HIGH"
	SeverityMedium AnomalySeverity = "MEDIUM"
	SeverityLow    AnomalySeverity = "LOW"

	AnomalyCPCSpike          AnomalyType = "CPC_SPIKE"
	AnomalyCTRDrop           AnomalyType = "CTR_DROP"
	AnomalyCreativeFatigue   AnomalyType = "CREATIVE_FATIGUE"
	AnomalyBudgetWaste       AnomalyType = "BUDGET_WASTE"
	AnomalyAudienceSaturation AnomalyType = "AUDIENCE_SATURATION"
	AnomalyROASCollapse      AnomalyType = "ROAS_COLLAPSE"
	AnomalyDeliveryStall     AnomalyType = "DELIVERY_STALL"
)

type Anomaly struct {
	ID          string          `json:"id"`
	CampaignID  string          `json:"campaign_id"`
	Type        AnomalyType     `json:"type"`
	Severity    AnomalySeverity `json:"severity"`
	Description string          `json:"description"`
	IsActive    bool            `json:"is_active"`
	DetectedAt  time.Time       `json:"detected_at"`
	ResolvedAt  *time.Time      `json:"resolved_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type RecommendationPriority string
type RecommendationCategory string

const (
	PriorityHigh   RecommendationPriority = "HIGH"
	PriorityMedium RecommendationPriority = "MEDIUM"
	PriorityLow    RecommendationPriority = "LOW"

	CategoryBudget    RecommendationCategory = "BUDGET"
	CategoryTargeting RecommendationCategory = "TARGETING"
	CategoryCreative  RecommendationCategory = "CREATIVE"
	CategoryBidding   RecommendationCategory = "BIDDING"
	CategoryAudience  RecommendationCategory = "AUDIENCE"
	CategorySchedule  RecommendationCategory = "SCHEDULE"
)

type Recommendation struct {
	ID             string                 `json:"id"`
	CampaignID     string                 `json:"campaign_id"`
	Priority       RecommendationPriority `json:"priority"`
	Category       RecommendationCategory `json:"category"`
	Action         string                 `json:"action"`
	ExpectedImpact string                 `json:"expected_impact"`
	Rationale      string                 `json:"rationale"`
	ModelUsed      string                 `json:"model_used"`
	IsApplied      bool                   `json:"is_applied"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type BudgetSuggestion struct {
	ID                      string    `json:"id"`
	UserID                  string    `json:"user_id"`
	AdAccountID             string    `json:"ad_account_id"`
	CampaignID              *string   `json:"campaign_id,omitempty"`
	CampaignName            string    `json:"campaign_name,omitempty"`
	CurrentBudget           float64   `json:"current_budget"`
	SuggestedBudget         float64   `json:"suggested_budget"`
	SuggestedChange         float64   `json:"suggested_change"`
	ChangeReason            string    `json:"change_reason"`
	ShouldPause             bool      `json:"should_pause"`
	ExpectedROASImprovement string    `json:"expected_roas_improvement"`
	PortfolioSummary        string    `json:"portfolio_summary"`
	ModelUsed               string    `json:"model_used"`
	IsApplied               bool      `json:"is_applied"`
	CreatedAt               time.Time `json:"created_at"`
}

type LLMUsage struct {
	ID           string    `json:"id"`
	UserID       *string   `json:"user_id,omitempty"`
	TaskType     string    `json:"task_type"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	LatencyMs    int       `json:"latency_ms"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
