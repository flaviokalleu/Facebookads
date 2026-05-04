package postgres

import (
	"context"

	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetaPageRepo struct {
	db  *pgxpool.Pool
	cfg *config.Service
}

func NewMetaPageRepo(db *pgxpool.Pool, cfg *config.Service) *MetaPageRepo {
	return &MetaPageRepo{db: db, cfg: cfg}
}

func (r *MetaPageRepo) Upsert(ctx context.Context, p *domain.MetaPage) error {
	// Encrypt page token if provided in plaintext form. Caller signals plaintext
	// by using EncryptedPageToken field BEFORE encryption. We always run it
	// through cfg.Encrypt at this layer if non-empty and not already a base64
	// blob — for simplicity we always encrypt here.
	enc := p.EncryptedPageToken
	if enc != "" {
		var err error
		enc, err = r.cfg.Encrypt(p.EncryptedPageToken)
		if err != nil {
			return err
		}
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO meta_pages
		  (meta_id, bm_id, user_id, name, category, fan_count,
		   encrypted_page_token, ig_user_id, raw)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (meta_id) DO UPDATE SET
		  bm_id                = EXCLUDED.bm_id,
		  user_id              = EXCLUDED.user_id,
		  name                 = EXCLUDED.name,
		  category             = EXCLUDED.category,
		  fan_count            = EXCLUDED.fan_count,
		  encrypted_page_token = COALESCE(NULLIF(EXCLUDED.encrypted_page_token,''), meta_pages.encrypted_page_token),
		  ig_user_id           = EXCLUDED.ig_user_id,
		  raw                  = EXCLUDED.raw,
		  synced_at            = now(),
		  updated_at           = now()
		RETURNING id, synced_at, created_at, updated_at
	`,
		p.MetaID, p.BMID, p.UserID, p.Name, p.Category, p.FanCount,
		enc, p.IGUserID, p.Raw,
	).Scan(&p.ID, &p.SyncedAt, &p.CreatedAt, &p.UpdatedAt)
}

func (r *MetaPageRepo) ListByUser(ctx context.Context, userID string) ([]*domain.MetaPage, error) {
	return r.scanMany(ctx, `
		SELECT id, meta_id, bm_id, user_id,
		       COALESCE(name,''), COALESCE(category,''), COALESCE(fan_count,0),
		       COALESCE(encrypted_page_token,''), COALESCE(ig_user_id,''),
		       raw, synced_at, created_at, updated_at
		FROM meta_pages WHERE user_id=$1
		ORDER BY name NULLS LAST
	`, userID)
}

func (r *MetaPageRepo) ListByBM(ctx context.Context, bmMetaID string) ([]*domain.MetaPage, error) {
	return r.scanMany(ctx, `
		SELECT p.id, p.meta_id, p.bm_id, p.user_id,
		       COALESCE(p.name,''), COALESCE(p.category,''), COALESCE(p.fan_count,0),
		       COALESCE(p.encrypted_page_token,''), COALESCE(p.ig_user_id,''),
		       p.raw, p.synced_at, p.created_at, p.updated_at
		FROM meta_pages p
		JOIN business_managers b ON b.id = p.bm_id
		WHERE b.meta_id=$1
		ORDER BY p.name NULLS LAST
	`, bmMetaID)
}

func (r *MetaPageRepo) scanMany(ctx context.Context, sql string, args ...any) ([]*domain.MetaPage, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.MetaPage
	for rows.Next() {
		var p domain.MetaPage
		if err := rows.Scan(&p.ID, &p.MetaID, &p.BMID, &p.UserID,
			&p.Name, &p.Category, &p.FanCount,
			&p.EncryptedPageToken, &p.IGUserID,
			&p.Raw, &p.SyncedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &p)
	}
	return result, rows.Err()
}
