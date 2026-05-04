package repository

import (
	"context"
	"time"

	"github.com/facebookads/backend/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

type UserTokenRepository interface {
	Upsert(ctx context.Context, token *domain.UserToken) error
	GetByUserAndAccount(ctx context.Context, userID, adAccountID string) (*domain.UserToken, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.UserToken, error)
}

type CampaignRepository interface {
	Upsert(ctx context.Context, campaign *domain.Campaign) error
	GetByID(ctx context.Context, id string) (*domain.Campaign, error)
	GetByMetaID(ctx context.Context, userID, metaID string) (*domain.Campaign, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.Campaign, error)
	UpdateHealthStatus(ctx context.Context, id string, status domain.HealthStatus) error
	Update(ctx context.Context, campaign *domain.Campaign) error
	MarkDeleted(ctx context.Context, id string) error
}

type AdSetRepository interface {
	Upsert(ctx context.Context, adSet *domain.AdSet) error
	ListByCampaign(ctx context.Context, campaignID string) ([]*domain.AdSet, error)
}

type AdRepository interface {
	Upsert(ctx context.Context, ad *domain.Ad) error
	ListByAdSet(ctx context.Context, adSetID string) ([]*domain.Ad, error)
}

type InsightRepository interface {
	Upsert(ctx context.Context, insight *domain.CampaignInsight) error
	ListByCampaign(ctx context.Context, campaignID string, from, to time.Time) ([]*domain.CampaignInsight, error)
	GetAccountAverages(ctx context.Context, userID string, days int) (avgCTR, avgCPC float64, err error)
}

type AnomalyRepository interface {
	Create(ctx context.Context, anomaly *domain.Anomaly) error
	ListActive(ctx context.Context, userID string) ([]*domain.Anomaly, error)
	ListByCampaign(ctx context.Context, campaignID string) ([]*domain.Anomaly, error)
	Resolve(ctx context.Context, id string) error
}

type RecommendationRepository interface {
	BulkCreate(ctx context.Context, recs []*domain.Recommendation) error
	ListByCampaign(ctx context.Context, campaignID string) ([]*domain.Recommendation, error)
	MarkApplied(ctx context.Context, id string) error
}

type BudgetSuggestionRepository interface {
	BulkCreate(ctx context.Context, suggestions []*domain.BudgetSuggestion) error
	ListByUser(ctx context.Context, userID string) ([]*domain.BudgetSuggestion, error)
	MarkApplied(ctx context.Context, id string) error
}

type LLMUsageRepository interface {
	Create(ctx context.Context, usage *domain.LLMUsage) error
	SummaryByProvider(ctx context.Context, userID string, from, to time.Time) ([]LLMProviderSummary, error)
	DailyCost(ctx context.Context, userID string, days int) ([]LLMDailyCost, error)
}

type LLMProviderSummary struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	Requests     int     `json:"requests"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

type LLMDailyCost struct {
	Date     time.Time `json:"date"`
	Provider string    `json:"provider"`
	CostUSD  float64   `json:"cost_usd"`
}

// ─── Phase F1: Meta Business Manager hierarchy + Imovel catalog ───────────────

type AppCredentialRepository interface {
	Upsert(ctx context.Context, cred *domain.AppCredential) error
	GetByUserAndAppID(ctx context.Context, userID, appID string) (*domain.AppCredential, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.AppCredential, error)
}

type MetaTokenRepository interface {
	Upsert(ctx context.Context, token *domain.MetaToken) error
	GetActiveByUser(ctx context.Context, userID string) (*domain.MetaToken, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.MetaToken, error)
}

type BusinessManagerRepository interface {
	Upsert(ctx context.Context, bm *domain.BusinessManager) error
	ListByUser(ctx context.Context, userID string) ([]*domain.BusinessManager, error)
	GetByMetaID(ctx context.Context, metaID string) (*domain.BusinessManager, error)
}

type MetaAdAccountRepository interface {
	Upsert(ctx context.Context, account *domain.MetaAdAccount) error
	ListByUser(ctx context.Context, userID string) ([]*domain.MetaAdAccount, error)
	ListByBM(ctx context.Context, bmMetaID string) ([]*domain.MetaAdAccount, error)
	ListPersonalByUser(ctx context.Context, userID string) ([]*domain.MetaAdAccount, error)
}

type MetaPageRepository interface {
	Upsert(ctx context.Context, page *domain.MetaPage) error
	ListByUser(ctx context.Context, userID string) ([]*domain.MetaPage, error)
	ListByBM(ctx context.Context, bmMetaID string) ([]*domain.MetaPage, error)
}

type MetaPixelRepository interface {
	Upsert(ctx context.Context, pixel *domain.MetaPixel) error
	ListByUser(ctx context.Context, userID string) ([]*domain.MetaPixel, error)
	ListByBM(ctx context.Context, bmMetaID string) ([]*domain.MetaPixel, error)
}

type ImovelRepository interface {
	Create(ctx context.Context, imovel *domain.Imovel) error
	Update(ctx context.Context, imovel *domain.Imovel) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.Imovel, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.Imovel, error)
}

// ─── Phase F2/F5/F6: Autonomous AI optimization agent ─────────────────────────

type AIActionRepository interface {
	Create(ctx context.Context, action *domain.AIAction) error
	GetByID(ctx context.Context, id string) (*domain.AIAction, error)
	ListPendingByUser(ctx context.Context, userID string, limit int) ([]*domain.AIAction, error)
	ListByUser(ctx context.Context, userID, status string, limit int) ([]*domain.AIAction, error)
	MarkApproved(ctx context.Context, id string) error
	MarkRejected(ctx context.Context, id string) error
	MarkExecuted(ctx context.Context, id string, metaResponse []byte) error
	MarkFailed(ctx context.Context, id string, metaResponse []byte) error
	MarkReverted(ctx context.Context, id string) error
	CountPausesLast24h(ctx context.Context, userID, accountMetaID string) (int, error)
}

type AISafetyRuleRepository interface {
	Upsert(ctx context.Context, rule *domain.AISafetyRule) error
	ListByUser(ctx context.Context, userID string) ([]*domain.AISafetyRule, error)
}
