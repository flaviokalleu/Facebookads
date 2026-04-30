package postgres

import (
	"context"
	"time"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LLMUsageRepo struct {
	db *pgxpool.Pool
}

func NewLLMUsageRepo(db *pgxpool.Pool) *LLMUsageRepo {
	return &LLMUsageRepo{db: db}
}

func (r *LLMUsageRepo) Create(ctx context.Context, u *domain.LLMUsage) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO llm_usage (user_id, task_type, provider, model, input_tokens, output_tokens,
		                       cost_usd, latency_ms, success, error_message)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at
	`, u.UserID, u.TaskType, u.Provider, u.Model,
		u.InputTokens, u.OutputTokens, u.CostUSD, u.LatencyMs,
		u.Success, u.ErrorMessage).
		Scan(&u.ID, &u.CreatedAt)
}

func (r *LLMUsageRepo) SummaryByProvider(ctx context.Context, userID string, from, to time.Time) ([]repository.LLMProviderSummary, error) {
	rows, err := r.db.Query(ctx, `
		SELECT provider, model,
		       COUNT(*)::int              AS requests,
		       SUM(input_tokens)::int     AS input_tokens,
		       SUM(output_tokens)::int    AS output_tokens,
		       SUM(cost_usd)              AS total_cost_usd,
		       AVG(latency_ms)            AS avg_latency_ms
		FROM llm_usage
		WHERE ($1::uuid IS NULL OR user_id = $1)
		  AND created_at BETWEEN $2 AND $3
		GROUP BY provider, model
		ORDER BY total_cost_usd DESC
	`, userID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []repository.LLMProviderSummary
	for rows.Next() {
		var s repository.LLMProviderSummary
		if err := rows.Scan(&s.Provider, &s.Model, &s.Requests,
			&s.InputTokens, &s.OutputTokens, &s.TotalCostUSD, &s.AvgLatencyMs); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *LLMUsageRepo) DailyCost(ctx context.Context, userID string, days int) ([]repository.LLMDailyCost, error) {
	rows, err := r.db.Query(ctx, `
		SELECT date_trunc('day', created_at)::date AS date,
		       provider,
		       SUM(cost_usd) AS cost_usd
		FROM llm_usage
		WHERE ($1::uuid IS NULL OR user_id = $1)
		  AND created_at >= CURRENT_DATE - $2::int
		GROUP BY 1, 2
		ORDER BY 1, 2
	`, userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []repository.LLMDailyCost
	for rows.Next() {
		var d repository.LLMDailyCost
		if err := rows.Scan(&d.Date, &d.Provider, &d.CostUSD); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}
