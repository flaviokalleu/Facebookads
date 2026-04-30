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

const optimizerPrompt = `You are a senior Meta Ads media buyer managing 7-figure monthly budgets.

Analyze this campaign and provide specific, actionable recommendations.

Campaign: %s
Ad Sets: %s
Last 14 Days (daily): %s
Active Anomalies: %s

Respond in this exact JSON:
{
  "recommendations": [
    {
      "priority": "HIGH|MEDIUM|LOW",
      "category": "BUDGET|TARGETING|CREATIVE|BIDDING|AUDIENCE|SCHEDULE",
      "action": "specific action to take",
      "expected_impact": "improvement to expect",
      "rationale": "why this will help"
    }
  ],
  "overall_assessment": "2-3 sentence summary",
  "estimated_roas_improvement": "X%"
}`

type optimizerResponse struct {
	Recommendations []struct {
		Priority       string `json:"priority"`
		Category       string `json:"category"`
		Action         string `json:"action"`
		ExpectedImpact string `json:"expected_impact"`
		Rationale      string `json:"rationale"`
	} `json:"recommendations"`
	OverallAssessment        string `json:"overall_assessment"`
	EstimatedROASImprovement string `json:"estimated_roas_improvement"`
	ModelUsed                string `json:"model_used"`
}

// Optimizer generates optimization recommendations for at-risk and underperforming campaigns.
// Primary model: claude-opus-4-7 (routed via TaskOptimization).
type Optimizer struct {
	db     *pgxpool.Pool
	router *ai.Router
}

func NewOptimizer(db *pgxpool.Pool, router *ai.Router) *Optimizer {
	return &Optimizer{db: db, router: router}
}

// RunGenerateAll generates recommendations for all at-risk/underperforming campaigns.
func (o *Optimizer) RunGenerateAll(ctx context.Context, userID string) error {
	campaigns, err := o.listAtRisk(ctx, userID)
	if err != nil {
		return fmt.Errorf("optimizer: list at-risk: %w", err)
	}
	if len(campaigns) == 0 {
		slog.Info("optimizer: no at-risk or underperforming campaigns", "user_id", userID)
		return nil
	}

	for _, camp := range campaigns {
		if err := o.generateForCampaign(ctx, camp.ID, camp); err != nil {
			slog.Error("optimizer: failed for campaign", "campaign_id", camp.ID, "name", camp.Name, "err", err)
			continue
		}
		slog.Info("optimizer: recommendations generated", "campaign_id", camp.ID, "name", camp.Name)
	}
	return nil
}

func (o *Optimizer) generateForCampaign(ctx context.Context, campaignID string, camp *domain.Campaign) error {
	insights, err := o.listInsights(ctx, campaignID, 14)
	if err != nil {
		return fmt.Errorf("list insights: %w", err)
	}

	adSets := o.listAdSets(ctx, campaignID)
	anomalies := o.listActiveAnomalies(ctx, campaignID)

	userPrompt := fmt.Sprintf(optimizerPrompt,
		mustMarshal(camp),
		mustMarshal(adSets),
		mustMarshal(toSnapshots(insights)),
		mustMarshal(anomalies),
	)

	resp, err := o.router.Complete(ctx, ai.TaskOptimization, ai.CompletionRequest{
		SystemPrompt: "You are a senior Meta Ads media buyer. Respond only with valid JSON.",
		UserPrompt:   userPrompt,
		MaxTokens:    1500,
		Temperature:  0.3,
		JSONMode:     true,
	})
	if err != nil {
		return fmt.Errorf("router: %w", err)
	}

	var out optimizerResponse
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &out); err != nil {
		return fmt.Errorf("parse response: %w (raw: %s)", err, truncate(resp.Content, 300))
	}

	out.ModelUsed = fmt.Sprintf("%s/%s", resp.Provider, resp.ModelUsed)
	return o.saveRecommendations(ctx, campaignID, &out)
}

func (o *Optimizer) listAtRisk(ctx context.Context, userID string) ([]*domain.Campaign, error) {
	rows, err := o.db.Query(ctx, `
		SELECT id, meta_campaign_id, user_id, ad_account_id, name, objective, status,
		       daily_budget, lifetime_budget, health_status, last_synced_at, created_at, updated_at
		FROM campaigns
		WHERE user_id = $1 AND health_status IN ('AT_RISK','UNDERPERFORMING') AND deleted_at IS NULL
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

func (o *Optimizer) listInsights(ctx context.Context, campaignID string, days int) ([]*domain.CampaignInsight, error) {
	rows, err := o.db.Query(ctx, `
		SELECT id, campaign_id, date, spend, impressions, clicks, ctr, cpc, cpm,
		       reach, frequency, leads, purchases, roas, created_at, updated_at
		FROM campaign_insights
		WHERE campaign_id = $1 AND date >= CURRENT_DATE - $2::int
		ORDER BY date
	`, campaignID, days)
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

func (o *Optimizer) listAdSets(ctx context.Context, campaignID string) []*domain.AdSet {
	rows, err := o.db.Query(ctx, `
		SELECT id, meta_ad_set_id, campaign_id, name, status, daily_budget, optimization_goal, billing_event, created_at, updated_at
		FROM ad_sets WHERE campaign_id = $1 AND deleted_at IS NULL
	`, campaignID)
	if err != nil {
		slog.Error("optimizer: list ad sets", "campaign_id", campaignID, "err", err)
		return nil
	}
	defer rows.Close()

	var result []*domain.AdSet
	for rows.Next() {
		var a domain.AdSet
		if err := rows.Scan(
			&a.ID, &a.MetaAdSetID, &a.CampaignID, &a.Name, &a.Status,
			&a.DailyBudget, &a.OptimizationGoal, &a.BillingEvent,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			slog.Error("optimizer: scan ad set", "err", err)
			continue
		}
		result = append(result, &a)
	}
	return result
}

func (o *Optimizer) listActiveAnomalies(ctx context.Context, campaignID string) []*domain.Anomaly {
	rows, err := o.db.Query(ctx, `
		SELECT id, campaign_id, type, severity, description, is_active, detected_at, resolved_at, created_at, updated_at
		FROM anomalies WHERE campaign_id = $1 AND is_active = true ORDER BY detected_at DESC
	`, campaignID)
	if err != nil {
		slog.Error("optimizer: list anomalies", "campaign_id", campaignID, "err", err)
		return nil
	}
	defer rows.Close()

	var result []*domain.Anomaly
	for rows.Next() {
		var a domain.Anomaly
		if err := rows.Scan(
			&a.ID, &a.CampaignID, &a.Type, &a.Severity, &a.Description,
			&a.IsActive, &a.DetectedAt, &a.ResolvedAt,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			continue
		}
		result = append(result, &a)
	}
	return result
}

func (o *Optimizer) saveRecommendations(ctx context.Context, campaignID string, out *optimizerResponse) error {
	for _, r := range out.Recommendations {
		priority := domain.RecommendationPriority(strings.ToUpper(r.Priority))
		category := domain.RecommendationCategory(strings.ToUpper(r.Category))
		rec := &domain.Recommendation{
			CampaignID:     campaignID,
			Priority:       priority,
			Category:       category,
			Action:         r.Action,
			ExpectedImpact: r.ExpectedImpact,
			Rationale:      r.Rationale,
			ModelUsed:      out.ModelUsed,
		}
		err := o.db.QueryRow(ctx, `
			INSERT INTO recommendations (campaign_id, priority, category, action, expected_impact, rationale, model_used)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			RETURNING id, created_at, updated_at
		`, rec.CampaignID, rec.Priority, rec.Category, rec.Action,
			rec.ExpectedImpact, rec.Rationale, rec.ModelUsed).
			Scan(&rec.ID, &rec.CreatedAt, &rec.UpdatedAt)
		if err != nil {
			return fmt.Errorf("insert recommendation: %w", err)
		}
	}
	return nil
}

// toSnapshots converts insights to a serializable format for prompts.
func toSnapshots(insights []*domain.CampaignInsight) []InsightSnapshot {
	var snap []InsightSnapshot
	for _, i := range insights {
		snap = append(snap, InsightSnapshot{
			Date:        i.Date,
			Spend:       i.Spend,
			CTR:         i.CTR,
			CPC:         i.CPC,
			ROAS:        i.ROAS,
			Leads:       i.Leads,
			Impressions: i.Impressions,
		})
	}
	return snap
}
