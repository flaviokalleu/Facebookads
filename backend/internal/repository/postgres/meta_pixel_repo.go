package postgres

import (
	"context"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetaPixelRepo struct {
	db *pgxpool.Pool
}

func NewMetaPixelRepo(db *pgxpool.Pool) *MetaPixelRepo {
	return &MetaPixelRepo{db: db}
}

func (r *MetaPixelRepo) Upsert(ctx context.Context, p *domain.MetaPixel) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO meta_pixels
		  (meta_id, bm_id, account_id, user_id, name, last_fired, is_active, raw)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (meta_id) DO UPDATE SET
		  bm_id      = EXCLUDED.bm_id,
		  account_id = EXCLUDED.account_id,
		  user_id    = EXCLUDED.user_id,
		  name       = EXCLUDED.name,
		  last_fired = EXCLUDED.last_fired,
		  is_active  = EXCLUDED.is_active,
		  raw        = EXCLUDED.raw,
		  synced_at  = now(),
		  updated_at = now()
		RETURNING id, synced_at, created_at, updated_at
	`,
		p.MetaID, p.BMID, p.AccountID, p.UserID, p.Name,
		p.LastFired, p.IsActive, p.Raw,
	).Scan(&p.ID, &p.SyncedAt, &p.CreatedAt, &p.UpdatedAt)
}

func (r *MetaPixelRepo) ListByUser(ctx context.Context, userID string) ([]*domain.MetaPixel, error) {
	return r.scanMany(ctx, `
		SELECT id, meta_id, bm_id, account_id, user_id,
		       COALESCE(name,''), last_fired, is_active,
		       raw, synced_at, created_at, updated_at
		FROM meta_pixels WHERE user_id=$1
		ORDER BY name NULLS LAST
	`, userID)
}

func (r *MetaPixelRepo) ListByBM(ctx context.Context, bmMetaID string) ([]*domain.MetaPixel, error) {
	return r.scanMany(ctx, `
		SELECT p.id, p.meta_id, p.bm_id, p.account_id, p.user_id,
		       COALESCE(p.name,''), p.last_fired, p.is_active,
		       p.raw, p.synced_at, p.created_at, p.updated_at
		FROM meta_pixels p
		JOIN business_managers b ON b.id = p.bm_id
		WHERE b.meta_id=$1
		ORDER BY p.name NULLS LAST
	`, bmMetaID)
}

func (r *MetaPixelRepo) scanMany(ctx context.Context, sql string, args ...any) ([]*domain.MetaPixel, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.MetaPixel
	for rows.Next() {
		var p domain.MetaPixel
		if err := rows.Scan(&p.ID, &p.MetaID, &p.BMID, &p.AccountID, &p.UserID,
			&p.Name, &p.LastFired, &p.IsActive,
			&p.Raw, &p.SyncedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &p)
	}
	return result, rows.Err()
}
