package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("failed to connect", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()

	// ── Get user ──────────────────────────────────────────────────────────
	var userID string
	if err := db.QueryRow(ctx, `SELECT id FROM users WHERE email = 'test@metaads.ai' LIMIT 1`).Scan(&userID); err != nil {
		slog.Error("user not found — create one first", "err", err)
		os.Exit(1)
	}
	slog.Info("seeding for user", "id", userID)

	adAccountID := "act_987654321"

	// ── Truncate existing seed data ────────────────────────────────────────
	if _, err := db.Exec(ctx, `DELETE FROM campaign_insights WHERE campaign_id IN (SELECT id FROM campaigns WHERE ad_account_id = $1)`, adAccountID); err != nil {
		slog.Error("truncate insights", "err", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM anomalies WHERE campaign_id IN (SELECT id FROM campaigns WHERE ad_account_id = $1)`, adAccountID); err != nil {
		slog.Error("truncate anomalies", "err", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM recommendations WHERE campaign_id IN (SELECT id FROM campaigns WHERE ad_account_id = $1)`, adAccountID); err != nil {
		slog.Error("truncate recommendations", "err", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM budget_suggestions WHERE ad_account_id = $1`, adAccountID); err != nil {
		slog.Error("truncate budget suggestions", "err", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM campaigns WHERE ad_account_id = $1`, adAccountID); err != nil {
		slog.Error("truncate campaigns", "err", err)
	}

	// ── Campaigns ──────────────────────────────────────────────────────────
	type seedCampaign struct {
		Name          string
		Objective     string
		Status        string
		HealthStatus  string
		DailyBudget   float64
		LifetimeBudget float64
	}

	campaignDefs := []seedCampaign{
		{Name: "Summer Sale 2026 - Retargeting", Objective: "CONVERSIONS", Status: "ACTIVE", HealthStatus: "HEALTHY", DailyBudget: 150, LifetimeBudget: 4500},
		{Name: "New Arrivals - Lookalike 3%", Objective: "CONVERSIONS", Status: "ACTIVE", HealthStatus: "SCALING", DailyBudget: 300, LifetimeBudget: 9000},
		{Name: "Brand Awareness - Video Story", Objective: "REACH", Status: "ACTIVE", HealthStatus: "AT_RISK", DailyBudget: 200, LifetimeBudget: 6000},
		{Name: "Holiday Promo - Conversions", Objective: "CONVERSIONS", Status: "ACTIVE", HealthStatus: "HEALTHY", DailyBudget: 500, LifetimeBudget: 15000},
		{Name: "Clearance - Dynamic Product Ads", Objective: "CATALOG_SALES", Status: "ACTIVE", HealthStatus: "UNDERPERFORMING", DailyBudget: 100, LifetimeBudget: 3000},
		{Name: "Spring Collection - Prospecting", Objective: "TRAFFIC", Status: "ACTIVE", HealthStatus: "SCALING", DailyBudget: 250, LifetimeBudget: 7500},
	}

	type campaignRow struct {
		ID   string
		Name string
		Def  seedCampaign
	}

	var campaigns []campaignRow
	for i, c := range campaignDefs {
		metaID := fmt.Sprintf("2385%d", 490000+i)
		var id string
		err := db.QueryRow(ctx,
			`INSERT INTO campaigns (meta_campaign_id, user_id, ad_account_id, name, objective, status, daily_budget, lifetime_budget, health_status, last_synced_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,now())
			 ON CONFLICT (user_id, meta_campaign_id) DO UPDATE SET name=EXCLUDED.name, health_status=EXCLUDED.health_status
			 RETURNING id`,
			metaID, userID, adAccountID, c.Name, c.Objective, c.Status, c.DailyBudget, c.LifetimeBudget, c.HealthStatus,
		).Scan(&id)
		if err != nil {
			slog.Error("insert campaign", "name", c.Name, "err", err)
			os.Exit(1)
		}
		campaigns = append(campaigns, campaignRow{ID: id, Name: c.Name, Def: c})
		slog.Info("campaign inserted", "name", c.Name, "id", id)
	}

	// ── Daily Insights (last 30 days) ──────────────────────────────────────
	now := time.Now()
	for _, camp := range campaigns {
		for daysAgo := 0; daysAgo < 30; daysAgo++ {
			date := now.AddDate(0, 0, -daysAgo)
			spend := camp.Def.DailyBudget * (0.7 + rand.Float64()*0.6) // 70-130% of daily budget
			impressions := int64(5000 + rand.Intn(15000))
			clicks := int64(float64(impressions) * (0.01 + rand.Float64()*0.04))
			ctr := float64(clicks) / float64(impressions)
			cpc := spend / float64(clicks)
			cpm := spend / float64(impressions) * 1000
			reach := int64(float64(impressions) * (0.3 + rand.Float64()*0.4))
			frequency := float64(impressions) / float64(reach)
			leads := int64(float64(clicks) * (0.02 + rand.Float64()*0.08))
			purchases := int64(float64(clicks) * (0.005 + rand.Float64()*0.02))
			roas := 0.0
			if spend > 0 {
				roas = float64(purchases) * 50.0 / spend // average $50 order value
			}

			// Skew based on health
			switch camp.Def.HealthStatus {
			case "UNDERPERFORMING":
				spend *= 0.5
				roas *= 0.4
				clicks = int64(float64(clicks) * 0.6)
				purchases = int64(float64(purchases) * 0.3)
			case "AT_RISK":
				if daysAgo < 7 { // recent decline
					spend *= 0.6
					roas *= 0.5
				}
			case "SCALING":
				if daysAgo < 10 { // recent improvement
					spend *= 1.3
					roas *= 1.4
				}
			}

			_, err := db.Exec(ctx,
				`INSERT INTO campaign_insights (campaign_id, date, spend, impressions, clicks, ctr, cpc, cpm, reach, frequency, leads, purchases, roas)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
				 ON CONFLICT (campaign_id, date) DO UPDATE SET spend=$3, impressions=$4, clicks=$5`,
				camp.ID, date,
				round(spend, 2), impressions, clicks,
				round(ctr, 4), round(cpc, 4), round(cpm, 4),
				reach, round(frequency, 4), leads, purchases, round(roas, 4),
			)
			if err != nil {
				slog.Error("insert insight", "campaign", camp.Name, "date", date, "err", err)
			}
		}
		slog.Info("insights inserted", "campaign", camp.Name)
	}

	// ── Anomalies ──────────────────────────────────────────────────────────
	type anomalyDef struct {
		CampaignName string
		Type         string
		Severity     string
		Description  string
		IsActive     bool
		DaysAgo      int
	}

	anomalyDefs := []anomalyDef{
		{CampaignName: "Brand Awareness - Video Story", Type: "CPM_SPIKE", Severity: "HIGH", Description: "CPM increased 340% over 48h — targeting saturation likely causing auction pressure.", IsActive: true, DaysAgo: 2},
		{CampaignName: "Brand Awareness - Video Story", Type: "CTR_DROP", Severity: "MEDIUM", Description: "CTR dropped from 2.1% to 0.8% — creative fatigue detected across all ad sets.", IsActive: true, DaysAgo: 5},
		{CampaignName: "Clearance - Dynamic Product Ads", Type: "ROAS_DECLINE", Severity: "HIGH", Description: "ROAS fell from 2.4 to 0.6 over 7 days. Audience overlap with other campaigns suspected.", IsActive: true, DaysAgo: 3},
		{CampaignName: "Clearance - Dynamic Product Ads", Type: "BUDGET_PACING", Severity: "MEDIUM", Description: "Campaign pacing at 40% of daily budget — delivery issues detected.", IsActive: true, DaysAgo: 1},
		{CampaignName: "Summer Sale 2026 - Retargeting", Type: "FREQUENCY_SPIKE", Severity: "MEDIUM", Description: "Frequency reached 8.2x in retargeting audience — ad fatigue risk.", IsActive: false, DaysAgo: 15},
		{CampaignName: "Holiday Promo - Conversions", Type: "BUDGET_SPIKE", Severity: "LOW", Description: "Daily spend exceeded budget by 22% due to accelerated delivery.", IsActive: false, DaysAgo: 20},
		{CampaignName: "New Arrivals - Lookalike 3%", Type: "ROAS_SPIKE", Severity: "LOW", Description: "ROAS jumped from 2.1 to 4.8 temporarily — investigate for scaling opportunity.", IsActive: true, DaysAgo: 1},
	}

	for _, a := range anomalyDefs {
		for _, camp := range campaigns {
			if camp.Name == a.CampaignName {
				detectedAt := now.AddDate(0, 0, -a.DaysAgo)
				var resolvedAt *time.Time
				if !a.IsActive {
					t := detectedAt.AddDate(0, 0, 2)
					resolvedAt = &t
				}
				_, err := db.Exec(ctx,
					`INSERT INTO anomalies (campaign_id, type, severity, description, is_active, detected_at, resolved_at)
					 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
					camp.ID, a.Type, a.Severity, a.Description, a.IsActive, detectedAt, resolvedAt,
				)
				if err != nil {
					slog.Error("insert anomaly", "campaign", camp.Name, "err", err)
				}
			}
		}
	}
	slog.Info("anomalies inserted")

	// ── Recommendations ────────────────────────────────────────────────────
	type recDef struct {
		CampaignName string
		Priority     string
		Category     string
		Action       string
		Impact       string
		Rationale    string
		Model        string
	}

	recDefs := []recDef{
		{CampaignName: "Clearance - Dynamic Product Ads", Priority: "HIGH", Category: "Budget", Action: "Reduce daily budget to $60 and reallocate $40 to Spring Collection", Impact: "12% projected ROAS improvement", Rationale: "Marginal ROAS below breakeven for 7 consecutive days. Budget reallocation to SCALING campaign yields higher marginal return.", Model: "claude-opus-4-7"},
		{CampaignName: "Clearance - Dynamic Product Ads", Priority: "HIGH", Category: "Creative", Action: "Replace top 3 underperforming product images with lifestyle variants", Impact: "Estimated 1.5% CTR uplift", Rationale: "Product-only creatives showing 0.4% CTR vs 1.2% for lifestyle creatives in this account.", Model: "claude-sonnet-4-6"},
		{CampaignName: "Brand Awareness - Video Story", Priority: "HIGH", Category: "Targeting", Action: "Exclude audiences with frequency > 6 and refresh creative set", Impact: "Expected 40-50% CPM reduction", Rationale: "Frequency saturation driving auction costs. Audience refresh with 2 new video variants recommended.", Model: "claude-opus-4-7"},
		{CampaignName: "Brand Awareness - Video Story", Priority: "MEDIUM", Category: "Bidding", Action: "Switch from lowest cost to cost cap ($25 CPM)", Impact: "Should stabilize CPM within 48h", Rationale: "Lowest cost bidding susceptible to auction spikes when targeting saturates.", Model: "claude-sonnet-4-6"},
		{CampaignName: "New Arrivals - Lookalike 3%", Priority: "MEDIUM", Category: "Budget", Action: "Increase daily budget to $400 with 20% increment per 3 days", Impact: "Projected 2.5x spend capacity at 1.8+ ROAS", Rationale: "Current ROAS 2.4x with stable CPA. Lookalike audience size supports 2-3x budget scaling.", Model: "claude-opus-4-7"},
		{CampaignName: "Summer Sale 2026 - Retargeting", Priority: "LOW", Category: "Creative", Action: "Add urgency-driven copy variant for cart abandoners", Impact: "Potential 0.8% CVR improvement", Rationale: "Retargeting CTR healthy but CVR declining week-over-week. Fresh messaging could reverse trend.", Model: "deepseek-v4-pro"},
		{CampaignName: "Holiday Promo - Conversions", Priority: "LOW", Category: "Scheduling", Action: "Shift 20% of budget to weekend delivery (Sat-Sun)", Impact: "Projected 8% efficiency gain", Rationale: "Weekend CVR 22% higher than weekday average over last 14 days.", Model: "claude-sonnet-4-6"},
		{CampaignName: "Spring Collection - Prospecting", Priority: "MEDIUM", Category: "Targeting", Action: "Test interest-based layer on top of broad targeting", Impact: "Expected 15% CPA improvement", Rationale: "Broad targeting CPM is efficient but CPA is 20% above goal. Interest layering can improve conversion quality.", Model: "deepseek-r2"},
	}

	for _, r := range recDefs {
		for _, camp := range campaigns {
			if camp.Name == r.CampaignName {
				_, err := db.Exec(ctx,
					`INSERT INTO recommendations (campaign_id, priority, category, action, expected_impact, rationale, model_used)
					 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
					camp.ID, r.Priority, r.Category, r.Action, r.Impact, r.Rationale, r.Model,
				)
				if err != nil {
					slog.Error("insert recommendation", "campaign", camp.Name, "err", err)
				}
			}
		}
	}
	slog.Info("recommendations inserted")

	// ── Budget Suggestions ─────────────────────────────────────────────────
	type budgetDef struct {
		CampaignName    string
		CurrentBudget   float64
		SuggestedBudget float64
		Reason          string
		ShouldPause     bool
		ROASImprovement string
		Model           string
	}

	budgetDefs := []budgetDef{
		{CampaignName: "Clearance - Dynamic Product Ads", CurrentBudget: 100, SuggestedBudget: 60, ShouldPause: false, Reason: "Reduce — ROAS below 1.0 for 7 days. Reallocate to higher-performing campaigns.", ROASImprovement: "+0.4x projected", Model: "claude-opus-4-7"},
		{CampaignName: "New Arrivals - Lookalike 3%", CurrentBudget: 300, SuggestedBudget: 400, ShouldPause: false, Reason: "Increase — ROAS 2.4x with stable CPA. Audience size supports scaling.", ROASImprovement: "+$1,200/mo revenue", Model: "claude-opus-4-7"},
		{CampaignName: "Brand Awareness - Video Story", CurrentBudget: 200, SuggestedBudget: 150, ShouldPause: false, Reason: "Reduce temporarily until creative refresh — CPM spike eroding efficiency.", ROASImprovement: "+30% efficiency", Model: "claude-sonnet-4-6"},
	}

	for _, b := range budgetDefs {
		for _, camp := range campaigns {
			if camp.Name == b.CampaignName {
				_, err := db.Exec(ctx,
					`INSERT INTO budget_suggestions (user_id, ad_account_id, campaign_id, current_budget, suggested_budget, change_reason, should_pause, expected_roas_improvement, model_used)
					 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
					userID, adAccountID, camp.ID, b.CurrentBudget, b.SuggestedBudget, b.Reason, b.ShouldPause, b.ROASImprovement, b.Model,
				)
				if err != nil {
					slog.Error("insert budget suggestion", "campaign", camp.Name, "err", err)
				}
			}
		}
	}
	slog.Info("budget suggestions inserted")

	// ── LLM Usage (last 14 days) ──────────────────────────────────────────
	providers := []string{"anthropic", "openai", "deepseek", "zhipu", "moonshot", "alibaba", "xai"}
	tasks := []string{"anomaly_detection", "recommendation", "budget_advisor", "creative_insights", "campaign_analysis"}
	for daysAgo := 0; daysAgo < 14; daysAgo++ {
		date := now.AddDate(0, 0, -daysAgo)
		for _, p := range providers {
			// Random 1-4 calls per provider per day
			calls := 1 + rand.Intn(4)
			for call := 0; call < calls; call++ {
				task := tasks[rand.Intn(len(tasks))]
				inputTokens := 200 + rand.Intn(2000)
				outputTokens := 100 + rand.Intn(1500)
				cost := float64(inputTokens)*0.000003 + float64(outputTokens)*0.000015
				latency := 300 + rand.Intn(3000)
				_, err := db.Exec(ctx,
					`INSERT INTO llm_usage (user_id, task_type, provider, model, input_tokens, output_tokens, cost_usd, latency_ms, success, created_at)
					 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
					userID, task, p, p+"-default", inputTokens, outputTokens, round(cost, 6), latency, true, date,
				)
				if err != nil {
					slog.Error("insert llm_usage", "err", err)
				}
			}
		}
	}
	slog.Info("llm usage inserted")

	slog.Info("seed complete — all dashboards should now show data")
}

func round(v float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(v*pow+0.5)) / pow
}
