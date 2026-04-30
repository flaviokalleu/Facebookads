package usecase

import (
	"context"
	"time"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/repository"
)

type DashboardUseCase struct {
	campaigns   repository.CampaignRepository
	insights    repository.InsightRepository
	anomalies   repository.AnomalyRepository
	budget      repository.BudgetSuggestionRepository
	llmUsage    repository.LLMUsageRepository
	recommend   repository.RecommendationRepository
}

func NewDashboardUseCase(
	campaigns repository.CampaignRepository,
	insights repository.InsightRepository,
	anomalies repository.AnomalyRepository,
	budget repository.BudgetSuggestionRepository,
	llmUsage repository.LLMUsageRepository,
	recommend repository.RecommendationRepository,
) *DashboardUseCase {
	return &DashboardUseCase{
		campaigns: campaigns,
		insights:  insights,
		anomalies: anomalies,
		budget:    budget,
		llmUsage:  llmUsage,
		recommend: recommend,
	}
}

type DailySpendEntry struct {
	Date  string  `json:"date"`
	Spend float64 `json:"spend"`
	Leads int64   `json:"leads"`
}

type DashboardSummary struct {
	TotalSpend   float64           `json:"total_spend"`
	TotalLeads   int64             `json:"total_leads"`
	AvgCTR       float64           `json:"avg_ctr"`
	AvgCPC       float64           `json:"avg_cpc"`
	AvgROAS      float64           `json:"avg_roas"`
	SpendDelta   float64           `json:"spend_delta"`
	LeadsDelta   float64           `json:"leads_delta"`
	CTRDelta     float64           `json:"ctr_delta"`
	ROASDelta    float64           `json:"roas_delta"`
	DailySpend   []DailySpendEntry `json:"daily_spend"`
	LastSyncedAt *time.Time        `json:"last_synced_at,omitempty"`
	IsStale      bool              `json:"is_stale"`
}

func (uc *DashboardUseCase) Summary(ctx context.Context, userID string) (*DashboardSummary, error) {
	campaigns, err := uc.campaigns.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	currentFrom := now.AddDate(0, 0, -30)
	previousFrom := now.AddDate(0, 0, -60)
	previousTo := currentFrom.AddDate(0, 0, -1)

	var (
		totalSpend       float64
		totalLeads       int64
		prevTotalSpend   float64
		prevTotalLeads   int64
		sumCTR           float64
		sumCPC           float64
		sumROAS          float64
		prevSumCTR       float64
		prevSumCPC       float64
		prevSumROAS      float64
		count            int
		prevCount        int
		lastSync         *time.Time
		dailyMap         = make(map[string]*DailySpendEntry)
	)

	for _, c := range campaigns {
		if c.LastSyncedAt != nil {
			if lastSync == nil || c.LastSyncedAt.After(*lastSync) {
				lastSync = c.LastSyncedAt
			}
		}

		// Current period
		rows, err := uc.insights.ListByCampaign(ctx, c.ID, currentFrom, now)
		if err != nil {
			continue
		}
		for _, row := range rows {
			totalSpend += row.Spend
			totalLeads += row.Leads
			sumCTR += row.CTR
			sumCPC += row.CPC
			sumROAS += row.ROAS
			count++

			dateKey := row.Date.Format("2006-01-02")
			if dailyMap[dateKey] == nil {
				dailyMap[dateKey] = &DailySpendEntry{Date: dateKey}
			}
			dailyMap[dateKey].Spend += row.Spend
			dailyMap[dateKey].Leads += row.Leads
		}

		// Previous period (for delta)
		prevRows, _ := uc.insights.ListByCampaign(ctx, c.ID, previousFrom, previousTo)
		for _, row := range prevRows {
			prevTotalSpend += row.Spend
			prevTotalLeads += row.Leads
			prevSumCTR += row.CTR
			prevSumCPC += row.CPC
			prevSumROAS += row.ROAS
			prevCount++
		}
	}

	// Build sorted daily spend array (last 30 days)
	dailySpend := make([]DailySpendEntry, 0, 30)
	for d := now.AddDate(0, 0, -29); !d.After(now); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if entry, ok := dailyMap[key]; ok {
			entry.Spend = round2(entry.Spend)
			dailySpend = append(dailySpend, *entry)
		} else {
			dailySpend = append(dailySpend, DailySpendEntry{Date: key})
		}
	}

	s := &DashboardSummary{
		TotalSpend:   round2(totalSpend),
		TotalLeads:   totalLeads,
		DailySpend:   dailySpend,
		LastSyncedAt: lastSync,
	}

	// Averages
	if count > 0 {
		s.AvgCTR = round2(sumCTR / float64(count))
		s.AvgCPC = round2(sumCPC / float64(count))
		s.AvgROAS = round2(sumROAS / float64(count))
	}

	// Deltas
	if prevCount > 0 {
		prevAvgSpend := prevTotalSpend
		if totalSpend > 0 {
			s.SpendDelta = round2((totalSpend - prevAvgSpend) / totalSpend * 100)
		}
		prevAvgLeads := float64(prevTotalLeads)
		if totalLeads > 0 {
			s.LeadsDelta = round2((float64(totalLeads) - prevAvgLeads) / float64(totalLeads) * 100)
		}
		prevAvgCTR := prevSumCTR / float64(prevCount)
		if s.AvgCTR > 0 {
			s.CTRDelta = round2((s.AvgCTR - prevAvgCTR) / s.AvgCTR * 100)
		}
		prevAvgROAS := prevSumROAS / float64(prevCount)
		if s.AvgROAS > 0 {
			s.ROASDelta = round2((s.AvgROAS - prevAvgROAS) / s.AvgROAS * 100)
		}
	}

	if lastSync != nil && time.Since(*lastSync) > 7*time.Hour {
		s.IsStale = true
	}
	return s, nil
}

func (uc *DashboardUseCase) CampaignsWithMetrics(ctx context.Context, userID string) ([]*CampaignWithMetrics, error) {
	campaigns, err := uc.campaigns.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	from7d := time.Now().AddDate(0, 0, -7)
	from30d := time.Now().AddDate(0, 0, -30)
	to := time.Now()

	result := make([]*CampaignWithMetrics, 0, len(campaigns))
	for _, c := range campaigns {
		cm := &CampaignWithMetrics{Campaign: c}

		rows30, _ := uc.insights.ListByCampaign(ctx, c.ID, from30d, to)
		for _, r := range rows30 {
			cm.Spend30d += r.Spend
			cm.Leads30d += r.Leads
		}
		cm.Spend30d = round2(cm.Spend30d)

		rows7, _ := uc.insights.ListByCampaign(ctx, c.ID, from7d, to)
		if n := len(rows7); n > 0 {
			var sCTR, sCPC, sROAS float64
			for _, r := range rows7 {
				sCTR += r.CTR
				sCPC += r.CPC
				sROAS += r.ROAS
			}
			cm.AvgCTR7d = round2(sCTR / float64(n))
			cm.AvgCPC7d = round2(sCPC / float64(n))
			cm.AvgROAS7d = round2(sROAS / float64(n))
		}
		result = append(result, cm)
	}
	return result, nil
}

func (uc *DashboardUseCase) Anomalies(ctx context.Context, userID string) ([]*domain.Anomaly, error) {
	return uc.anomalies.ListActive(ctx, userID)
}

func (uc *DashboardUseCase) BudgetSuggestions(ctx context.Context, userID string) ([]*domain.BudgetSuggestion, error) {
	return uc.budget.ListByUser(ctx, userID)
}

func (uc *DashboardUseCase) CampaignInsights(ctx context.Context, campaignID string, from, to time.Time) ([]*domain.CampaignInsight, error) {
	return uc.insights.ListByCampaign(ctx, campaignID, from, to)
}

func (uc *DashboardUseCase) Recommendations(ctx context.Context, userID string) ([]*domain.Recommendation, error) {
	campaigns, err := uc.campaigns.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	var all []*domain.Recommendation
	for _, c := range campaigns {
		recs, err := uc.recommend.ListByCampaign(ctx, c.ID)
		if err != nil {
			continue
		}
		all = append(all, recs...)
	}
	return all, nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
