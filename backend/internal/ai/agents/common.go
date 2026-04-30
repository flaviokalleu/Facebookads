package agents

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/facebookads/backend/internal/domain"
)

// ─── Helpers shared across agents ──────────────────────────────────────────────

func cleanJSON(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func mustMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return string(b)
}

func mustMarshalSlice(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`[{"error":"%s"}]`, err.Error())
	}
	return string(b)
}

func aggregateInsights(insights []*domain.CampaignInsight) (ctr, cpc, roas, spend, leads, freq float64) {
	if len(insights) == 0 {
		return
	}
	var ctrSum, cpcSum, roasSum, spendSum, leadsSum, freqSum float64
	n := float64(len(insights))
	for _, i := range insights {
		ctrSum += i.CTR
		cpcSum += i.CPC
		roasSum += i.ROAS
		spendSum += i.Spend
		leadsSum += float64(i.Leads)
		freqSum += i.Frequency
	}
	return ctrSum / n, cpcSum / n, roasSum / n, spendSum, leadsSum, freqSum / n
}

func trendString(insights []*domain.CampaignInsight, get func(*domain.CampaignInsight) float64, days int) string {
	if len(insights) < days {
		return "insufficient data"
	}
	last := insights[len(insights)-days:]
	var parts []string
	for _, i := range last {
		parts = append(parts, fmt.Sprintf("%s: %.2f", i.Date.Format("2006-01-02"), get(i)))
	}
	return strings.Join(parts, ", ")
}

func makeMapField(data map[string]any, key string) map[string]any {
	if data == nil {
		return nil
	}
	if v, ok := data[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func makeSliceField(data map[string]any, key string) []any {
	if data == nil {
		return nil
	}
	if v, ok := data[key]; ok {
		if s, ok := v.([]any); ok {
			return s
		}
	}
	return nil
}

func floatField(data map[string]any, key string) float64 {
	if v, ok := data[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case json.Number:
			f, _ := n.Float64()
			return f
		}
	}
	return 0
}

func stringField(data map[string]any, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type InsightSnapshot struct {
	Date    time.Time `json:"date"`
	Spend   float64   `json:"spend"`
	CTR     float64   `json:"ctr"`
	CPC     float64   `json:"cpc"`
	ROAS    float64   `json:"roas"`
	Leads   int64     `json:"leads"`
	Impressions int64 `json:"impressions"`
}

func dailySnapshots(insights []*domain.CampaignInsight) []InsightSnapshot {
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
