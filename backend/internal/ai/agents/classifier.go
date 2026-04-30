package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/domain"
)

const classifierPrompt = `You are a Meta Ads performance analyst.

Classify the health of this campaign based on the last 7 days of data.

Campaign: %s | Objective: %s
CTR: %.2f%% (account avg: %.2f%%)
CPC: $%.2f (account avg: $%.2f)
ROAS: %.2f | Spend: $%.2f | Leads: %d | Frequency: %.2f
CTR trend (last 3 days): %s
ROAS trend (last 3 days): %s

Respond ONLY with valid JSON:
{
  "health_status": "SCALING|HEALTHY|AT_RISK|UNDERPERFORMING",
  "confidence": 0.0-1.0,
  "reason": "one sentence explanation"
}`

type classifierResponse struct {
	HealthStatus string  `json:"health_status"`
	Confidence   float64 `json:"confidence"`
	Reason       string  `json:"reason"`
}

// Classifier classifies campaign health using the AI router.
// Primary model: deepseek-v4-pro (routed via TaskClassification).
type Classifier struct {
	db     *pgxpool.Pool
	router *ai.Router
}

func NewClassifier(db *pgxpool.Pool, router *ai.Router) *Classifier {
	return &Classifier{db: db, router: router}
}

// RunClassifyAll fetches all campaigns and classifies each one's health.
func (c *Classifier) RunClassifyAll(ctx context.Context, userID string) error {
	campaigns, err := c.listCampaigns(ctx, userID)
	if err != nil {
		return fmt.Errorf("classifier: list campaigns: %w", err)
	}
	if len(campaigns) == 0 {
		slog.Info("classifier: no campaigns to classify", "user_id", userID)
		return nil
	}

	avgCTR, avgCPC, err := c.getAccountAverages(ctx, userID)
	if err != nil {
		slog.Warn("classifier: account averages unavailable", "user_id", userID, "err", err)
	}

	for _, camp := range campaigns {
		if err := c.classifyOne(ctx, camp, avgCTR, avgCPC); err != nil {
			slog.Error("classifier: failed for campaign", "campaign_id", camp.ID, "name", camp.Name, "err", err)
			continue
		}
		slog.Info("classifier: campaign classified", "campaign_id", camp.ID, "health", camp.HealthStatus)
	}
	return nil
}

func (c *Classifier) classifyOne(ctx context.Context, camp *domain.Campaign, avgCTR, avgCPC float64) error {
	to := time.Now().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -7)
	insights, err := c.listInsights(ctx, camp.ID, from, to)
	if err != nil {
		return fmt.Errorf("list insights: %w", err)
	}
	if len(insights) == 0 {
		return nil
	}

	ctrAvg, cpcAvg, roasAvg, spendTotal, leadsTotal, freqAvg := aggregateInsights(insights)
	ctrTrend := trendString(insights, func(i *domain.CampaignInsight) float64 { return i.CTR }, 3)
	roasTrend := trendString(insights, func(i *domain.CampaignInsight) float64 { return i.ROAS }, 3)

	userPrompt := fmt.Sprintf(classifierPrompt,
		camp.Name, camp.Objective,
		ctrAvg, avgCTR,
		cpcAvg, avgCPC,
		roasAvg, spendTotal, int64(leadsTotal), freqAvg,
		ctrTrend, roasTrend,
	)

	resp, err := c.router.Complete(ctx, ai.TaskClassification, ai.CompletionRequest{
		SystemPrompt: "You are a Meta Ads performance analyst. Respond only with valid JSON.",
		UserPrompt:   userPrompt,
		MaxTokens:    200,
		Temperature:  0.1,
		JSONMode:     true,
	})
	if err != nil {
		return fmt.Errorf("router: %w", err)
	}

	var out classifierResponse
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &out); err != nil {
		return fmt.Errorf("parse response: %w (raw: %s)", err, truncate(resp.Content, 200))
	}

	hs := domain.HealthStatus(strings.ToUpper(out.HealthStatus))
	switch hs {
	case domain.HealthScaling, domain.HealthHealthy, domain.HealthAtRisk, domain.HealthUnderperforming:
		camp.HealthStatus = hs
	default:
		slog.Warn("classifier: unknown health status, defaulting to HEALTHY", "raw", out.HealthStatus)
		camp.HealthStatus = domain.HealthHealthy
	}

	if err := c.updateHealth(ctx, camp.ID, camp.HealthStatus); err != nil {
		return fmt.Errorf("update health: %w", err)
	}
	return nil
}

func (c *Classifier) listCampaigns(ctx context.Context, userID string) ([]*domain.Campaign, error) {
	rows, err := c.db.Query(ctx, `
		SELECT id, meta_campaign_id, user_id, ad_account_id, name, objective, status,
		       daily_budget, lifetime_budget, health_status, last_synced_at, created_at, updated_at
		FROM campaigns WHERE user_id = $1 AND deleted_at IS NULL
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Campaign
	for rows.Next() {
		var camp domain.Campaign
		var lastSynced *time.Time
		if err := rows.Scan(
			&camp.ID, &camp.MetaCampaignID, &camp.UserID, &camp.AdAccountID,
			&camp.Name, &camp.Objective, &camp.Status,
			&camp.DailyBudget, &camp.LifetimeBudget,
			&camp.HealthStatus, &lastSynced,
			&camp.CreatedAt, &camp.UpdatedAt,
		); err != nil {
			return nil, err
		}
		camp.LastSyncedAt = lastSynced
		result = append(result, &camp)
	}
	return result, rows.Err()
}

func (c *Classifier) listInsights(ctx context.Context, campaignID string, from, to time.Time) ([]*domain.CampaignInsight, error) {
	rows, err := c.db.Query(ctx, `
		SELECT id, campaign_id, date, spend, impressions, clicks, ctr, cpc, cpm,
		       reach, frequency, leads, purchases, roas, created_at, updated_at
		FROM campaign_insights
		WHERE campaign_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date
	`, campaignID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.CampaignInsight
	for rows.Next() {
		var i domain.CampaignInsight
		if err := rows.Scan(
			&i.ID, &i.CampaignID, &i.Date,
			&i.Spend, &i.Impressions, &i.Clicks, &i.CTR,
			&i.CPC, &i.CPM, &i.Reach, &i.Frequency,
			&i.Leads, &i.Purchases, &i.ROAS,
			&i.CreatedAt, &i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, &i)
	}
	return result, rows.Err()
}

func (c *Classifier) getAccountAverages(ctx context.Context, userID string) (avgCTR, avgCPC float64, err error) {
	err = c.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(ci.ctr), 0), COALESCE(AVG(ci.cpc), 0)
		FROM campaign_insights ci
		JOIN campaigns c2 ON c2.id = ci.campaign_id
		WHERE c2.user_id = $1 AND ci.date >= CURRENT_DATE - 7 AND c2.deleted_at IS NULL
	`, userID).Scan(&avgCTR, &avgCPC)
	return
}

func (c *Classifier) updateHealth(ctx context.Context, id string, status domain.HealthStatus) error {
	_, err := c.db.Exec(ctx, `UPDATE campaigns SET health_status = $1, updated_at = now() WHERE id = $2`, status, id)
	return err
}
