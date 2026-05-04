package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MetaTokenRepo struct {
	db  *pgxpool.Pool
	cfg *config.Service
}

func NewMetaTokenRepo(db *pgxpool.Pool, cfg *config.Service) *MetaTokenRepo {
	return &MetaTokenRepo{db: db, cfg: cfg}
}

// Upsert inserts or updates a meta token. Caller must have already populated
// EncryptedToken with ciphertext (use cfg.Encrypt). PlainToken is ignored on write.
//
// To keep "one active per user" semantics simple at this stage, we deactivate
// all other rows for the same (user_id, app_id, meta_user_id) before inserting.
func (r *MetaTokenRepo) Upsert(ctx context.Context, t *domain.MetaToken) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if t.IsActive {
		_, err = tx.Exec(ctx, `
			UPDATE meta_tokens SET is_active=false, updated_at=now()
			WHERE user_id=$1 AND app_id=$2 AND token_type=$3 AND is_active=true
		`, t.UserID, t.AppID, t.TokenType)
		if err != nil {
			return err
		}
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO meta_tokens
		  (user_id, app_id, meta_user_id, encrypted_token, token_type, scopes,
		   expires_at, last_refresh, is_active)
		VALUES ($1,$2,$3,$4,$5,$6,$7, now(), $8)
		RETURNING id, created_at, updated_at, last_refresh
	`, t.UserID, t.AppID, t.MetaUserID, t.EncryptedToken, t.TokenType, t.Scopes,
		t.ExpiresAt, t.IsActive,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt, &t.LastRefresh)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *MetaTokenRepo) GetActiveByUser(ctx context.Context, userID string) (*domain.MetaToken, error) {
	var t domain.MetaToken
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, app_id, meta_user_id, encrypted_token, token_type, scopes,
		       expires_at, last_refresh, is_active, created_at, updated_at
		FROM meta_tokens
		WHERE user_id=$1 AND is_active=true
		ORDER BY last_refresh DESC
		LIMIT 1
	`, userID).Scan(&t.ID, &t.UserID, &t.AppID, &t.MetaUserID, &t.EncryptedToken,
		&t.TokenType, &t.Scopes, &t.ExpiresAt, &t.LastRefresh, &t.IsActive,
		&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: meta token", domain.ErrNotFound)
		}
		return nil, err
	}
	plain, err := r.cfg.Decrypt(t.EncryptedToken)
	if err != nil {
		return nil, fmt.Errorf("decrypt meta token: %w", err)
	}
	t.PlainToken = plain
	return &t, nil
}

func (r *MetaTokenRepo) ListByUser(ctx context.Context, userID string) ([]*domain.MetaToken, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, app_id, meta_user_id, encrypted_token, token_type, scopes,
		       expires_at, last_refresh, is_active, created_at, updated_at
		FROM meta_tokens
		WHERE user_id=$1
		ORDER BY last_refresh DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.MetaToken
	for rows.Next() {
		var t domain.MetaToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.AppID, &t.MetaUserID, &t.EncryptedToken,
			&t.TokenType, &t.Scopes, &t.ExpiresAt, &t.LastRefresh, &t.IsActive,
			&t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &t)
	}
	return result, rows.Err()
}
