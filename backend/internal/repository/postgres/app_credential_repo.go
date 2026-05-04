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

type AppCredentialRepo struct {
	db  *pgxpool.Pool
	cfg *config.Service
}

func NewAppCredentialRepo(db *pgxpool.Pool, cfg *config.Service) *AppCredentialRepo {
	return &AppCredentialRepo{db: db, cfg: cfg}
}

// Upsert encrypts AppSecret on the way in. Caller may pass either a plain
// secret in EncryptedAppSecret (we'll detect by trying to encrypt fresh — see
// SyncMetaAccount) or an already-encrypted blob. To keep this simple, the
// caller is responsible for encrypting BEFORE calling Upsert.
func (r *AppCredentialRepo) Upsert(ctx context.Context, c *domain.AppCredential) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO app_credentials (user_id, app_id, encrypted_app_secret)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, app_id) DO UPDATE SET
		  encrypted_app_secret = EXCLUDED.encrypted_app_secret,
		  updated_at           = now()
		RETURNING id, created_at, updated_at
	`, c.UserID, c.AppID, c.EncryptedAppSecret).
		Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *AppCredentialRepo) GetByUserAndAppID(ctx context.Context, userID, appID string) (*domain.AppCredential, error) {
	var c domain.AppCredential
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, app_id, encrypted_app_secret, created_at, updated_at
		FROM app_credentials WHERE user_id=$1 AND app_id=$2
	`, userID, appID).Scan(&c.ID, &c.UserID, &c.AppID, &c.EncryptedAppSecret, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: app credential", domain.ErrNotFound)
		}
		return nil, err
	}
	return &c, nil
}

func (r *AppCredentialRepo) ListByUser(ctx context.Context, userID string) ([]*domain.AppCredential, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, app_id, encrypted_app_secret, created_at, updated_at
		FROM app_credentials WHERE user_id=$1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.AppCredential
	for rows.Next() {
		var c domain.AppCredential
		if err := rows.Scan(&c.ID, &c.UserID, &c.AppID, &c.EncryptedAppSecret, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &c)
	}
	return result, rows.Err()
}

// DecryptAppSecret returns the plain-text secret for a credential row.
func (r *AppCredentialRepo) DecryptAppSecret(c *domain.AppCredential) (string, error) {
	return r.cfg.Decrypt(c.EncryptedAppSecret)
}
