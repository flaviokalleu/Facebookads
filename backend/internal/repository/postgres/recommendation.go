package postgres

import (
	"context"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RecommendationRepo struct {
	db *pgxpool.Pool
}

func NewRecommendationRepo(db *pgxpool.Pool) *RecommendationRepo {
	return &RecommendationRepo{db: db}
}

func (r *RecommendationRepo) BulkCreate(ctx context.Context, recs []*domain.Recommendation) error {
	for _, rec := range recs {
		err := r.db.QueryRow(ctx, `
			INSERT INTO recommendations (campaign_id, priority, category, action, expected_impact, rationale, model_used)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			RETURNING id, created_at, updated_at
		`, rec.CampaignID, rec.Priority, rec.Category, rec.Action,
			rec.ExpectedImpact, rec.Rationale, rec.ModelUsed).
			Scan(&rec.ID, &rec.CreatedAt, &rec.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RecommendationRepo) ListByCampaign(ctx context.Context, campaignID string) ([]*domain.Recommendation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, campaign_id, priority, category, action, expected_impact, rationale,
		       model_used, is_applied, created_at, updated_at
		FROM recommendations
		WHERE campaign_id = $1
		ORDER BY
		  CASE priority WHEN 'HIGH' THEN 1 WHEN 'MEDIUM' THEN 2 ELSE 3 END,
		  created_at DESC
	`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Recommendation
	for rows.Next() {
		var rec domain.Recommendation
		if err := rows.Scan(&rec.ID, &rec.CampaignID, &rec.Priority, &rec.Category,
			&rec.Action, &rec.ExpectedImpact, &rec.Rationale,
			&rec.ModelUsed, &rec.IsApplied, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &rec)
	}
	return result, rows.Err()
}

func (r *RecommendationRepo) MarkApplied(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE recommendations SET is_applied = true, updated_at = now() WHERE id = $1`, id)
	return err
}

