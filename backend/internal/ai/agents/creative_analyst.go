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

const creativeAnalystPrompt = `You are a Meta Ads creative strategist.

Analyze these ad creatives and their performance metrics to identify what makes the top performers succeed and what is causing the bottom performers to fail.

Top 5 Creatives (by CTR):
%s

Bottom 5 Creatives (by CTR):
%s

Respond in JSON:
{
  "winning_patterns": ["pattern 1", "pattern 2"],
  "losing_patterns": ["pattern 1", "pattern 2"],
  "headline_insights": "what works in headlines",
  "cta_insights": "which CTAs convert best",
  "recommendations": ["specific creative suggestion 1", "specific creative suggestion 2"]
}`

type creativeAnalystResponse struct {
	WinningPatterns  []string `json:"winning_patterns"`
	LosingPatterns   []string `json:"losing_patterns"`
	HeadlineInsights string   `json:"headline_insights"`
	CTAInsights      string   `json:"cta_insights"`
	Recommendations  []string `json:"recommendations"`
}

// CreativeAnalyst analyzes ad creatives and their performance to identify patterns.
// Primary model: gpt-5-4 (routed via TaskCreativeAnalysis).
type CreativeAnalyst struct {
	db     *pgxpool.Pool
	router *ai.Router
}

func NewCreativeAnalyst(db *pgxpool.Pool, router *ai.Router) *CreativeAnalyst {
	return &CreativeAnalyst{db: db, router: router}
}

// RunAnalyzeAll analyzes creatives for all campaigns belonging to a user.
func (ca *CreativeAnalyst) RunAnalyzeAll(ctx context.Context, userID string) error {
	adAccountID, top, bottom, err := ca.fetchTopBottomCreatives(ctx, userID)
	if err != nil {
		return fmt.Errorf("creative_analyst: fetch creatives: %w", err)
	}
	if len(top) == 0 && len(bottom) == 0 {
		slog.Info("creative_analyst: no creatives to analyze", "user_id", userID)
		return nil
	}

	userPrompt := fmt.Sprintf(creativeAnalystPrompt,
		mustMarshal(top),
		mustMarshal(bottom),
	)

	resp, err := ca.router.Complete(ctx, ai.TaskCreativeAnalysis, ai.CompletionRequest{
		SystemPrompt: "You are a Meta Ads creative strategist. Respond only with valid JSON.",
		UserPrompt:   userPrompt,
		MaxTokens:    1500,
		Temperature:  0.3,
		JSONMode:     true,
	})
	if err != nil {
		return fmt.Errorf("router: %w", err)
	}

	var out creativeAnalystResponse
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &out); err != nil {
		return fmt.Errorf("parse response: %w (raw: %s)", err, truncate(resp.Content, 300))
	}

	wJSON, _ := json.Marshal(out.WinningPatterns)
	lJSON, _ := json.Marshal(out.LosingPatterns)
	rJSON, _ := json.Marshal(out.Recommendations)

	_, err = ca.db.Exec(ctx, `
		INSERT INTO creative_insights
		  (user_id, ad_account_id, winning_patterns, losing_patterns, headline_insights, cta_insights, recommendations, model_used)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, userID, adAccountID, string(wJSON), string(lJSON),
		out.HeadlineInsights, out.CTAInsights, string(rJSON),
		resp.Provider+"/"+resp.ModelUsed,
	)
	if err != nil {
		return fmt.Errorf("save creative insights: %w", err)
	}

	slog.Info("creative_analyst: insights saved",
		"user_id", userID,
		"top_count", len(top),
		"bottom_count", len(bottom),
		"model", resp.ModelUsed,
	)
	return nil
}

// fetchTopBottomCreatives gets the top and bottom 5 ads by CTR across user's campaigns.
func (ca *CreativeAnalyst) fetchTopBottomCreatives(ctx context.Context, userID string) (adAccountID string, top, bottom []map[string]any, err error) {
	// Get the user's ad account ID
	err = ca.db.QueryRow(ctx, `
		SELECT ad_account_id FROM campaigns WHERE user_id = $1 AND deleted_at IS NULL LIMIT 1
	`, userID).Scan(&adAccountID)
	if err != nil {
		return adAccountID, nil, nil, err
	}

	// Top 5 by creative CTR (using campaign insights for proxy)
	topRows, err := ca.db.Query(ctx, `
		SELECT a.creative_title, a.creative_body, a.cta_type, a.name,
		       COALESCE(AVG(ci.ctr), 0) as avg_ctr,
		       SUM(ci.leads) as total_leads,
		       COALESCE(AVG(ci.frequency), 0) as avg_freq
		FROM ads a
		JOIN ad_sets ads2 ON ads2.id = a.ad_set_id
		JOIN campaigns c ON c.id = ads2.campaign_id
		LEFT JOIN campaign_insights ci ON ci.campaign_id = c.id AND ci.date >= CURRENT_DATE - 30
		WHERE c.user_id = $1 AND c.deleted_at IS NULL AND a.deleted_at IS NULL
		GROUP BY a.id, a.creative_title, a.creative_body, a.cta_type, a.name
		HAVING AVG(ci.ctr) > 0
		ORDER BY avg_ctr DESC
		LIMIT 5
	`, userID)
	if err != nil {
		return adAccountID, nil, nil, err
	}
	defer topRows.Close()
	top, err = scanCreativeRows(topRows)
	if err != nil {
		return adAccountID, nil, nil, err
	}

	// Bottom 5 by CTR
	bottomRows, err := ca.db.Query(ctx, `
		SELECT a.creative_title, a.creative_body, a.cta_type, a.name,
		       COALESCE(AVG(ci.ctr), 0) as avg_ctr,
		       SUM(ci.leads) as total_leads,
		       COALESCE(AVG(ci.frequency), 0) as avg_freq
		FROM ads a
		JOIN ad_sets ads2 ON ads2.id = a.ad_set_id
		JOIN campaigns c ON c.id = ads2.campaign_id
		LEFT JOIN campaign_insights ci ON ci.campaign_id = c.id AND ci.date >= CURRENT_DATE - 30
		WHERE c.user_id = $1 AND c.deleted_at IS NULL AND a.deleted_at IS NULL
		GROUP BY a.id, a.creative_title, a.creative_body, a.cta_type, a.name
		HAVING AVG(ci.ctr) > 0
		ORDER BY avg_ctr ASC
		LIMIT 5
	`, userID)
	if err != nil {
		return adAccountID, nil, nil, err
	}
	defer bottomRows.Close()
	bottom, err = scanCreativeRows(bottomRows)
	if err != nil {
		return adAccountID, nil, nil, err
	}

	return adAccountID, top, bottom, nil
}

func scanCreativeRows(rows interface{ Scan(...any) error; Next() bool; Err() error }) ([]map[string]any, error) {
	var result []map[string]any
	for rows.Next() {
		var title, body, cta, name string
		var ctr, freq float64
		var leads int64
		if err := rows.Scan(&title, &body, &cta, &name, &ctr, &leads, &freq); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{
			"title":     title,
			"body":      body,
			"cta":       cta,
			"name":      name,
			"ctr":       fmt.Sprintf("%.2f%%", ctr),
			"leads":     leads,
			"frequency": fmt.Sprintf("%.1f", freq),
		})
	}
	return result, rows.Err()
}

// unused stubs to prevent import errors
var _ = domain.HealthScaling
var _ = time.Now
