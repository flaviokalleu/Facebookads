package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/domain"
)

type BudgetSuggestionRepo struct{ db *pgxpool.Pool }

func NewBudgetSuggestionRepo(db *pgxpool.Pool) *BudgetSuggestionRepo {
	return &BudgetSuggestionRepo{db: db}
}

func (r *BudgetSuggestionRepo) BulkCreate(ctx context.Context, suggestions []*domain.BudgetSuggestion) error {
	for _, s := range suggestions {
		err := r.db.QueryRow(ctx, `
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
			return err
		}
	}
	return nil
}

func (r *BudgetSuggestionRepo) ListByUser(ctx context.Context, userID string) ([]*domain.BudgetSuggestion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT bs.id, bs.user_id, bs.ad_account_id, bs.campaign_id,
		       bs.current_budget, bs.suggested_budget,
		       bs.change_reason, bs.should_pause,
		       bs.expected_roas_improvement, bs.portfolio_summary,
		       bs.model_used, bs.is_applied, bs.created_at,
		       COALESCE(c.name, '') AS campaign_name
		FROM budget_suggestions bs
		LEFT JOIN campaigns c ON c.id = bs.campaign_id::uuid
		WHERE bs.user_id = $1 AND bs.is_applied = false
		ORDER BY bs.created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.BudgetSuggestion
	for rows.Next() {
		s := &domain.BudgetSuggestion{}
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.AdAccountID, &s.CampaignID,
			&s.CurrentBudget, &s.SuggestedBudget,
			&s.ChangeReason, &s.ShouldPause,
			&s.ExpectedROASImprovement, &s.PortfolioSummary,
			&s.ModelUsed, &s.IsApplied, &s.CreatedAt, &s.CampaignName,
		); err != nil {
			return nil, err
		}
		if s.CurrentBudget > 0 {
			s.SuggestedChange = (s.SuggestedBudget - s.CurrentBudget) / s.CurrentBudget * 100
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *BudgetSuggestionRepo) MarkApplied(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE budget_suggestions SET is_applied = true WHERE id = $1`, id)
	return err
}
