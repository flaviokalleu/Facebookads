package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BusinessManagerRepo struct {
	db *pgxpool.Pool
}

func NewBusinessManagerRepo(db *pgxpool.Pool) *BusinessManagerRepo {
	return &BusinessManagerRepo{db: db}
}

func (r *BusinessManagerRepo) Upsert(ctx context.Context, b *domain.BusinessManager) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO business_managers
		  (meta_id, user_id, name, verification_status, timezone_id, vertical, raw)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (meta_id) DO UPDATE SET
		  user_id             = EXCLUDED.user_id,
		  name                = EXCLUDED.name,
		  verification_status = EXCLUDED.verification_status,
		  timezone_id         = EXCLUDED.timezone_id,
		  vertical            = EXCLUDED.vertical,
		  raw                 = EXCLUDED.raw,
		  synced_at           = now(),
		  updated_at          = now()
		RETURNING id, synced_at, created_at, updated_at
	`, b.MetaID, b.UserID, b.Name, b.VerificationStatus, b.TimezoneID, b.Vertical, b.Raw).
		Scan(&b.ID, &b.SyncedAt, &b.CreatedAt, &b.UpdatedAt)
}

func (r *BusinessManagerRepo) ListByUser(ctx context.Context, userID string) ([]*domain.BusinessManager, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, meta_id, user_id,
		       COALESCE(name,''), COALESCE(verification_status,''),
		       COALESCE(timezone_id,0), COALESCE(vertical,''),
		       raw, synced_at, created_at, updated_at
		FROM business_managers WHERE user_id=$1 ORDER BY name NULLS LAST
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.BusinessManager
	for rows.Next() {
		var b domain.BusinessManager
		if err := rows.Scan(&b.ID, &b.MetaID, &b.UserID, &b.Name, &b.VerificationStatus,
			&b.TimezoneID, &b.Vertical, &b.Raw, &b.SyncedAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &b)
	}
	return result, rows.Err()
}

func (r *BusinessManagerRepo) GetByMetaID(ctx context.Context, metaID string) (*domain.BusinessManager, error) {
	var b domain.BusinessManager
	err := r.db.QueryRow(ctx, `
		SELECT id, meta_id, user_id,
		       COALESCE(name,''), COALESCE(verification_status,''),
		       COALESCE(timezone_id,0), COALESCE(vertical,''),
		       raw, synced_at, created_at, updated_at
		FROM business_managers WHERE meta_id=$1
	`, metaID).Scan(&b.ID, &b.MetaID, &b.UserID, &b.Name, &b.VerificationStatus,
		&b.TimezoneID, &b.Vertical, &b.Raw, &b.SyncedAt, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: business manager", domain.ErrNotFound)
		}
		return nil, err
	}
	return &b, nil
}
