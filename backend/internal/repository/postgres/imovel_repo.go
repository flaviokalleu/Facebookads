package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImovelRepo struct {
	db *pgxpool.Pool
}

func NewImovelRepo(db *pgxpool.Pool) *ImovelRepo {
	return &ImovelRepo{db: db}
}

func (r *ImovelRepo) Create(ctx context.Context, im *domain.Imovel) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO imoveis
		  (user_id, nome, segmento, cidade, bairro, preco_min, preco_max,
		   quartos, area_m2, tipologia, diferenciais, fotos,
		   whatsapp_destino, link_landing, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, created_at, updated_at
	`,
		im.UserID, im.Nome, im.Segmento, im.Cidade, im.Bairro,
		im.PrecoMin, im.PrecoMax, im.Quartos, im.AreaM2, im.Tipologia,
		im.Diferenciais, im.Fotos,
		im.WhatsAppDestino, im.LinkLanding, im.Status,
	).Scan(&im.ID, &im.CreatedAt, &im.UpdatedAt)
}

func (r *ImovelRepo) Update(ctx context.Context, im *domain.Imovel) error {
	_, err := r.db.Exec(ctx, `
		UPDATE imoveis SET
		  nome             = $2,
		  segmento         = $3,
		  cidade           = $4,
		  bairro           = $5,
		  preco_min        = $6,
		  preco_max        = $7,
		  quartos          = $8,
		  area_m2          = $9,
		  tipologia        = $10,
		  diferenciais     = $11,
		  fotos            = $12,
		  whatsapp_destino = $13,
		  link_landing     = $14,
		  status           = $15,
		  updated_at       = now()
		WHERE id=$1 AND deleted_at IS NULL
	`,
		im.ID, im.Nome, im.Segmento, im.Cidade, im.Bairro,
		im.PrecoMin, im.PrecoMax, im.Quartos, im.AreaM2, im.Tipologia,
		im.Diferenciais, im.Fotos,
		im.WhatsAppDestino, im.LinkLanding, im.Status,
	)
	return err
}

func (r *ImovelRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE imoveis SET deleted_at = now() WHERE id=$1`, id)
	return err
}

func (r *ImovelRepo) GetByID(ctx context.Context, id string) (*domain.Imovel, error) {
	var im domain.Imovel
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, nome, segmento,
		       COALESCE(cidade,''), COALESCE(bairro,''),
		       preco_min, preco_max, quartos, area_m2,
		       COALESCE(tipologia,''), diferenciais, fotos,
		       COALESCE(whatsapp_destino,''), COALESCE(link_landing,''),
		       status, created_at, updated_at, deleted_at
		FROM imoveis WHERE id=$1 AND deleted_at IS NULL
	`, id).Scan(&im.ID, &im.UserID, &im.Nome, &im.Segmento,
		&im.Cidade, &im.Bairro,
		&im.PrecoMin, &im.PrecoMax, &im.Quartos, &im.AreaM2,
		&im.Tipologia, &im.Diferenciais, &im.Fotos,
		&im.WhatsAppDestino, &im.LinkLanding,
		&im.Status, &im.CreatedAt, &im.UpdatedAt, &im.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: imovel", domain.ErrNotFound)
		}
		return nil, err
	}
	return &im, nil
}

func (r *ImovelRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Imovel, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, nome, segmento,
		       COALESCE(cidade,''), COALESCE(bairro,''),
		       preco_min, preco_max, quartos, area_m2,
		       COALESCE(tipologia,''), diferenciais, fotos,
		       COALESCE(whatsapp_destino,''), COALESCE(link_landing,''),
		       status, created_at, updated_at, deleted_at
		FROM imoveis WHERE user_id=$1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.Imovel
	for rows.Next() {
		var im domain.Imovel
		if err := rows.Scan(&im.ID, &im.UserID, &im.Nome, &im.Segmento,
			&im.Cidade, &im.Bairro,
			&im.PrecoMin, &im.PrecoMax, &im.Quartos, &im.AreaM2,
			&im.Tipologia, &im.Diferenciais, &im.Fotos,
			&im.WhatsAppDestino, &im.LinkLanding,
			&im.Status, &im.CreatedAt, &im.UpdatedAt, &im.DeletedAt); err != nil {
			return nil, err
		}
		result = append(result, &im)
	}
	return result, rows.Err()
}
