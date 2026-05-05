package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/ai/prompts"
	"github.com/facebookads/backend/internal/ai/providers"
	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
)

// breakdownCacheEntry is one row in the in-memory cache used by Breakdowns.
type breakdownCacheEntry struct {
	rows      []metaads.BreakdownRow
	expiresAt time.Time
}

type AccountDetailHandler struct {
	db              *pgxpool.Pool
	cfg             *config.Service
	metaTokens      repository.MetaTokenRepository
	metaClient      metaads.Client
	breakdownsCache sync.Map // key=string -> *breakdownCacheEntry
}

func NewAccountDetailHandler(
	db *pgxpool.Pool,
	cfg *config.Service,
	metaTokens repository.MetaTokenRepository,
	metaClient metaads.Client,
) *AccountDetailHandler {
	return &AccountDetailHandler{
		db:         db,
		cfg:        cfg,
		metaTokens: metaTokens,
		metaClient: metaClient,
	}
}

// Meta returns balance/amount_spent/spend_cap as integer cents (smallest currency unit).
// Convert to BRL units before exposing to the frontend.
func centsToReal(v float64) float64 { return v / 100.0 }

// normalizeAccountID accepts both `act_123` and `123`, returns `act_123`.
func normalizeAccountID(in string) string {
	in = strings.TrimSpace(in)
	if !strings.HasPrefix(in, "act_") {
		return "act_" + in
	}
	return in
}

// Get handles GET /api/v1/contas/:account_id?days=N
// Returns: account record + KPIs aggregated over the chosen window (default 7d).
func (h *AccountDetailHandler) Get(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	accountID := normalizeAccountID(c.Params("account_id"))
	ctx := c.UserContext()

	days := c.QueryInt("days", 7)
	if days < 1 || days > 90 {
		days = 7
	}

	var (
		metaID, name, currency, accessKind string
		accountStatus                      int
		balance, amountSpent, spendCap     float64
		bmName                             *string
	)
	err := h.db.QueryRow(ctx, `
		SELECT a.meta_id, a.name, a.currency, a.access_kind,
		       a.account_status, a.balance, a.amount_spent, a.spend_cap,
		       b.name
		FROM meta_ad_accounts a
		LEFT JOIN business_managers b ON b.id = a.bm_id
		WHERE a.user_id = $1 AND a.meta_id = $2
	`, userID, accountID).Scan(&metaID, &name, &currency, &accessKind,
		&accountStatus, &balance, &amountSpent, &spendCap, &bmName)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "account not found")
	}

	type kpi struct {
		Spend             float64  `json:"spend_7d"`
		SpendPrev7d       float64  `json:"spend_prev_7d"`
		Impressions       int64    `json:"impressions_7d"`
		Clicks            int64    `json:"clicks_7d"`
		Leads             int64    `json:"leads_7d"`
		LeadsPrev7d       int64    `json:"leads_prev_7d"`
		AvgCPL            float64  `json:"avg_cpl_7d"`
		AvgCPLPrev7d      float64  `json:"avg_cpl_prev_7d"`
		AvgCTR            float64  `json:"avg_ctr_7d"`
		ActiveCount       int64    `json:"active_campaigns"`
		PausedCount       int64    `json:"paused_campaigns"`
		DaysBalanceLeft   *float64 `json:"days_balance_left,omitempty"`
		BestDay           *string  `json:"best_day,omitempty"`
		BestDayLeads      int64    `json:"best_day_leads"`
		BestDayCPL        float64  `json:"best_day_cpl"`
	}
	var k kpi

	// Window includes today: N=1 → só hoje; N=7 → hoje + 6 anteriores.
	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(ci.spend), 0),
		       COALESCE(SUM(ci.impressions), 0),
		       COALESCE(SUM(ci.clicks), 0),
		       COALESCE(SUM(ci.leads), 0)
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id = $1 AND c.ad_account_id = $2
		  AND ci.date > CURRENT_DATE - $3::int
	`, userID, accountID, days).Scan(&k.Spend, &k.Impressions, &k.Clicks, &k.Leads); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if k.Leads > 0 {
		k.AvgCPL = k.Spend / float64(k.Leads)
	}
	if k.Impressions > 0 {
		k.AvgCTR = float64(k.Clicks) / float64(k.Impressions)
	}

	// Previous window of equal length for trend deltas.
	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(ci.spend), 0),
		       COALESCE(SUM(ci.leads), 0)
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id = $1 AND c.ad_account_id = $2
		  AND ci.date > CURRENT_DATE - ($3::int * 2)
		  AND ci.date <= CURRENT_DATE - $3::int
	`, userID, accountID, days).Scan(&k.SpendPrev7d, &k.LeadsPrev7d); err == nil && k.LeadsPrev7d > 0 {
		k.AvgCPLPrev7d = k.SpendPrev7d / float64(k.LeadsPrev7d)
	}

	// Active/paused counts.
	if err := h.db.QueryRow(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE status = 'ACTIVE'),
		  COUNT(*) FILTER (WHERE status = 'PAUSED')
		FROM campaigns
		WHERE user_id = $1 AND ad_account_id = $2 AND deleted_at IS NULL
	`, userID, accountID).Scan(&k.ActiveCount, &k.PausedCount); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// "Saldo dura X dias" — taxa diária = spend_window / days.
	if k.Spend > 0 {
		dailyBurn := k.Spend / float64(days)
		if dailyBurn > 0 {
			d := centsToReal(balance) / dailyBurn
			k.DaysBalanceLeft = &d
		}
	}

	// Best day in last 14d (max leads).
	var bestDate *time.Time
	if err := h.db.QueryRow(ctx, `
		SELECT ci.date,
		       SUM(ci.leads),
		       CASE WHEN SUM(ci.leads) > 0 THEN SUM(ci.spend)/SUM(ci.leads) ELSE 0 END
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id = $1 AND c.ad_account_id = $2
		  AND ci.date > CURRENT_DATE - 14
		GROUP BY ci.date
		HAVING SUM(ci.leads) > 0
		ORDER BY SUM(ci.leads) DESC, SUM(ci.spend)/NULLIF(SUM(ci.leads),0) ASC NULLS LAST
		LIMIT 1
	`, userID, accountID).Scan(&bestDate, &k.BestDayLeads, &k.BestDayCPL); err == nil && bestDate != nil {
		s := bestDate.Format("2006-01-02")
		k.BestDay = &s
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"account": fiber.Map{
			"meta_id":        metaID,
			"name":           name,
			"currency":       currency,
			"access_kind":    accessKind,
			"account_status": accountStatus,
			"balance":        centsToReal(balance),
			"amount_spent":   centsToReal(amountSpent),
			"spend_cap":      centsToReal(spendCap),
			"bm_name":        bmName,
		},
		"kpis": k,
	}})
}

// DailyInsights handles GET /api/v1/contas/:account_id/insights/daily?days=14
// Returns daily aggregates for sparklines.
func (h *AccountDetailHandler) DailyInsights(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	accountID := normalizeAccountID(c.Params("account_id"))
	ctx := c.UserContext()

	days := c.QueryInt("days", 14)
	if days < 1 || days > 90 {
		days = 14
	}

	rows, err := h.db.Query(ctx, `
		WITH dates AS (
		  SELECT generate_series(CURRENT_DATE - ($3::int - 1), CURRENT_DATE, '1 day')::date AS d
		)
		SELECT d.d,
		       COALESCE(SUM(ci.spend), 0),
		       COALESCE(SUM(ci.impressions), 0),
		       COALESCE(SUM(ci.clicks), 0),
		       COALESCE(SUM(ci.leads), 0)
		FROM dates d
		LEFT JOIN campaigns c
		       ON c.user_id = $1 AND c.ad_account_id = $2 AND c.deleted_at IS NULL
		LEFT JOIN campaign_insights ci
		       ON ci.campaign_id = c.id AND ci.date = d.d
		GROUP BY d.d
		ORDER BY d.d
	`, userID, accountID, days)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	type point struct {
		Date        string  `json:"date"`
		Spend       float64 `json:"spend"`
		Impressions int64   `json:"impressions"`
		Clicks      int64   `json:"clicks"`
		Leads       int64   `json:"leads"`
		CPL         float64 `json:"cpl"`
	}
	out := make([]point, 0, days)
	for rows.Next() {
		var p point
		var d time.Time
		if err := rows.Scan(&d, &p.Spend, &p.Impressions, &p.Clicks, &p.Leads); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		p.Date = d.Format("2006-01-02")
		if p.Leads > 0 {
			p.CPL = p.Spend / float64(p.Leads)
		}
		out = append(out, p)
	}
	return c.JSON(fiber.Map{"data": out})
}

// GetAnalysis returns the latest cached AI analysis for the account.
// GET /api/v1/contas/:account_id/analysis
func (h *AccountDetailHandler) GetAnalysis(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	accountID := normalizeAccountID(c.Params("account_id"))
	ctx := c.UserContext()

	var (
		summary, model string
		highlights     *string
		createdAt      time.Time
	)
	err := h.db.QueryRow(ctx, `
		SELECT summary, COALESCE(model_used,''), highlights::text, created_at
		FROM account_analyses
		WHERE user_id=$1 AND account_meta_id=$2
		ORDER BY created_at DESC
		LIMIT 1
	`, userID, accountID).Scan(&summary, &model, &highlights, &createdAt)
	if err != nil {
		return c.JSON(fiber.Map{"data": nil})
	}

	resp := fiber.Map{
		"summary":    summary,
		"model_used": model,
		"created_at": createdAt,
	}
	if highlights != nil && *highlights != "" {
		var hl any
		_ = json.Unmarshal([]byte(*highlights), &hl)
		resp["highlights"] = hl
	}
	return c.JSON(fiber.Map{"data": resp})
}

// Analyze runs DeepSeek on the account's last 14 days and saves the analysis.
// POST /api/v1/contas/:account_id/analyze
func (h *AccountDetailHandler) Analyze(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	accountID := normalizeAccountID(c.Params("account_id"))
	ctx := c.UserContext()

	apiKey := h.cfg.GetSecret("ai.deepseek.api_key")
	if apiKey == "" {
		return fiber.NewError(fiber.StatusBadRequest, "deepseek api key not configured — go to /ajustes/api-keys")
	}

	snapshot, err := h.buildAccountSnapshot(ctx, userID, accountID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if snapshot == "" {
		return fiber.NewError(fiber.StatusBadRequest, "sem dados suficientes — espere a próxima sincronização")
	}

	provider := providers.NewDeepSeek(apiKey, "deepseek-chat")

	start := time.Now()
	resp, err := provider.Complete(ctx, ai.CompletionRequest{
		SystemPrompt: prompts.AnalyzeSystemPrompt,
		UserPrompt:   snapshot,
		MaxTokens:    1400,
		Temperature:  0.3,
		JSONMode:     true,
	})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		slog.Error("account analyze: deepseek failed", "account", accountID, "err", err)
		return fiber.NewError(fiber.StatusBadGateway, "deepseek error: "+err.Error())
	}

	var parsed struct {
		Summary     string            `json:"summary"`
		Highlights  []json.RawMessage `json:"highlights"`
		NextActions []json.RawMessage `json:"next_actions"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &parsed); err != nil {
		// fallback: store raw text in summary
		parsed.Summary = resp.Content
	}

	hlPayload, _ := json.Marshal(map[string]any{
		"highlights":   parsed.Highlights,
		"next_actions": parsed.NextActions,
	})

	if _, err := h.db.Exec(ctx, `
		INSERT INTO account_analyses
		  (user_id, account_meta_id, summary, highlights, model_used,
		   input_tokens, output_tokens, cost_usd, latency_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, userID, accountID, parsed.Summary, string(hlPayload), resp.ModelUsed,
		resp.InputTokens, resp.OutputTokens, resp.CostUSD, latency); err != nil {
		slog.Warn("account analyze: persist failed", "err", err)
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"summary":      parsed.Summary,
		"highlights":   parsed.Highlights,
		"next_actions": parsed.NextActions,
		"model_used":   resp.ModelUsed,
		"latency_ms":   latency,
		"cost_usd":     resp.CostUSD,
	}})
}

// buildAccountSnapshot creates a textual snapshot for DeepSeek.
func (h *AccountDetailHandler) buildAccountSnapshot(ctx context.Context, userID, accountID string) (string, error) {
	var (
		name, currency string
		balance, spent float64
	)
	if err := h.db.QueryRow(ctx, `
		SELECT name, currency, balance, amount_spent
		FROM meta_ad_accounts
		WHERE user_id=$1 AND meta_id=$2
	`, userID, accountID).Scan(&name, &currency, &balance, &spent); err != nil {
		return "", fmt.Errorf("conta não encontrada")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Conta %s (%s) — saldo R$ %.2f, gasto acumulado R$ %.2f\n\n",
		name, accountID, centsToReal(balance), centsToReal(spent))

	// Daily roll-up last 14 days.
	rows, err := h.db.Query(ctx, `
		SELECT ci.date,
		       COALESCE(SUM(ci.spend),0),
		       COALESCE(SUM(ci.impressions),0),
		       COALESCE(SUM(ci.clicks),0),
		       COALESCE(SUM(ci.leads),0),
		       COALESCE(AVG(ci.frequency),0)
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id=$1 AND c.ad_account_id=$2
		  AND ci.date >= CURRENT_DATE - INTERVAL '14 days'
		GROUP BY ci.date ORDER BY ci.date
	`, userID, accountID)
	if err == nil {
		defer rows.Close()
		fmt.Fprintln(&b, "Últimos 14 dias (por dia):")
		for rows.Next() {
			var d time.Time
			var sp float64
			var imp, clk, lds int64
			var freq float64
			if err := rows.Scan(&d, &sp, &imp, &clk, &lds, &freq); err == nil {
				cpl := 0.0
				if lds > 0 {
					cpl = sp / float64(lds)
				}
				ctr := 0.0
				if imp > 0 {
					ctr = float64(clk) / float64(imp) * 100
				}
				fmt.Fprintf(&b, "  %s — gasto R$ %.2f, contatos %d, custo/contato R$ %.2f, CTR %.2f%%, freq %.2f\n",
					d.Format("02/01"), sp, lds, cpl, ctr, freq)
			}
		}
	}

	// Per-campaign 7d.
	crows, err := h.db.Query(ctx, `
		SELECT c.name, c.status, c.objective,
		       COALESCE(SUM(ci.spend),0),
		       COALESCE(SUM(ci.leads),0),
		       COALESCE(AVG(ci.frequency),0),
		       COALESCE(SUM(ci.clicks),0),
		       COALESCE(SUM(ci.impressions),0)
		FROM campaigns c
		LEFT JOIN campaign_insights ci ON ci.campaign_id = c.id
		   AND ci.date >= CURRENT_DATE - INTERVAL '7 days'
		WHERE c.user_id=$1 AND c.ad_account_id=$2 AND c.deleted_at IS NULL
		GROUP BY c.id ORDER BY SUM(ci.spend) DESC NULLS LAST
	`, userID, accountID)
	if err == nil {
		defer crows.Close()
		fmt.Fprintln(&b, "\nCampanhas (últimos 7 dias):")
		any := false
		for crows.Next() {
			var cname, cstatus, cobj string
			var sp float64
			var lds int64
			var freq float64
			var clk, imp int64
			if err := crows.Scan(&cname, &cstatus, &cobj, &sp, &lds, &freq, &clk, &imp); err == nil {
				cpl := 0.0
				if lds > 0 {
					cpl = sp / float64(lds)
				}
				ctr := 0.0
				if imp > 0 {
					ctr = float64(clk) / float64(imp) * 100
				}
				fmt.Fprintf(&b, "  - \"%s\" [%s, obj=%s] gasto R$ %.2f, contatos %d, custo/contato R$ %.2f, CTR %.2f%%, freq %.2f\n",
					cname, cstatus, cobj, sp, lds, cpl, ctr, freq)
				any = true
			}
		}
		if !any {
			fmt.Fprintln(&b, "  (nenhuma campanha ativa com dados nos últimos 7 dias)")
		}
	}

	return b.String(), nil
}

// ListCampaigns handles GET /api/v1/contas/:account_id/campanhas?days=N
// Returns campaigns for the account with insights aggregated over the chosen window.
func (h *AccountDetailHandler) ListCampaigns(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	accountID := normalizeAccountID(c.Params("account_id"))
	ctx := c.UserContext()

	days := c.QueryInt("days", 7)
	if days < 1 || days > 90 {
		days = 7
	}

	rows, err := h.db.Query(ctx, `
		SELECT c.id, c.meta_campaign_id, c.name, c.status, c.objective,
		       COALESCE(c.daily_budget, 0), COALESCE(c.lifetime_budget, 0),
		       c.health_status,
		       c.meta_created_time, c.meta_start_time, c.meta_stop_time,
		       COALESCE(SUM(ci.spend), 0)        AS spend_window,
		       COALESCE(SUM(ci.impressions), 0)  AS impressions_window,
		       COALESCE(SUM(ci.clicks), 0)       AS clicks_window,
		       COALESCE(SUM(ci.leads), 0)        AS leads_window,
		       COALESCE(AVG(ci.frequency), 0)    AS avg_freq_window,
		       MAX(ci.date)                      AS last_insight_date,
		       (SELECT MIN(date) FROM campaign_insights WHERE campaign_id = c.id) AS first_insight_date
		FROM campaigns c
		LEFT JOIN campaign_insights ci ON ci.campaign_id = c.id
		   AND ci.date > CURRENT_DATE - $3::int
		WHERE c.user_id = $1
		  AND c.ad_account_id = $2
		  AND c.deleted_at IS NULL
		GROUP BY c.id
		ORDER BY spend_window DESC, c.name
	`, userID, accountID, days)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	type row struct {
		ID              string  `json:"id"`
		MetaID          string  `json:"meta_campaign_id"`
		Name            string  `json:"name"`
		Status          string  `json:"status"`
		Objective       string  `json:"objective"`
		DailyBudget     float64 `json:"daily_budget"`
		LifetimeBudget  float64 `json:"lifetime_budget"`
		HealthStatus    string  `json:"health_status"`
		Spend7d         float64 `json:"spend_7d"`
		Impressions7d   int64   `json:"impressions_7d"`
		Clicks7d        int64   `json:"clicks_7d"`
		Leads7d         int64   `json:"leads_7d"`
		CTR7d           float64 `json:"ctr_7d"`
		CPL7d           float64 `json:"cpl_7d"`
		AvgFrequency7d   float64 `json:"avg_frequency_7d"`
		LastInsightDate  *string `json:"last_insight_date,omitempty"`
		FirstInsightDate *string `json:"first_insight_date,omitempty"`
		MetaCreatedTime  *string `json:"meta_created_time,omitempty"`
		MetaStartTime    *string `json:"meta_start_time,omitempty"`
		MetaStopTime     *string `json:"meta_stop_time,omitempty"`
		DaysRunning      int     `json:"days_running"`
	}

	var out []row
	for rows.Next() {
		var r row
		var lastDate, firstDate *time.Time
		var metaCreated, metaStart, metaStop *time.Time
		if err := rows.Scan(&r.ID, &r.MetaID, &r.Name, &r.Status, &r.Objective,
			&r.DailyBudget, &r.LifetimeBudget, &r.HealthStatus,
			&metaCreated, &metaStart, &metaStop,
			&r.Spend7d, &r.Impressions7d, &r.Clicks7d, &r.Leads7d,
			&r.AvgFrequency7d, &lastDate, &firstDate); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if lastDate != nil {
			s := lastDate.Format("2006-01-02")
			r.LastInsightDate = &s
		}
		if firstDate != nil {
			s := firstDate.Format("2006-01-02")
			r.FirstInsightDate = &s
		}
		if metaCreated != nil {
			s := metaCreated.Format(time.RFC3339)
			r.MetaCreatedTime = &s
		}
		if metaStart != nil {
			s := metaStart.Format(time.RFC3339)
			r.MetaStartTime = &s
		}
		if metaStop != nil {
			s := metaStop.Format(time.RFC3339)
			r.MetaStopTime = &s
		}

		// Days running: prefer Meta start_time → meta created_time → first insight date.
		switch {
		case metaStart != nil:
			r.DaysRunning = int(time.Since(*metaStart).Hours()/24) + 1
		case metaCreated != nil:
			r.DaysRunning = int(time.Since(*metaCreated).Hours()/24) + 1
		case firstDate != nil:
			r.DaysRunning = int(time.Since(*firstDate).Hours()/24) + 1
		}
		if r.DaysRunning < 0 {
			r.DaysRunning = 0
		}
		if r.Impressions7d > 0 {
			r.CTR7d = float64(r.Clicks7d) / float64(r.Impressions7d)
		}
		if r.Leads7d > 0 {
			r.CPL7d = r.Spend7d / float64(r.Leads7d)
		}
		out = append(out, r)
	}
	if out == nil {
		out = []row{}
	}
	return c.JSON(fiber.Map{"data": out})
}

// breakdownTTL is the lifetime of one entry in the in-memory breakdowns cache.
const breakdownTTL = 5 * time.Minute

// breakdownAliases maps the friendly query-param value to the Meta `breakdowns`
// API value. The handler accepts either form; the friendly one is what the
// frontend uses ("region", "age_gender", "hour", "placement", "device").
var breakdownAliases = map[string]string{
	"region":             "region",
	"age_gender":         "age,gender",
	"hour":               "hourly_stats_aggregated_by_advertiser_time_zone",
	"placement":          "publisher_platform",
	"device":             "impression_device",
	"platform_position":  "platform_position",
	// raw forms still work for callers that already use the Meta names
	"age,gender":                                      "age,gender",
	"hourly_stats_aggregated_by_advertiser_time_zone": "hourly_stats_aggregated_by_advertiser_time_zone",
	"publisher_platform":                              "publisher_platform",
	"impression_device":                               "impression_device",
}

// daysToPreset maps a numeric `days` query param to a Meta date_preset string.
// Anything not in this set falls back to last_7d.
func daysToPreset(days int) string {
	switch days {
	case 1:
		return "yesterday"
	case 7:
		return "last_7d"
	case 14:
		return "last_14d"
	case 30:
		return "last_30d"
	default:
		return "last_7d"
	}
}

// Breakdowns handles GET /api/v1/contas/:account_id/breakdowns?dim=...&days=N
//
// dim accepts friendly aliases (region|age_gender|hour|placement|device) or the
// raw Meta breakdown name. days is mapped to a date_preset; default 7d.
// Results are cached in-memory per (user, account, dim, days) for 5 minutes to
// keep the per-account detail page snappy without hammering the Meta API.
func (h *AccountDetailHandler) Breakdowns(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	accountID := normalizeAccountID(c.Params("account_id"))
	ctx := c.UserContext()

	rawDim := strings.TrimSpace(c.Query("dim"))
	if rawDim == "" {
		return fiber.NewError(fiber.StatusBadRequest, "informe o parâmetro 'dim' (ex.: region, age_gender, hour, placement, device)")
	}
	metaDim, ok := breakdownAliases[rawDim]
	if !ok {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("dim inválido: %q — use region, age_gender, hour, placement ou device", rawDim))
	}

	days := c.QueryInt("days", 7)
	preset := daysToPreset(days)

	cacheKey := fmt.Sprintf("%s|%s|%s|%d", userID, accountID, metaDim, days)
	if v, hit := h.breakdownsCache.Load(cacheKey); hit {
		if entry, ok := v.(*breakdownCacheEntry); ok && time.Now().Before(entry.expiresAt) {
			return c.JSON(fiber.Map{"data": fiber.Map{
				"rows":         entry.rows,
				"cached_until": entry.expiresAt.UTC().Format(time.RFC3339),
			}})
		}
	}

	tok, err := h.metaTokens.GetActiveByUser(ctx, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "conecte sua conta Meta antes — nenhum token ativo encontrado")
	}

	rows, err := h.metaClient.GetAccountInsightsBreakdown(ctx, tok.PlainToken, accountID, metaDim, preset)
	if err != nil {
		// Surface the Meta error message verbatim if available, otherwise a
		// generic 502 — never leak wrapped Go errors to the client.
		var me *metaads.MetaError
		if errors.As(err, &me) {
			slog.Warn("breakdowns: meta api error", "user", userID, "account", accountID, "dim", metaDim, "code", me.Code, "msg", me.Message)
			return fiber.NewError(fiber.StatusBadGateway, me.Message)
		}
		slog.Warn("breakdowns: client error", "user", userID, "account", accountID, "dim", metaDim, "err", err)
		return fiber.NewError(fiber.StatusBadGateway, "erro ao consultar a API do Meta")
	}
	if rows == nil {
		rows = []metaads.BreakdownRow{}
	}

	entry := &breakdownCacheEntry{rows: rows, expiresAt: time.Now().Add(breakdownTTL)}
	h.breakdownsCache.Store(cacheKey, entry)

	return c.JSON(fiber.Map{"data": fiber.Map{
		"rows":         rows,
		"cached_until": entry.expiresAt.UTC().Format(time.RFC3339),
	}})
}
