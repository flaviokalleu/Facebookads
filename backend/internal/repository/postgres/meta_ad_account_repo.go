package postgres

import (
	"context"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetaAdAccountRepo struct {
	db *pgxpool.Pool
}

func NewMetaAdAccountRepo(db *pgxpool.Pool) *MetaAdAccountRepo {
	return &MetaAdAccountRepo{db: db}
}

func (r *MetaAdAccountRepo) Upsert(ctx context.Context, a *domain.MetaAdAccount) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO meta_ad_accounts
		  (meta_id, bm_id, user_id, name, currency, timezone_name,
		   account_status, disable_reason, spend_cap, amount_spent, balance,
		   access_kind, raw)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (meta_id) DO UPDATE SET
		  bm_id          = EXCLUDED.bm_id,
		  user_id        = EXCLUDED.user_id,
		  name           = EXCLUDED.name,
		  currency       = EXCLUDED.currency,
		  timezone_name  = EXCLUDED.timezone_name,
		  account_status = EXCLUDED.account_status,
		  disable_reason = EXCLUDED.disable_reason,
		  spend_cap      = EXCLUDED.spend_cap,
		  amount_spent   = EXCLUDED.amount_spent,
		  balance        = EXCLUDED.balance,
		  access_kind    = EXCLUDED.access_kind,
		  raw            = EXCLUDED.raw,
		  synced_at      = now(),
		  updated_at     = now()
		RETURNING id, synced_at, created_at, updated_at
	`,
		a.MetaID, a.BMID, a.UserID, a.Name, a.Currency, a.TimezoneName,
		a.AccountStatus, a.DisableReason, a.SpendCap, a.AmountSpent, a.Balance,
		a.AccessKind, a.Raw,
	).Scan(&a.ID, &a.SyncedAt, &a.CreatedAt, &a.UpdatedAt)
}

func (r *MetaAdAccountRepo) ListByUser(ctx context.Context, userID string) ([]*domain.MetaAdAccount, error) {
	return r.scanMany(ctx, `
		SELECT id, meta_id, bm_id, user_id,
		       COALESCE(name,''), COALESCE(currency,''), COALESCE(timezone_name,''),
		       COALESCE(account_status,0), COALESCE(disable_reason,0),
		       COALESCE(spend_cap,0), COALESCE(amount_spent,0), COALESCE(balance,0),
		       COALESCE(access_kind,''), raw, synced_at, created_at, updated_at
		FROM meta_ad_accounts WHERE user_id=$1
		ORDER BY name NULLS LAST
	`, userID)
}

func (r *MetaAdAccountRepo) ListByBM(ctx context.Context, bmMetaID string) ([]*domain.MetaAdAccount, error) {
	return r.scanMany(ctx, `
		SELECT a.id, a.meta_id, a.bm_id, a.user_id,
		       COALESCE(a.name,''), COALESCE(a.currency,''), COALESCE(a.timezone_name,''),
		       COALESCE(a.account_status,0), COALESCE(a.disable_reason,0),
		       COALESCE(a.spend_cap,0), COALESCE(a.amount_spent,0), COALESCE(a.balance,0),
		       COALESCE(a.access_kind,''), a.raw, a.synced_at, a.created_at, a.updated_at
		FROM meta_ad_accounts a
		JOIN business_managers b ON b.id = a.bm_id
		WHERE b.meta_id=$1
		ORDER BY a.name NULLS LAST
	`, bmMetaID)
}

func (r *MetaAdAccountRepo) ListPersonalByUser(ctx context.Context, userID string) ([]*domain.MetaAdAccount, error) {
	return r.scanMany(ctx, `
		SELECT id, meta_id, bm_id, user_id,
		       COALESCE(name,''), COALESCE(currency,''), COALESCE(timezone_name,''),
		       COALESCE(account_status,0), COALESCE(disable_reason,0),
		       COALESCE(spend_cap,0), COALESCE(amount_spent,0), COALESCE(balance,0),
		       COALESCE(access_kind,''), raw, synced_at, created_at, updated_at
		FROM meta_ad_accounts WHERE user_id=$1 AND bm_id IS NULL
		ORDER BY name NULLS LAST
	`, userID)
}

func (r *MetaAdAccountRepo) scanMany(ctx context.Context, sql string, args ...any) ([]*domain.MetaAdAccount, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.MetaAdAccount
	for rows.Next() {
		var a domain.MetaAdAccount
		if err := rows.Scan(&a.ID, &a.MetaID, &a.BMID, &a.UserID,
			&a.Name, &a.Currency, &a.TimezoneName,
			&a.AccountStatus, &a.DisableReason,
			&a.SpendCap, &a.AmountSpent, &a.Balance,
			&a.AccessKind, &a.Raw, &a.SyncedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &a)
	}
	return result, rows.Err()
}
