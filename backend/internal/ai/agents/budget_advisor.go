package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/domain"
)

const budgetAdvisorPrompt = `You are a Meta Ads budget optimization specialist.

Portfolio for account %s. Total daily budget: $%.2f

Campaigns (ranked by ROAS):
%s

Rules:
- Never suggest cutting below $10/day
- Prioritize campaigns with ROAS > 2 and scaling headroom
- Flag campaigns that should be paused

Respond in JSON:
{
  "reallocations": [
    {
      "campaign_id": "...",
      "campaign_name": "...",
      "current_budget": 0.00,
      "suggested_budget": 0.00,
      "change_reason": "..."
    }
  ],
  "campaigns_to_pause": ["campaign_id_1"],
  "expected_portfolio_roas_improvement": "X%",
  "summary": "..."
}`

type reallocation struct {
	CampaignID      string  `json:"campaign_id"`
	CampaignName    string  `json:"campaign_name"`
	CurrentBudget   float64 `json:"current_budget"`
	SuggestedBudget float64 `json:"suggested_budget"`
	ChangeReason    string  `json:"change_reason"`
}

type budgetAdvisorResponse struct {
	Reallocations                  []reallocation `json:"reallocations"`
	CampaignsToPause               []string       `json:"campaigns_to_pause"`
	ExpectedPortfolioROASImprovement string        `json:"expected_portfolio_roas_improvement"`
	Summary                        string         `json:"summary"`
}

type campaignROAS struct {
	camp *domain.Campaign
	roas float64
	ctr  float64
	cpc  float64
}

// BudgetAdvisor analyzes the full portfolio and suggests budget redistribution.
// Primary model: gemini-2-5-pro (routed via TaskBudgetAdvisor).
type BudgetAdvisor struct {
	db     *pgxpool.Pool
	router *ai.Router
}

func NewBudgetAdvisor(db *pgxpool.Pool, router *ai.Router) *BudgetAdvisor {
	return &BudgetAdvisor{db: db, router: router}
}

// RunAnalyzeAll generates budget suggestions for all users with campaigns.
func (ba *BudgetAdvisor) RunAnalyzeAll(ctx context.Context, userID string) error {
	campaigns, err := ba.listCampaigns(ctx, userID)
	if err != nil {
		return fmt.Errorf("budget_advisor: list campaigns: %w", err)
	}
	if len(campaigns) == 0 {
		slog.Info("budget_advisor: no campaigns", "user_id", userID)
		return nil
	}

	// Enrich campaigns with last 7-day ROAS
	var ranked []campaignROAS

	for _, camp := range campaigns {
		insights, err := ba.listInsights(ctx, camp.ID, 7)
		if err != nil {
			slog.Error("budget_advisor: fetch insights", "campaign_id", camp.ID, "err", err)
			continue
		}
		if len(insights) == 0 {
			continue
		}
		var roasSum, ctrSum, cpcSum float64
		for _, i := range insights {
			roasSum += i.ROAS
			ctrSum += i.CTR
			cpcSum += i.CPC
		}
		n := float64(len(insights))
		ranked = append(ranked, campaignROAS{
			camp: camp,
			roas: roasSum / n,
			ctr:  ctrSum / n,
			cpc:  cpcSum / n,
		})
	}

	if len(ranked) == 0 {
		slog.Info("budget_advisor: no campaigns with insights", "user_id", userID)
		return nil
	}

	// Rank by ROAS descending
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			if ranked[j].roas > ranked[i].roas {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			}
		}
	}

	// Total daily budget
	var totalBudget float64
	for _, cr := range ranked {
		if cr.camp.DailyBudget != nil {
			totalBudget += *cr.camp.DailyBudget
		}
	}

	// Build portfolio JSON
	type portfolioItem struct {
		CampaignID      string  `json:"campaign_id"`
		Name            string  `json:"name"`
		Objective       string  `json:"objective"`
		Status          string  `json:"status"`
		DailyBudget     float64 `json:"daily_budget"`
		HealthStatus    string  `json:"health_status"`
		AvgROAS         float64 `json:"avg_roas_7d"`
		AvgCTR          float64 `json:"avg_ctr_7d"`
		AvgCPC          float64 `json:"avg_cpc_7d"`
	}

	var portfolio []portfolioItem
	for _, cr := range ranked {
		budget := 0.0
		if cr.camp.DailyBudget != nil {
			budget = *cr.camp.DailyBudget
		}
		portfolio = append(portfolio, portfolioItem{
			CampaignID:   cr.camp.MetaCampaignID,
			Name:         cr.camp.Name,
			Objective:    cr.camp.Objective,
			Status:       cr.camp.Status,
			DailyBudget:  budget,
			HealthStatus: string(cr.camp.HealthStatus),
			AvgROAS:      cr.roas,
			AvgCTR:       cr.ctr,
			AvgCPC:       cr.cpc,
		})
	}

	adAccountID := ""
	if len(ranked) > 0 {
		adAccountID = ranked[0].camp.AdAccountID
	}

	userPrompt := fmt.Sprintf(budgetAdvisorPrompt,
		adAccountID,
		totalBudget,
		mustMarshal(portfolio),
	)

	resp, err := ba.router.Complete(ctx, ai.TaskBudgetAdvisor, ai.CompletionRequest{
		SystemPrompt: "You are a Meta Ads budget optimization specialist. Respond only with valid JSON.",
		UserPrompt:   userPrompt,
		MaxTokens:    2000,
		Temperature:  0.2,
		JSONMode:     true,
	})
	if err != nil {
		return fmt.Errorf("router: %w", err)
	}

	var out budgetAdvisorResponse
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &out); err != nil {
		return fmt.Errorf("parse response: %w (raw: %s)", err, truncate(resp.Content, 300))
	}

	return ba.saveSuggestions(ctx, userID, adAccountID, ranked, &out, resp)
}

func (ba *BudgetAdvisor) saveSuggestions(ctx context.Context, userID, adAccountID string, ranked []campaignROAS, out *budgetAdvisorResponse, resp ai.CompletionResponse) error {
	// Build lookup from meta_campaign_id to internal campaign ID
	lookup := make(map[string]*domain.Campaign)
	for _, cr := range ranked {
		lookup[cr.camp.MetaCampaignID] = cr.camp
	}

	for _, r := range out.Reallocations {
		camp, ok := lookup[r.CampaignID]
		if !ok {
			continue
		}
		s := &domain.BudgetSuggestion{
			UserID:                  userID,
			AdAccountID:             adAccountID,
			CampaignID:              &camp.ID,
			CampaignName:            camp.Name,
			CurrentBudget:           r.CurrentBudget,
			SuggestedBudget:         r.SuggestedBudget,
			ChangeReason:            r.ChangeReason,
			ShouldPause:             false,
			ExpectedROASImprovement: out.ExpectedPortfolioROASImprovement,
			PortfolioSummary:        out.Summary,
			ModelUsed:               resp.Provider + "/" + resp.ModelUsed,
		}
		err := ba.db.QueryRow(ctx, `
			INSERT INTO budget_suggestions
			  (user_id, ad_account_id, campaign_id, current_budget, suggested_budget,
			   change_reason, should_pause, expected_roas_improvement, portfolio_summary, model_used)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id, created_at
		`, s.UserID, s.AdAccountID, s.CampaignID,
			s.CurrentBudget, s.SuggestedBudget, s.ChangeReason,
			s.ShouldPause, s.ExpectedROASImprovement, s.PortfolioSummary, s.ModelUsed,
		).Scan(&s.ID, &s.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert budget suggestion: %w", err)
		}
	}

	// Pause suggestions
	for _, pauseID := range out.CampaignsToPause {
		camp, ok := lookup[pauseID]
		if !ok {
			continue
		}
		s := &domain.BudgetSuggestion{
			UserID:                  userID,
			AdAccountID:             adAccountID,
			CampaignID:              &camp.ID,
			CampaignName:            camp.Name,
			ShouldPause:             true,
			ChangeReason:            "AI suggested pause due to underperformance",
			ExpectedROASImprovement: out.ExpectedPortfolioROASImprovement,
			PortfolioSummary:        out.Summary,
			ModelUsed:               resp.Provider + "/" + resp.ModelUsed,
		}
		err := ba.db.QueryRow(ctx, `
			INSERT INTO budget_suggestions
			  (user_id, ad_account_id, campaign_id, current_budget, suggested_budget,
			   change_reason, should_pause, expected_roas_improvement, portfolio_summary, model_used)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id, created_at
		`, s.UserID, s.AdAccountID, s.CampaignID,
			s.CurrentBudget, s.SuggestedBudget, s.ChangeReason,
			s.ShouldPause, s.ExpectedROASImprovement, s.PortfolioSummary, s.ModelUsed,
		).Scan(&s.ID, &s.CreatedAt)
		if err != nil {
			slog.Error("budget_advisor: save pause suggestion", "campaign_id", pauseID, "err", err)
		}
	}

	slog.Info("budget_advisor: suggestions saved",
		"reallocations", len(out.Reallocations),
		"pauses", len(out.CampaignsToPause),
		"model", resp.ModelUsed,
	)
	return nil
}

func (ba *BudgetAdvisor) listCampaigns(ctx context.Context, userID string) ([]*domain.Campaign, error) {
	rows, err := ba.db.Query(ctx, `
		SELECT id, meta_campaign_id, user_id, ad_account_id, name, objective, status,
		       daily_budget, lifetime_budget, health_status, last_synced_at, created_at, updated_at
		FROM campaigns WHERE user_id = $1 AND status = 'ACTIVE' AND deleted_at IS NULL
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

func (ba *BudgetAdvisor) listInsights(ctx context.Context, campaignID string, days int) ([]*domain.CampaignInsight, error) {
	rows, err := ba.db.Query(ctx, `
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
