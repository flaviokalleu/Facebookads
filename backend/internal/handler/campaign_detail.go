package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/middleware"
)

type CampaignDetailHandler struct {
	db *pgxpool.Pool
}

func NewCampaignDetailHandler(db *pgxpool.Pool) *CampaignDetailHandler {
	return &CampaignDetailHandler{db: db}
}

// Get handles GET /api/v1/campanhas/:id?days=N
// Returns: campaign + account + daily insights + adsets + ads + ai actions.
func (h *CampaignDetailHandler) Get(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	id := c.Params("id")
	ctx := c.UserContext()

	days := c.QueryInt("days", 14)
	if days < 1 || days > 90 {
		days = 14
	}

	// 1) Campaign + account info.
	type campaign struct {
		ID              string     `json:"id"`
		MetaCampaignID  string     `json:"meta_campaign_id"`
		Name            string     `json:"name"`
		Status          string     `json:"status"`
		Objective       string     `json:"objective"`
		DailyBudget     float64    `json:"daily_budget"`
		LifetimeBudget  float64    `json:"lifetime_budget"`
		HealthStatus   string      `json:"health_status"`
		AccountMetaID   string     `json:"account_meta_id"`
		AccountName     string     `json:"account_name"`
		BMName          string     `json:"bm_name"`
		MetaCreatedTime *time.Time `json:"meta_created_time,omitempty"`
		MetaStartTime   *time.Time `json:"meta_start_time,omitempty"`
		MetaStopTime    *time.Time `json:"meta_stop_time,omitempty"`
		FirstInsightAt  *time.Time `json:"first_insight_date,omitempty"`
		LastInsightAt   *time.Time `json:"last_insight_date,omitempty"`
		DaysRunning     int        `json:"days_running"`
	}
	var camp campaign
	err := h.db.QueryRow(ctx, `
		SELECT c.id, c.meta_campaign_id, c.name, c.status, c.objective,
		       COALESCE(c.daily_budget, 0)::float8,
		       COALESCE(c.lifetime_budget, 0)::float8,
		       c.health_status,
		       c.ad_account_id,
		       COALESCE(a.name, ''), COALESCE(b.name, ''),
		       c.meta_created_time, c.meta_start_time, c.meta_stop_time,
		       (SELECT MIN(date) FROM campaign_insights WHERE campaign_id = c.id),
		       (SELECT MAX(date) FROM campaign_insights WHERE campaign_id = c.id)
		FROM campaigns c
		LEFT JOIN meta_ad_accounts a ON a.user_id = c.user_id AND a.meta_id = c.ad_account_id
		LEFT JOIN business_managers b ON b.id = a.bm_id
		WHERE c.id = $1 AND c.user_id = $2 AND c.deleted_at IS NULL
	`, id, userID).Scan(
		&camp.ID, &camp.MetaCampaignID, &camp.Name, &camp.Status, &camp.Objective,
		&camp.DailyBudget, &camp.LifetimeBudget, &camp.HealthStatus,
		&camp.AccountMetaID, &camp.AccountName, &camp.BMName,
		&camp.MetaCreatedTime, &camp.MetaStartTime, &camp.MetaStopTime,
		&camp.FirstInsightAt, &camp.LastInsightAt,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "campanha não encontrada")
	}
	switch {
	case camp.MetaStartTime != nil:
		camp.DaysRunning = int(time.Since(*camp.MetaStartTime).Hours()/24) + 1
	case camp.MetaCreatedTime != nil:
		camp.DaysRunning = int(time.Since(*camp.MetaCreatedTime).Hours()/24) + 1
	case camp.FirstInsightAt != nil:
		camp.DaysRunning = int(time.Since(*camp.FirstInsightAt).Hours()/24) + 1
	}
	if camp.DaysRunning < 0 {
		camp.DaysRunning = 0
	}

	// 2) Aggregated KPIs over the chosen window.
	type kpi struct {
		Spend       float64 `json:"spend"`
		Impressions int64   `json:"impressions"`
		Clicks      int64   `json:"clicks"`
		Leads       int64   `json:"leads"`
		CTR         float64 `json:"ctr"`
		CPL         float64 `json:"cpl"`
		Frequency   float64 `json:"avg_frequency"`
		SpendPrev   float64 `json:"spend_prev"`
		LeadsPrev   int64   `json:"leads_prev"`
		CPLPrev     float64 `json:"cpl_prev"`
	}
	var k kpi
	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(spend),0), COALESCE(SUM(impressions),0)::bigint,
		       COALESCE(SUM(clicks),0)::bigint, COALESCE(SUM(leads),0)::bigint,
		       COALESCE(AVG(frequency),0)
		FROM campaign_insights
		WHERE campaign_id = $1 AND date > CURRENT_DATE - $2::int
	`, id, days).Scan(&k.Spend, &k.Impressions, &k.Clicks, &k.Leads, &k.Frequency); err == nil {
		if k.Impressions > 0 {
			k.CTR = float64(k.Clicks) / float64(k.Impressions)
		}
		if k.Leads > 0 {
			k.CPL = k.Spend / float64(k.Leads)
		}
	}
	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(spend),0), COALESCE(SUM(leads),0)::bigint
		FROM campaign_insights
		WHERE campaign_id = $1
		  AND date > CURRENT_DATE - ($2::int * 2)
		  AND date <= CURRENT_DATE - $2::int
	`, id, days).Scan(&k.SpendPrev, &k.LeadsPrev); err == nil && k.LeadsPrev > 0 {
		k.CPLPrev = k.SpendPrev / float64(k.LeadsPrev)
	}

	// 3) Daily insights for chart.
	dailyDays := days
	if dailyDays < 7 {
		dailyDays = 7
	}
	type point struct {
		Date        string  `json:"date"`
		Spend       float64 `json:"spend"`
		Leads       int64   `json:"leads"`
		Impressions int64   `json:"impressions"`
		Clicks      int64   `json:"clicks"`
		Frequency   float64 `json:"frequency"`
	}
	daily := make([]point, 0, dailyDays)
	if rows, err := h.db.Query(ctx, `
		WITH dates AS (
		  SELECT generate_series(CURRENT_DATE - ($2::int - 1), CURRENT_DATE, '1 day')::date AS d
		)
		SELECT d.d,
		       COALESCE(ci.spend,0), COALESCE(ci.leads,0)::bigint,
		       COALESCE(ci.impressions,0)::bigint, COALESCE(ci.clicks,0)::bigint,
		       COALESCE(ci.frequency,0)
		FROM dates d
		LEFT JOIN campaign_insights ci ON ci.campaign_id = $1 AND ci.date = d.d
		ORDER BY d.d
	`, id, dailyDays); err == nil {
		defer rows.Close()
		for rows.Next() {
			var p point
			var d time.Time
			if err := rows.Scan(&d, &p.Spend, &p.Leads, &p.Impressions, &p.Clicks, &p.Frequency); err == nil {
				p.Date = d.Format("2006-01-02")
				daily = append(daily, p)
			}
		}
	}

	// 4) Adsets with their windowed insights (joining campaign-level insights since
	//    we don't have per-adset insights — proxy: split by share of spend... too
	//    complex. For now show adset metadata only and per-campaign insights.).
	type adset struct {
		ID               string     `json:"id"`
		MetaAdSetID      string     `json:"meta_adset_id"`
		Name             string     `json:"name"`
		Status           string     `json:"status"`
		DailyBudget      float64    `json:"daily_budget"`
		OptimizationGoal string     `json:"optimization_goal"`
		BillingEvent     string     `json:"billing_event"`
		MetaStartTime    *time.Time `json:"meta_start_time,omitempty"`
		MetaEndTime      *time.Time `json:"meta_end_time,omitempty"`
		AdsCount         int        `json:"ads_count"`
		ActiveAdsCount   int        `json:"active_ads_count"`
	}
	adsets := make([]adset, 0)
	if rows, err := h.db.Query(ctx, `
		SELECT aset.id, aset.meta_ad_set_id, aset.name, aset.status,
		       COALESCE(aset.daily_budget, 0)::float8,
		       COALESCE(aset.optimization_goal, ''),
		       COALESCE(aset.billing_event, ''),
		       aset.meta_start_time, aset.meta_end_time,
		       (SELECT COUNT(*) FROM ads WHERE ad_set_id = aset.id AND deleted_at IS NULL),
		       (SELECT COUNT(*) FROM ads WHERE ad_set_id = aset.id AND status = 'ACTIVE' AND deleted_at IS NULL)
		FROM ad_sets aset
		WHERE aset.campaign_id = $1 AND aset.deleted_at IS NULL
		ORDER BY aset.created_at
	`, id); err == nil {
		defer rows.Close()
		for rows.Next() {
			var a adset
			if err := rows.Scan(&a.ID, &a.MetaAdSetID, &a.Name, &a.Status,
				&a.DailyBudget, &a.OptimizationGoal, &a.BillingEvent,
				&a.MetaStartTime, &a.MetaEndTime,
				&a.AdsCount, &a.ActiveAdsCount); err == nil {
				adsets = append(adsets, a)
			}
		}
	}

	// 5) Ads list with creative info.
	type ad struct {
		ID             string `json:"id"`
		MetaAdID       string `json:"meta_ad_id"`
		AdSetID        string `json:"ad_set_id"`
		AdSetName      string `json:"adset_name"`
		Name           string `json:"name"`
		Status         string `json:"status"`
		CreativeTitle  string `json:"creative_title"`
		CreativeBody   string `json:"creative_body"`
		ImageURL       string `json:"image_url"`
		CTAType        string `json:"cta_type"`
	}
	ads := make([]ad, 0)
	if rows, err := h.db.Query(ctx, `
		SELECT ads.id, ads.meta_ad_id, ads.ad_set_id, COALESCE(aset.name, ''),
		       ads.name, ads.status,
		       COALESCE(ads.creative_title,''), COALESCE(ads.creative_body,''),
		       COALESCE(ads.image_url,''), COALESCE(ads.cta_type,'')
		FROM ads
		LEFT JOIN ad_sets aset ON aset.id = ads.ad_set_id
		WHERE aset.campaign_id = $1 AND ads.deleted_at IS NULL
		ORDER BY ads.status DESC, ads.created_at
	`, id); err == nil {
		defer rows.Close()
		for rows.Next() {
			var a ad
			if err := rows.Scan(&a.ID, &a.MetaAdID, &a.AdSetID, &a.AdSetName,
				&a.Name, &a.Status, &a.CreativeTitle, &a.CreativeBody,
				&a.ImageURL, &a.CTAType); err == nil {
				ads = append(ads, a)
			}
		}
	}

	// 6) Recent AI actions touching this campaign or its children.
	type aiActionRow struct {
		ID         string    `json:"id"`
		ActionType string    `json:"action_type"`
		TargetKind string    `json:"target_kind"`
		Reason     string    `json:"reason"`
		Status     string    `json:"status"`
		Source     string    `json:"source"`
		Mode       string    `json:"mode"`
		CreatedAt  time.Time `json:"created_at"`
	}
	aiActions := make([]aiActionRow, 0)
	if rows, err := h.db.Query(ctx, `
		SELECT id, action_type, target_kind, reason, status, source, mode, created_at
		FROM ai_actions_log
		WHERE user_id = $1
		  AND (
		    target_meta_id = $2
		    OR target_meta_id IN (SELECT meta_ad_set_id FROM ad_sets WHERE campaign_id = $3 AND deleted_at IS NULL)
		    OR target_meta_id IN (
		      SELECT ads.meta_ad_id FROM ads
		      JOIN ad_sets aset ON aset.id = ads.ad_set_id
		      WHERE aset.campaign_id = $3 AND ads.deleted_at IS NULL
		    )
		  )
		ORDER BY created_at DESC LIMIT 20
	`, userID, camp.MetaCampaignID, id); err == nil {
		defer rows.Close()
		for rows.Next() {
			var r aiActionRow
			if err := rows.Scan(&r.ID, &r.ActionType, &r.TargetKind, &r.Reason,
				&r.Status, &r.Source, &r.Mode, &r.CreatedAt); err == nil {
				aiActions = append(aiActions, r)
			}
		}
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"campaign":   camp,
		"kpis":       k,
		"daily":      daily,
		"adsets":     adsets,
		"ads":        ads,
		"ai_actions": aiActions,
	}})
}
