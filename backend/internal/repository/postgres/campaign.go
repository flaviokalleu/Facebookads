package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignRepo struct {
	db *pgxpool.Pool
}

func NewCampaignRepo(db *pgxpool.Pool) *CampaignRepo {
	return &CampaignRepo{db: db}
}

func (r *CampaignRepo) Upsert(ctx context.Context, c *domain.Campaign) error {
	err := r.db.QueryRow(ctx, `
		INSERT INTO campaigns
		  (meta_campaign_id, user_id, ad_account_id, name, objective, status,
		   daily_budget, lifetime_budget, health_status, last_synced_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, now())
		ON CONFLICT (user_id, meta_campaign_id) DO UPDATE SET
		  name            = EXCLUDED.name,
		  objective       = EXCLUDED.objective,
		  status          = EXCLUDED.status,
		  daily_budget    = EXCLUDED.daily_budget,
		  lifetime_budget = EXCLUDED.lifetime_budget,
		  last_synced_at  = EXCLUDED.last_synced_at,
		  updated_at      = now(),
		  deleted_at      = NULL
		RETURNING id, created_at, updated_at
	`,
		c.MetaCampaignID, c.UserID, c.AdAccountID, c.Name, c.Objective, c.Status,
		c.DailyBudget, c.LifetimeBudget, c.HealthStatus, c.LastSyncedAt,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	return err
}

func (r *CampaignRepo) GetByID(ctx context.Context, id string) (*domain.Campaign, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, meta_campaign_id, user_id, ad_account_id, name, objective, status,
		       daily_budget, lifetime_budget, health_status, last_synced_at, created_at, updated_at
		FROM campaigns WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return scanCampaign(row)
}

func (r *CampaignRepo) GetByMetaID(ctx context.Context, userID, metaID string) (*domain.Campaign, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, meta_campaign_id, user_id, ad_account_id, name, objective, status,
		       daily_budget, lifetime_budget, health_status, last_synced_at, created_at, updated_at
		FROM campaigns WHERE user_id = $1 AND meta_campaign_id = $2 AND deleted_at IS NULL
	`, userID, metaID)
	return scanCampaign(row)
}

func (r *CampaignRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Campaign, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, meta_campaign_id, user_id, ad_account_id, name, objective, status,
		       daily_budget, lifetime_budget, health_status, last_synced_at, created_at, updated_at
		FROM campaigns WHERE user_id = $1 AND deleted_at IS NULL ORDER BY name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Campaign
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *CampaignRepo) UpdateHealthStatus(ctx context.Context, id string, status domain.HealthStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE campaigns SET health_status = $1, updated_at = now() WHERE id = $2`,
		status, id,
	)
	return err
}

func (r *CampaignRepo) Update(ctx context.Context, c *domain.Campaign) error {
	_, err := r.db.Exec(ctx, `
		UPDATE campaigns SET
		  name = $1, status = $2, daily_budget = $3, lifetime_budget = $4,
		  objective = $5, health_status = $6, updated_at = now()
		WHERE id = $7 AND deleted_at IS NULL
	`, c.Name, c.Status, c.DailyBudget, c.LifetimeBudget,
		c.Objective, c.HealthStatus, c.ID)
	return err
}

func (r *CampaignRepo) MarkDeleted(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE campaigns SET deleted_at = now(), updated_at = now() WHERE id = $1`,
		id,
	)
	return err
}

// ─── scanner ─────────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanCampaign(s scanner) (*domain.Campaign, error) {
	var c domain.Campaign
	var lastSynced *time.Time
	err := s.Scan(
		&c.ID, &c.MetaCampaignID, &c.UserID, &c.AdAccountID,
		&c.Name, &c.Objective, &c.Status,
		&c.DailyBudget, &c.LifetimeBudget,
		&c.HealthStatus, &lastSynced,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: campaign", domain.ErrNotFound)
		}
		return nil, err
	}
	c.LastSyncedAt = lastSynced
	return &c, nil
}
