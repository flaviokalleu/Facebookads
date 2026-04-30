package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserTokenRepo struct {
	db *pgxpool.Pool
}

func NewUserTokenRepo(db *pgxpool.Pool) *UserTokenRepo {
	return &UserTokenRepo{db: db}
}

func (r *UserTokenRepo) Upsert(ctx context.Context, t *domain.UserToken) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO user_tokens (user_id, ad_account_id, encrypted_token, token_expiry)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, ad_account_id) DO UPDATE SET
		  encrypted_token = EXCLUDED.encrypted_token,
		  token_expiry    = EXCLUDED.token_expiry,
		  updated_at      = now()
		RETURNING id, created_at, updated_at
	`, t.UserID, t.AdAccountID, t.EncryptedToken, t.TokenExpiry).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *UserTokenRepo) GetByUserAndAccount(ctx context.Context, userID, adAccountID string) (*domain.UserToken, error) {
	var t domain.UserToken
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, ad_account_id, encrypted_token, token_expiry, created_at, updated_at
		FROM user_tokens WHERE user_id = $1 AND ad_account_id = $2
	`, userID, adAccountID).
		Scan(&t.ID, &t.UserID, &t.AdAccountID, &t.EncryptedToken, &t.TokenExpiry, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: user token", domain.ErrNotFound)
		}
		return nil, err
	}
	return &t, nil
}

func (r *UserTokenRepo) ListByUser(ctx context.Context, userID string) ([]*domain.UserToken, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, ad_account_id, encrypted_token, token_expiry, created_at, updated_at
		FROM user_tokens WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.UserToken
	for rows.Next() {
		var t domain.UserToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.AdAccountID, &t.EncryptedToken,
			&t.TokenExpiry, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	return result, rows.Err()
}
