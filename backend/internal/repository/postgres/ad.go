package postgres

import (
	"context"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdRepo struct {
	db *pgxpool.Pool
}

func NewAdRepo(db *pgxpool.Pool) *AdRepo {
	return &AdRepo{db: db}
}

func (r *AdRepo) Upsert(ctx context.Context, a *domain.Ad) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO ads
		  (meta_ad_id, ad_set_id, name, status, creative_title, creative_body,
		   image_url, cta_type, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now())
		ON CONFLICT (ad_set_id, meta_ad_id) DO UPDATE SET
		  name = EXCLUDED.name,
		  status = EXCLUDED.status,
		  creative_title = EXCLUDED.creative_title,
		  creative_body = EXCLUDED.creative_body,
		  image_url = EXCLUDED.image_url,
		  cta_type = EXCLUDED.cta_type,
		  updated_at = now(),
		  deleted_at = NULL
		RETURNING id, created_at, updated_at
	`, a.MetaAdID, a.AdSetID, a.Name, a.Status, a.CreativeTitle,
		a.CreativeBody, a.ImageURL, a.CTAType,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (r *AdRepo) ListByAdSet(ctx context.Context, adSetID string) ([]*domain.Ad, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, meta_ad_id, ad_set_id, name, status,
		       creative_title, creative_body, image_url, cta_type,
		       created_at, updated_at
		FROM ads WHERE ad_set_id = $1 AND deleted_at IS NULL ORDER BY name
	`, adSetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Ad
	for rows.Next() {
		a := &domain.Ad{}
		if err := rows.Scan(&a.ID, &a.MetaAdID, &a.AdSetID, &a.Name, &a.Status,
			&a.CreativeTitle, &a.CreativeBody, &a.ImageURL, &a.CTAType,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}
