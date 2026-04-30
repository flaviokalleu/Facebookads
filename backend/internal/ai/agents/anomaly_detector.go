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

const anomalyConfirmPrompt = `You are a Meta Ads performance anomaly detection specialist.

Confirm whether the following rule-triggered anomaly is genuine, and assign severity.

Campaign: %s
Anomaly type: %s
Current metrics: %s

Historical context (7-day daily):
%s

Respond ONLY with valid JSON:
{
  "is_genuine": true/false,
  "severity": "HIGH|MEDIUM|LOW",
  "description": "concise explanation of what happened and why"
}`

type anomalyConfirmResponse struct {
	IsGenuine   bool   `json:"is_genuine"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

// ruleTrigger holds a pre-filter match before AI confirmation.
type ruleTrigger struct {
	CampaignID string
	Type       domain.AnomalyType
	Severity   domain.AnomalySeverity
	Context    string
}

// AnomalyDetector detects performance anomalies using rule pre-filter + AI confirmation.
// Primary model: gemini-2-5-pro (routed via TaskAnomalyDetection).
type AnomalyDetector struct {
	db     *pgxpool.Pool
	router *ai.Router
}

func NewAnomalyDetector(db *pgxpool.Pool, router *ai.Router) *AnomalyDetector {
	return &AnomalyDetector{db: db, router: router}
}

// RunDetectAll scans all user campaigns and detects anomalies.
func (d *AnomalyDetector) RunDetectAll(ctx context.Context, userID string) error {
	campaigns, err := d.listCampaigns(ctx, userID)
	if err != nil {
		return fmt.Errorf("anomaly_detector: list campaigns: %w", err)
	}
	if len(campaigns) == 0 {
		slog.Info("anomaly_detector: no campaigns", "user_id", userID)
		return nil
	}

	to := time.Now().Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -7)

	for _, camp := range campaigns {
		insights, err := d.listInsights(ctx, camp.ID, from, to)
		if err != nil {
			slog.Error("anomaly_detector: fetch insights", "campaign_id", camp.ID, "err", err)
			continue
		}
		if len(insights) < 2 {
			continue
		}

		triggers := d.ruleFilter(camp, insights)
		for _, t := range triggers {
			confirmed, err := d.aiConfirm(ctx, camp, insights, t)
			if err != nil {
				slog.Error("anomaly_detector: ai confirm failed", "type", t.Type, "err", err)
				continue
			}
			if !confirmed.IsGenuine {
				continue
			}

			severity := domain.AnomalySeverity(strings.ToUpper(confirmed.Severity))
			if severity != domain.SeverityHigh && severity != domain.SeverityMedium && severity != domain.SeverityLow {
				severity = domain.SeverityMedium
			}

			if err := d.saveAnomaly(ctx, camp.ID, t.Type, severity, confirmed.Description); err != nil {
				slog.Error("anomaly_detector: save anomaly", "campaign_id", camp.ID, "err", err)
				continue
			}
			slog.Info("anomaly_detector: anomaly saved", "campaign_id", camp.ID, "type", t.Type, "severity", severity)
		}
	}
	return nil
}

// ruleFilter applies pre-filters before any AI call. Fast, no network.
func (d *AnomalyDetector) ruleFilter(camp *domain.Campaign, insights []*domain.CampaignInsight) []ruleTrigger {
	var triggers []ruleTrigger

	todayIdx := len(insights) - 1
	today := insights[todayIdx]

	sevenDayAvg := func(getter func(*domain.CampaignInsight) float64) float64 {
		var sum float64
		for _, i := range insights {
			sum += getter(i)
		}
		return sum / float64(len(insights))
	}

	// CPC Spike: today > 150% of 7-day average
	avgCPC := sevenDayAvg(func(i *domain.CampaignInsight) float64 { return i.CPC })
	if avgCPC > 0 && today.CPC > avgCPC*1.5 {
		triggers = append(triggers, ruleTrigger{
			CampaignID: camp.ID,
			Type:       domain.AnomalyCPCSpike,
			Severity:   domain.SeverityHigh,
			Context:    fmt.Sprintf("CPC $%.2f vs 7d avg $%.2f (+%.0f%%)", today.CPC, avgCPC, (today.CPC/avgCPC-1)*100),
		})
	}

	// CTR Drop: today < 50% of 7-day average
	avgCTR := sevenDayAvg(func(i *domain.CampaignInsight) float64 { return i.CTR })
	if avgCTR > 0 && today.CTR < avgCTR*0.5 {
		triggers = append(triggers, ruleTrigger{
			CampaignID: camp.ID,
			Type:       domain.AnomalyCTRDrop,
			Severity:   domain.SeverityHigh,
			Context:    fmt.Sprintf("CTR %.2f%% vs 7d avg %.2f%% (-%.0f%%)", today.CTR, avgCTR, (1-today.CTR/avgCTR)*100),
		})
	}

	// Creative Fatigue: Frequency > 4 AND CTR declining 3 days
	if today.Frequency > 4 && len(insights) >= 4 {
		last3 := insights[len(insights)-3:]
		if last3[0].CTR > last3[1].CTR && last3[1].CTR > last3[2].CTR {
			triggers = append(triggers, ruleTrigger{
				CampaignID: camp.ID,
				Type:       domain.AnomalyCreativeFatigue,
				Severity:   domain.SeverityMedium,
				Context:    fmt.Sprintf("Frequency %.2f, CTR declining 3 days: %.2f→%.2f→%.2f", today.Frequency, last3[0].CTR, last3[1].CTR, last3[2].CTR),
			})
		}
	}

	// Budget Waste: Spend > 80% of budget AND leads = 0
	if camp.DailyBudget != nil && *camp.DailyBudget > 0 {
		if today.Spend > *camp.DailyBudget*0.8 && today.Leads == 0 {
			triggers = append(triggers, ruleTrigger{
				CampaignID: camp.ID,
				Type:       domain.AnomalyBudgetWaste,
				Severity:   domain.SeverityMedium,
				Context:    fmt.Sprintf("Spend $%.2f of $%.2f budget with 0 leads", today.Spend, *camp.DailyBudget),
			})
		}
	}

	// Audience Saturation: Reach plateau AND frequency > 5
	if today.Frequency > 5 && len(insights) >= 3 {
		last3Reach := insights[len(insights)-3:]
		reachGrowth := true
		for i := 1; i < len(last3Reach); i++ {
			if float64(last3Reach[i].Reach) > float64(last3Reach[i-1].Reach)*1.05 {
				reachGrowth = false
				break
			}
		}
		if reachGrowth {
			triggers = append(triggers, ruleTrigger{
				CampaignID: camp.ID,
				Type:       domain.AnomalyAudienceSaturation,
				Severity:   domain.SeverityMedium,
				Context:    fmt.Sprintf("Frequency %.2f, reach plateau at %d", today.Frequency, today.Reach),
			})
		}
	}

	// ROAS Collapse: Dropped > 40% vs previous 7 days
	if len(insights) >= 7 {
		firstWeek := insights[:7]
		var firstWeekROAS float64
		for _, i := range firstWeek {
			firstWeekROAS += i.ROAS
		}
		firstWeekROAS /= float64(len(firstWeek))
		if firstWeekROAS > 0 && today.ROAS < firstWeekROAS*0.6 {
			triggers = append(triggers, ruleTrigger{
				CampaignID: camp.ID,
				Type:       domain.AnomalyROASCollapse,
				Severity:   domain.SeverityHigh,
				Context:    fmt.Sprintf("ROAS %.2f vs earlier 7d avg %.2f (-%.0f%%)", today.ROAS, firstWeekROAS, (1-today.ROAS/firstWeekROAS)*100),
			})
		}
	}

	// Delivery Stall: Impressions dropped > 70% with no status change
	avgImpr := sevenDayAvg(func(i *domain.CampaignInsight) float64 { return float64(i.Impressions) })
	if avgImpr > 0 && float64(today.Impressions) < avgImpr*0.3 {
		if camp.Status == "ACTIVE" {
			triggers = append(triggers, ruleTrigger{
				CampaignID: camp.ID,
				Type:       domain.AnomalyDeliveryStall,
				Severity:   domain.SeverityHigh,
				Context:    fmt.Sprintf("Impressions %d vs 7d avg %.0f (-%.0f%%)", today.Impressions, avgImpr, (1-float64(today.Impressions)/avgImpr)*100),
			})
		}
	}

	return triggers
}

func (d *AnomalyDetector) aiConfirm(ctx context.Context, camp *domain.Campaign, insights []*domain.CampaignInsight, trigger ruleTrigger) (*anomalyConfirmResponse, error) {
	snapshot := dailySnapshots(insights)
	userPrompt := fmt.Sprintf(anomalyConfirmPrompt,
		camp.Name,
		trigger.Type,
		trigger.Context,
		mustMarshal(snapshot),
	)

	resp, err := d.router.Complete(ctx, ai.TaskAnomalyDetection, ai.CompletionRequest{
		SystemPrompt: "You are a Meta Ads anomaly detection specialist. Respond only with valid JSON.",
		UserPrompt:   userPrompt,
		MaxTokens:    300,
		Temperature:  0.1,
		JSONMode:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("router: %w", err)
	}

	var out anomalyConfirmResponse
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &out); err != nil {
		return nil, fmt.Errorf("parse response: %w (raw: %s)", err, truncate(resp.Content, 200))
	}
	return &out, nil
}

func (d *AnomalyDetector) saveAnomaly(ctx context.Context, campaignID string, anomalyType domain.AnomalyType, severity domain.AnomalySeverity, description string) error {
	_, err := d.db.Exec(ctx, `
		INSERT INTO anomalies (campaign_id, type, severity, description, is_active, detected_at)
		VALUES ($1, $2, $3, $4, true, now())
	`, campaignID, anomalyType, severity, description)
	return err
}

func (d *AnomalyDetector) listCampaigns(ctx context.Context, userID string) ([]*domain.Campaign, error) {
	rows, err := d.db.Query(ctx, `
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

func (d *AnomalyDetector) listInsights(ctx context.Context, campaignID string, from, to time.Time) ([]*domain.CampaignInsight, error) {
	rows, err := d.db.Query(ctx, `
		SELECT id, campaign_id, date, spend, impressions, clicks, ctr, cpc, cpm,
		       reach, frequency, leads, purchases, roas, created_at, updated_at
		FROM campaign_insights
		WHERE campaign_id = $1 AND date BETWEEN $2 AND $3 ORDER BY date
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
