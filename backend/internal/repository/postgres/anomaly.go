package postgres

import (
	"context"
	"time"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnomalyRepo struct {
	db *pgxpool.Pool
}

func NewAnomalyRepo(db *pgxpool.Pool) *AnomalyRepo {
	return &AnomalyRepo{db: db}
}

func (r *AnomalyRepo) Create(ctx context.Context, a *domain.Anomaly) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO anomalies (campaign_id, type, severity, description, is_active, detected_at)
		VALUES ($1, $2, $3, $4, true, now())
		RETURNING id, created_at, updated_at
	`, a.CampaignID, a.Type, a.Severity, a.Description).
		Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AnomalyRepo) ListActive(ctx context.Context, userID string) ([]*domain.Anomaly, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.campaign_id, a.type, a.severity, a.description,
		       a.is_active, a.detected_at, a.resolved_at, a.created_at, a.updated_at
		FROM anomalies a
		JOIN campaigns c ON c.id = a.campaign_id
		WHERE c.user_id = $1 AND a.is_active = true AND c.deleted_at IS NULL
		ORDER BY
		  CASE a.severity WHEN 'HIGH' THEN 1 WHEN 'MEDIUM' THEN 2 ELSE 3 END,
		  a.detected_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAnomalies(rows)
}

func (r *AnomalyRepo) ListByCampaign(ctx context.Context, campaignID string) ([]*domain.Anomaly, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, campaign_id, type, severity, description,
		       is_active, detected_at, resolved_at, created_at, updated_at
		FROM anomalies WHERE campaign_id = $1
		ORDER BY detected_at DESC
	`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAnomalies(rows)
}

func (r *AnomalyRepo) Resolve(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE anomalies SET is_active = false, resolved_at = $1, updated_at = now()
		WHERE id = $2
	`, now, id)
	return err
}

func scanAnomalies(rows interface{ Next() bool; Scan(...any) error; Err() error }) ([]*domain.Anomaly, error) {
	var result []*domain.Anomaly
	for rows.Next() {
		var a domain.Anomaly
		if err := rows.Scan(
			&a.ID, &a.CampaignID, &a.Type, &a.Severity, &a.Description,
			&a.IsActive, &a.DetectedAt, &a.ResolvedAt,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, &a)
	}
	return result, rows.Err()
}
