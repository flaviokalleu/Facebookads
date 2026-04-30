package postgres

import (
	"context"
	"time"

	"github.com/facebookads/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InsightRepo struct {
	db *pgxpool.Pool
}

func NewInsightRepo(db *pgxpool.Pool) *InsightRepo {
	return &InsightRepo{db: db}
}

func (r *InsightRepo) Upsert(ctx context.Context, i *domain.CampaignInsight) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO campaign_insights
		  (campaign_id, date, spend, impressions, clicks, ctr, cpc, cpm, reach, frequency, leads, purchases, roas, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,now())
		ON CONFLICT (campaign_id, date) DO UPDATE SET
		  spend       = EXCLUDED.spend,
		  impressions = EXCLUDED.impressions,
		  clicks      = EXCLUDED.clicks,
		  ctr         = EXCLUDED.ctr,
		  cpc         = EXCLUDED.cpc,
		  cpm         = EXCLUDED.cpm,
		  reach       = EXCLUDED.reach,
		  frequency   = EXCLUDED.frequency,
		  leads       = EXCLUDED.leads,
		  purchases   = EXCLUDED.purchases,
		  roas        = EXCLUDED.roas,
		  updated_at  = now()
	`,
		i.CampaignID, i.Date, i.Spend, i.Impressions, i.Clicks,
		i.CTR, i.CPC, i.CPM, i.Reach, i.Frequency,
		i.Leads, i.Purchases, i.ROAS,
	)
	return err
}

func (r *InsightRepo) ListByCampaign(ctx context.Context, campaignID string, from, to time.Time) ([]*domain.CampaignInsight, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, campaign_id, date, spend, impressions, clicks, ctr, cpc, cpm,
		       reach, frequency, leads, purchases, roas, created_at, updated_at
		FROM campaign_insights
		WHERE campaign_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date
	`, campaignID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.CampaignInsight
	for rows.Next() {
		var i domain.CampaignInsight
		if err := rows.Scan(
			&i.ID, &i.CampaignID, &i.Date,
			&i.Spend, &i.Impressions, &i.Clicks,
			&i.CTR, &i.CPC, &i.CPM,
			&i.Reach, &i.Frequency,
			&i.Leads, &i.Purchases, &i.ROAS,
			&i.CreatedAt, &i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, &i)
	}
	return result, rows.Err()
}

func (r *InsightRepo) GetAccountAverages(ctx context.Context, userID string, days int) (avgCTR, avgCPC float64, err error) {
	row := r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(ci.ctr), 0), COALESCE(AVG(ci.cpc), 0)
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id = $1
		  AND ci.date >= CURRENT_DATE - $2::int
		  AND c.deleted_at IS NULL
	`, userID, days)
	err = row.Scan(&avgCTR, &avgCPC)
	return
}
