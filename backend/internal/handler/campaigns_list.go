package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/middleware"
)

// CampaignsListHandler exposes a cross-account campaigns list with insights
// aggregated over a chosen window. Used by /campanhas.
type CampaignsListHandler struct {
	db *pgxpool.Pool
}

func NewCampaignsListHandler(db *pgxpool.Pool) *CampaignsListHandler {
	return &CampaignsListHandler{db: db}
}

// List handles GET /api/v1/campanhas?days=N&status=&account=&q=
// Status: ACTIVE | PAUSED | DELETED | "" (all non-deleted)
// Account: act_xxx — filter to a single account
// q: search by name (substring, case-insensitive)
func (h *CampaignsListHandler) List(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	ctx := c.UserContext()

	days := c.QueryInt("days", 7)
	if days < 1 || days > 90 {
		days = 7
	}
	statusFilter := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	accountFilter := strings.TrimSpace(c.Query("account"))
	if accountFilter != "" && !strings.HasPrefix(accountFilter, "act_") {
		accountFilter = "act_" + accountFilter
	}
	q := strings.TrimSpace(c.Query("q"))

	args := []any{userID, days}
	where := []string{
		"c.user_id = $1",
		"c.deleted_at IS NULL",
	}
	if statusFilter != "" && statusFilter != "ALL" {
		args = append(args, statusFilter)
		where = append(where, "c.status = $"+itoa(len(args)))
	}
	if accountFilter != "" {
		args = append(args, accountFilter)
		where = append(where, "c.ad_account_id = $"+itoa(len(args)))
	}
	if q != "" {
		args = append(args, "%"+q+"%")
		where = append(where, "c.name ILIKE $"+itoa(len(args)))
	}
	whereSQL := "WHERE " + strings.Join(where, " AND ")

	rows, err := h.db.Query(ctx, `
		SELECT c.id, c.meta_campaign_id, c.name, c.status, c.objective,
		       COALESCE(c.daily_budget, 0)::float8,
		       COALESCE(c.lifetime_budget, 0)::float8,
		       c.health_status,
		       c.ad_account_id,
		       a.name AS account_name,
		       COALESCE(b.name, '') AS bm_name,
		       c.meta_start_time,
		       c.meta_stop_time,
		       (SELECT MIN(date) FROM campaign_insights WHERE campaign_id = c.id) AS first_insight_date,
		       COALESCE(s.spend, 0)::float8       AS spend,
		       COALESCE(s.impressions, 0)::bigint AS impressions,
		       COALESCE(s.clicks, 0)::bigint      AS clicks,
		       COALESCE(s.leads, 0)::bigint       AS leads,
		       COALESCE(s.avg_freq, 0)::float8    AS avg_freq
		FROM campaigns c
		LEFT JOIN meta_ad_accounts a ON a.user_id = c.user_id AND a.meta_id = c.ad_account_id
		LEFT JOIN business_managers b ON b.id = a.bm_id
		LEFT JOIN LATERAL (
		  SELECT SUM(spend) AS spend, SUM(impressions) AS impressions,
		         SUM(clicks) AS clicks, SUM(leads) AS leads,
		         AVG(frequency) AS avg_freq
		  FROM campaign_insights
		  WHERE campaign_id = c.id AND date > CURRENT_DATE - $2::int
		) s ON true
		`+whereSQL+`
		ORDER BY COALESCE(s.spend, 0) DESC, c.name ASC
		LIMIT 500
	`, args...)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	type row struct {
		ID              string  `json:"id"`
		MetaCampaignID  string  `json:"meta_campaign_id"`
		Name            string  `json:"name"`
		Status          string  `json:"status"`
		Objective       string  `json:"objective"`
		DailyBudget     float64 `json:"daily_budget"`
		LifetimeBudget  float64 `json:"lifetime_budget"`
		HealthStatus    string  `json:"health_status"`
		AccountMetaID   string  `json:"account_meta_id"`
		AccountName     string  `json:"account_name"`
		BMName          string  `json:"bm_name"`
		MetaStartTime   *string `json:"meta_start_time,omitempty"`
		MetaStopTime    *string `json:"meta_stop_time,omitempty"`
		DaysRunning     int     `json:"days_running"`
		Spend           float64 `json:"spend"`
		Impressions     int64   `json:"impressions"`
		Clicks          int64   `json:"clicks"`
		Leads           int64   `json:"leads"`
		CTR             float64 `json:"ctr"`
		CPL             float64 `json:"cpl"`
		AvgFrequency    float64 `json:"avg_frequency"`
	}

	out := make([]row, 0, 64)
	for rows.Next() {
		var r row
		var startT, stopT, firstInsight *time.Time
		if err := rows.Scan(&r.ID, &r.MetaCampaignID, &r.Name, &r.Status, &r.Objective,
			&r.DailyBudget, &r.LifetimeBudget, &r.HealthStatus,
			&r.AccountMetaID, &r.AccountName, &r.BMName,
			&startT, &stopT, &firstInsight,
			&r.Spend, &r.Impressions, &r.Clicks, &r.Leads, &r.AvgFrequency); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if startT != nil {
			s := startT.Format(time.RFC3339)
			r.MetaStartTime = &s
		}
		if stopT != nil {
			s := stopT.Format(time.RFC3339)
			r.MetaStopTime = &s
		}
		switch {
		case startT != nil:
			r.DaysRunning = int(time.Since(*startT).Hours()/24) + 1
		case firstInsight != nil:
			r.DaysRunning = int(time.Since(*firstInsight).Hours()/24) + 1
		}
		if r.DaysRunning < 0 {
			r.DaysRunning = 0
		}
		if r.Impressions > 0 {
			r.CTR = float64(r.Clicks) / float64(r.Impressions)
		}
		if r.Leads > 0 {
			r.CPL = r.Spend / float64(r.Leads)
		}
		out = append(out, r)
	}

	return c.JSON(fiber.Map{"data": out})
}

func itoa(i int) string {
	digits := "0123456789"
	if i == 0 {
		return "0"
	}
	negative := i < 0
	if negative {
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = digits[i%10]
		i /= 10
	}
	if negative {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
