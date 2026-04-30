package domain

import "time"

type HealthStatus string

const (
	HealthScaling         HealthStatus = "SCALING"
	HealthHealthy         HealthStatus = "HEALTHY"
	HealthAtRisk          HealthStatus = "AT_RISK"
	HealthUnderperforming HealthStatus = "UNDERPERFORMING"
)

type Campaign struct {
	ID              string       `json:"id"`
	MetaCampaignID  string       `json:"meta_campaign_id"`
	UserID          string       `json:"user_id"`
	AdAccountID     string       `json:"ad_account_id"`
	Name            string       `json:"name"`
	Objective       string       `json:"objective"`
	Status          string       `json:"status"`
	DailyBudget     *float64     `json:"daily_budget,omitempty"`
	LifetimeBudget  *float64     `json:"lifetime_budget,omitempty"`
	HealthStatus    HealthStatus `json:"health_status"`
	LastSyncedAt    *time.Time   `json:"last_synced_at,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type AdSet struct {
	ID              string     `json:"id"`
	MetaAdSetID     string     `json:"meta_ad_set_id"`
	CampaignID      string     `json:"campaign_id"`
	Name            string     `json:"name"`
	Status          string     `json:"status"`
	DailyBudget     *float64   `json:"daily_budget,omitempty"`
	OptimizationGoal string    `json:"optimization_goal,omitempty"`
	BillingEvent    string     `json:"billing_event,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Ad struct {
	ID            string    `json:"id"`
	MetaAdID      string    `json:"meta_ad_id"`
	AdSetID       string    `json:"ad_set_id"`
	Name          string    `json:"name"`
	Status        string    `json:"status"`
	CreativeTitle string    `json:"creative_title,omitempty"`
	CreativeBody  string    `json:"creative_body,omitempty"`
	ImageURL      string    `json:"image_url,omitempty"`
	CTAType       string    `json:"cta_type,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CampaignInsight struct {
	ID          string    `json:"id"`
	CampaignID  string    `json:"campaign_id"`
	Date        time.Time `json:"date"`
	Spend       float64   `json:"spend"`
	Impressions int64     `json:"impressions"`
	Clicks      int64     `json:"clicks"`
	CTR         float64   `json:"ctr"`
	CPC         float64   `json:"cpc"`
	CPM         float64   `json:"cpm"`
	Reach       int64     `json:"reach"`
	Frequency   float64   `json:"frequency"`
	Leads       int64     `json:"leads"`
	Purchases   int64     `json:"purchases"`
	ROAS        float64   `json:"roas"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
