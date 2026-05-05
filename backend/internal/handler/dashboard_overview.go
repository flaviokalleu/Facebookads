package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/middleware"
)

// DashboardOverviewHandler agrega métricas de TODAS as contas do usuário.
// Responde a primeira pergunta da manhã: "como meu negócio inteiro está?".
type DashboardOverviewHandler struct {
	db *pgxpool.Pool
}

func NewDashboardOverviewHandler(db *pgxpool.Pool) *DashboardOverviewHandler {
	return &DashboardOverviewHandler{db: db}
}

// Overview handles GET /api/v1/dashboard/overview?days=N
func (h *DashboardOverviewHandler) Overview(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	ctx := c.UserContext()

	days := c.QueryInt("days", 7)
	if days < 1 || days > 90 {
		days = 7
	}

	type kpi struct {
		Spend         float64  `json:"spend"`
		SpendPrev     float64  `json:"spend_prev"`
		Leads         int64    `json:"leads"`
		LeadsPrev     int64    `json:"leads_prev"`
		AvgCPL        float64  `json:"avg_cpl"`
		AvgCPLPrev    float64  `json:"avg_cpl_prev"`
		Impressions   int64    `json:"impressions"`
		Clicks        int64    `json:"clicks"`
		AccountsTotal int64    `json:"accounts_total"`
		AccountsActive int64   `json:"accounts_active"`
		BMsTotal      int64    `json:"bms_total"`
		ActiveCampaigns int64  `json:"active_campaigns"`
		LowBalanceCount int64  `json:"low_balance_count"`
		BurningNoLeads  int64  `json:"burning_no_leads"`
	}
	var k kpi

	// Aggregate over all accounts in the window.
	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(ci.spend), 0),
		       COALESCE(SUM(ci.impressions), 0),
		       COALESCE(SUM(ci.clicks), 0),
		       COALESCE(SUM(ci.leads), 0)
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id = $1
		  AND c.deleted_at IS NULL
		  AND ci.date > CURRENT_DATE - $2::int
	`, userID, days).Scan(&k.Spend, &k.Impressions, &k.Clicks, &k.Leads); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if k.Leads > 0 {
		k.AvgCPL = k.Spend / float64(k.Leads)
	}

	// Previous equal window for trend deltas.
	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(ci.spend), 0),
		       COALESCE(SUM(ci.leads), 0)
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id = $1
		  AND c.deleted_at IS NULL
		  AND ci.date > CURRENT_DATE - ($2::int * 2)
		  AND ci.date <= CURRENT_DATE - $2::int
	`, userID, days).Scan(&k.SpendPrev, &k.LeadsPrev); err == nil && k.LeadsPrev > 0 {
		k.AvgCPLPrev = k.SpendPrev / float64(k.LeadsPrev)
	}

	// Account / BM counts.
	if err := h.db.QueryRow(ctx, `
		SELECT
		  COUNT(*),
		  COUNT(*) FILTER (WHERE account_status = 1),
		  (SELECT COUNT(*) FROM business_managers WHERE user_id = $1)
		FROM meta_ad_accounts
		WHERE user_id = $1
	`, userID).Scan(&k.AccountsTotal, &k.AccountsActive, &k.BMsTotal); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Active campaigns count.
	if err := h.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM campaigns
		WHERE user_id = $1 AND status = 'ACTIVE' AND deleted_at IS NULL
	`, userID).Scan(&k.ActiveCampaigns); err != nil {
		k.ActiveCampaigns = 0
	}

	// Per-account roll-up (used by Top/Worst lists + alerts).
	rows, err := h.db.Query(ctx, `
		SELECT a.meta_id, a.name, a.balance, COALESCE(b.name,'') AS bm_name,
		       COALESCE(SUM(ci.spend), 0)::float8       AS spend,
		       COALESCE(SUM(ci.leads), 0)::bigint       AS leads,
		       COALESCE(SUM(ci.impressions), 0)::bigint AS impressions,
		       COALESCE(SUM(ci.clicks), 0)::bigint      AS clicks
		FROM meta_ad_accounts a
		LEFT JOIN business_managers b ON b.id = a.bm_id
		LEFT JOIN campaigns c ON c.user_id = a.user_id AND c.ad_account_id = a.meta_id AND c.deleted_at IS NULL
		LEFT JOIN campaign_insights ci ON ci.campaign_id = c.id AND ci.date > CURRENT_DATE - $2::int
		WHERE a.user_id = $1
		GROUP BY a.meta_id, a.name, a.balance, b.name
	`, userID, days)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	all := make([]accRow, 0)
	for rows.Next() {
		var r accRow
		var balanceCents float64
		if err := rows.Scan(&r.MetaID, &r.Name, &balanceCents, &r.BMName,
			&r.Spend, &r.Leads, &r.Impressions, &r.Clicks); err != nil {
			continue
		}
		r.Balance = balanceCents / 100.0
		if r.Leads > 0 {
			r.CPL = r.Spend / float64(r.Leads)
		}
		// Days balance left.
		if r.Spend > 0 {
			daily := r.Spend / float64(days)
			d := r.Balance / daily
			r.DaysLeft = &d
		}
		// Status flags.
		switch {
		case r.Spend > 0 && r.Leads == 0:
			r.Status = "burning_no_leads"
		case r.DaysLeft != nil && *r.DaysLeft < 3:
			r.Status = "low_balance"
		case r.Leads > 0 && r.AvgIsGood(k.AvgCPL):
			r.Status = "winner"
		case r.Spend == 0 && r.Leads == 0:
			r.Status = "idle"
		default:
			r.Status = "normal"
		}
		all = append(all, r)
	}

	// Top 5 by spend (busiest accounts).
	bySpend := append([]accRow(nil), all...)
	sortDesc(bySpend, func(x accRow) float64 { return x.Spend })

	// Best 5 by CPL (need at least 5 leads in window for statistical relevance).
	byCPL := make([]accRow, 0, len(all))
	for _, r := range all {
		if r.Leads >= 5 {
			byCPL = append(byCPL, r)
		}
	}
	bestByCPL := append([]accRow(nil), byCPL...)
	sortAsc(bestByCPL, func(x accRow) float64 { return x.CPL })

	// Worst 5 by CPL (same threshold).
	worstByCPL := append([]accRow(nil), byCPL...)
	sortDesc(worstByCPL, func(x accRow) float64 { return x.CPL })

	// Alerts: low balance + burning no leads.
	lowBalance := make([]accRow, 0)
	burningNoLeads := make([]accRow, 0)
	for _, r := range all {
		if r.Status == "low_balance" {
			lowBalance = append(lowBalance, r)
			k.LowBalanceCount++
		}
		if r.Status == "burning_no_leads" && r.Spend >= 5 {
			burningNoLeads = append(burningNoLeads, r)
			k.BurningNoLeads++
		}
	}
	sortDesc(lowBalance, func(x accRow) float64 {
		if x.DaysLeft != nil { return -*x.DaysLeft }
		return 0
	})
	sortDesc(burningNoLeads, func(x accRow) float64 { return x.Spend })

	// Daily aggregate series for the chart.
	chartDays := days
	if chartDays < 7 { chartDays = 7 }
	chartRows, err := h.db.Query(ctx, `
		WITH dates AS (
		  SELECT generate_series(CURRENT_DATE - ($2::int - 1), CURRENT_DATE, '1 day')::date AS d
		)
		SELECT d.d,
		       COALESCE(SUM(ci.spend), 0),
		       COALESCE(SUM(ci.leads), 0),
		       COALESCE(SUM(ci.impressions), 0),
		       COALESCE(SUM(ci.clicks), 0)
		FROM dates d
		LEFT JOIN campaigns c
		       ON c.user_id = $1 AND c.deleted_at IS NULL
		LEFT JOIN campaign_insights ci
		       ON ci.campaign_id = c.id AND ci.date = d.d
		GROUP BY d.d ORDER BY d.d
	`, userID, chartDays)
	type point struct {
		Date  string  `json:"date"`
		Spend float64 `json:"spend"`
		Leads int64   `json:"leads"`
		Imps  int64   `json:"impressions"`
		Clk   int64   `json:"clicks"`
	}
	chart := make([]point, 0)
	if err == nil {
		defer chartRows.Close()
		for chartRows.Next() {
			var p point
			var d time.Time
			if err := chartRows.Scan(&d, &p.Spend, &p.Leads, &p.Imps, &p.Clk); err == nil {
				p.Date = d.Format("2006-01-02")
				chart = append(chart, p)
			}
		}
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"kpis":              k,
		"top_by_spend":      first(bySpend, 5),
		"best_by_cpl":       first(bestByCPL, 5),
		"worst_by_cpl":      first(worstByCPL, 5),
		"low_balance":       first(lowBalance, 10),
		"burning_no_leads":  first(burningNoLeads, 10),
		"daily":             chart,
	}})
}

// AvgIsGood is true when this account's CPL is at most 70% of the global avg
// AND it has meaningful volume — used to flag "winners" in the dashboard.
func (r accRow) AvgIsGood(globalCPL float64) bool {
	if globalCPL <= 0 || r.CPL <= 0 || r.Leads < 5 {
		return false
	}
	return r.CPL <= 0.7*globalCPL
}

// accRow lifted to package scope so the receiver above works.
type accRow struct {
	MetaID      string   `json:"meta_id"`
	Name        string   `json:"name"`
	BMName      string   `json:"bm_name"`
	Balance     float64  `json:"balance"`
	Spend       float64  `json:"spend"`
	Leads       int64    `json:"leads"`
	CPL         float64  `json:"cpl"`
	Impressions int64    `json:"impressions"`
	Clicks      int64    `json:"clicks"`
	DaysLeft    *float64 `json:"days_balance_left,omitempty"`
	Status      string   `json:"status"`
}

func sortDesc(rs []accRow, key func(accRow) float64) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && key(rs[j]) > key(rs[j-1]); j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}
}
func sortAsc(rs []accRow, key func(accRow) float64) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && key(rs[j]) < key(rs[j-1]); j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}
}
func first(rs []accRow, n int) []accRow {
	if len(rs) > n {
		return rs[:n]
	}
	return rs
}
