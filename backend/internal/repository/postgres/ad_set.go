package postgres

import (
	"context"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdSetRepo struct {
	db *pgxpool.Pool
}

func NewAdSetRepo(db *pgxpool.Pool) *AdSetRepo {
	return &AdSetRepo{db: db}
}

func (r *AdSetRepo) Upsert(ctx context.Context, a *domain.AdSet) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO ad_sets
		  (meta_ad_set_id, campaign_id, name, status, daily_budget,
		   optimization_goal, billing_event, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7, now())
		ON CONFLICT (campaign_id, meta_ad_set_id) DO UPDATE SET
		  name = EXCLUDED.name,
		  status = EXCLUDED.status,
		  daily_budget = EXCLUDED.daily_budget,
		  optimization_goal = EXCLUDED.optimization_goal,
		  billing_event = EXCLUDED.billing_event,
		  updated_at = now(),
		  deleted_at = NULL
		RETURNING id, created_at, updated_at
	`, a.MetaAdSetID, a.CampaignID, a.Name, a.Status, a.DailyBudget,
		a.OptimizationGoal, a.BillingEvent,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AdSetRepo) ListByCampaign(ctx context.Context, campaignID string) ([]*domain.AdSet, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, meta_ad_set_id, campaign_id, name, status, daily_budget,
		       optimization_goal, billing_event, created_at, updated_at
		FROM ad_sets WHERE campaign_id = $1 AND deleted_at IS NULL ORDER BY name
	`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.AdSet
	for rows.Next() {
		a := &domain.AdSet{}
		if err := rows.Scan(&a.ID, &a.MetaAdSetID, &a.CampaignID, &a.Name, &a.Status,
			&a.DailyBudget, &a.OptimizationGoal, &a.BillingEvent,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
