package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/repository"
)

type CampaignUseCase struct {
	campaigns  repository.CampaignRepository
	insights   repository.InsightRepository
	tokens     repository.UserTokenRepository
	adSets     repository.AdSetRepository
	ads        repository.AdRepository
	metaClient metaads.Client
}

func NewCampaignUseCase(
	campaigns repository.CampaignRepository,
	insights repository.InsightRepository,
	tokens repository.UserTokenRepository,
	adSets repository.AdSetRepository,
	ads repository.AdRepository,
	metaClient metaads.Client,
) *CampaignUseCase {
	return &CampaignUseCase{
		campaigns:  campaigns,
		insights:   insights,
		tokens:     tokens,
		adSets:     adSets,
		ads:        ads,
		metaClient: metaClient,
	}
}

// CampaignWithMetrics enriches a campaign with computed 7d/30d metrics.
type CampaignWithMetrics struct {
	*domain.Campaign
	Spend30d  float64 `json:"spend_30d"`
	Leads30d  int64   `json:"leads_30d"`
	AvgCTR7d  float64 `json:"avg_ctr_7d"`
	AvgCPC7d  float64 `json:"avg_cpc_7d"`
	AvgROAS7d float64 `json:"avg_roas_7d"`
}

func (uc *CampaignUseCase) List(ctx context.Context, userID string) ([]*CampaignWithMetrics, error) {
	campaigns, err := uc.campaigns.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	from7d := time.Now().AddDate(0, 0, -7)
	from30d := time.Now().AddDate(0, 0, -30)
	to := time.Now()

	result := make([]*CampaignWithMetrics, 0, len(campaigns))
	for _, c := range campaigns {
		cm := &CampaignWithMetrics{Campaign: c}

		rows30, _ := uc.insights.ListByCampaign(ctx, c.ID, from30d, to)
		for _, r := range rows30 {
			cm.Spend30d += r.Spend
			cm.Leads30d += r.Leads
		}
		cm.Spend30d = round2(cm.Spend30d)

		rows7, _ := uc.insights.ListByCampaign(ctx, c.ID, from7d, to)
		if n := len(rows7); n > 0 {
			var sCTR, sCPC, sROAS float64
			for _, r := range rows7 {
				sCTR += r.CTR
				sCPC += r.CPC
				sROAS += r.ROAS
			}
			cm.AvgCTR7d = round2(sCTR / float64(n))
			cm.AvgCPC7d = round2(sCPC / float64(n))
			cm.AvgROAS7d = round2(sROAS / float64(n))
		}
		result = append(result, cm)
	}
	return result, nil
}

func (uc *CampaignUseCase) Get(ctx context.Context, userID, campaignID string) (*domain.Campaign, error) {
	c, err := uc.campaigns.GetByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if c.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return c, nil
}

func (uc *CampaignUseCase) Create(ctx context.Context, userID string, input CreateCampaignInput) (*domain.Campaign, error) {
	now := time.Now()
	c := &domain.Campaign{
		MetaCampaignID: "manual_" + userID + "_" + strconv.FormatInt(time.Now().UnixMilli(), 36),
		UserID:         userID,
		AdAccountID:    input.AdAccountID,
		Name:           input.Name,
		Objective:      input.Objective,
		Status:         "ACTIVE",
		DailyBudget:    input.DailyBudget,
		LifetimeBudget: input.LifetimeBudget,
		HealthStatus:   domain.HealthHealthy,
		LastSyncedAt:   &now,
	}
	if err := uc.campaigns.Upsert(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

type CreateCampaignInput struct {
	AdAccountID    string   `json:"ad_account_id"`
	Name           string   `json:"name"`
	Objective      string   `json:"objective"`
	DailyBudget    *float64 `json:"daily_budget,omitempty"`
	LifetimeBudget *float64 `json:"lifetime_budget,omitempty"`
}

type UpdateCampaignInput struct {
	Name           *string   `json:"name,omitempty"`
	Status         *string   `json:"status,omitempty"`
	Objective      *string   `json:"objective,omitempty"`
	DailyBudget    *float64  `json:"daily_budget,omitempty"`
	LifetimeBudget *float64  `json:"lifetime_budget,omitempty"`
	HealthStatus   *string   `json:"health_status,omitempty"`
}

func (uc *CampaignUseCase) Update(ctx context.Context, userID, campaignID string, input UpdateCampaignInput) (*domain.Campaign, error) {
	c, err := uc.campaigns.GetByID(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if c.UserID != userID {
		return nil, domain.ErrForbidden
	}
	if input.Name != nil { c.Name = *input.Name }
	if input.Status != nil { c.Status = *input.Status }
	if input.Objective != nil { c.Objective = *input.Objective }
	if input.DailyBudget != nil { c.DailyBudget = input.DailyBudget }
	if input.LifetimeBudget != nil { c.LifetimeBudget = input.LifetimeBudget }
	if input.HealthStatus != nil { c.HealthStatus = domain.HealthStatus(*input.HealthStatus) }
	if err := uc.campaigns.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (uc *CampaignUseCase) Delete(ctx context.Context, userID, campaignID string) error {
	c, err := uc.campaigns.GetByID(ctx, campaignID)
	if err != nil {
		return err
	}
	if c.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.campaigns.MarkDeleted(ctx, campaignID)
}

// Sync pulls campaigns, ad sets, ads, and insights from Meta.
func (uc *CampaignUseCase) Sync(ctx context.Context, userID string) (synced int, err error) {
	tokens, err := uc.tokens.ListByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("list tokens: %w", err)
	}
	if len(tokens) == 0 {
		return 0, fmt.Errorf("%w: no Meta ad accounts connected", domain.ErrNotFound)
	}

	for _, tok := range tokens {
		plainToken, decErr := decryptToken(tok.EncryptedToken)
		if decErr != nil {
			slog.Error("sync: decrypt token failed", "account", tok.AdAccountID, "err", decErr)
			continue
		}

		metaCampaigns, fetchErr := uc.metaClient.GetCampaigns(ctx, plainToken, tok.AdAccountID)
		if fetchErr != nil {
			slog.Error("sync: get campaigns failed", "account", tok.AdAccountID, "err", fetchErr)
			continue
		}

		for _, mc := range metaCampaigns {
			c := metaCampaignToDomain(mc, userID, tok.AdAccountID)
			if upsertErr := uc.campaigns.Upsert(ctx, c); upsertErr != nil {
				slog.Error("sync: upsert campaign failed", "meta_id", mc.ID, "err", upsertErr)
				continue
			}
			synced++

			// Sync ad sets for this campaign
			if err := uc.syncAdSets(ctx, plainToken, c.ID, mc.ID); err != nil {
				slog.Error("sync: ad sets failed", "campaign", mc.ID, "err", err)
			}

			// Sync insights for this campaign
			if insightErr := uc.syncInsights(ctx, plainToken, c); insightErr != nil {
				slog.Error("sync: insights failed", "campaign", mc.ID, "err", insightErr)
			}
		}
	}
	return synced, nil
}

func (uc *CampaignUseCase) syncAdSets(ctx context.Context, token, campaignLocalID, metaCampaignID string) error {
	metaAdSets, err := uc.metaClient.GetAdSets(ctx, token, metaCampaignID)
	if err != nil {
		return fmt.Errorf("get ad sets: %w", err)
	}
	for _, mas := range metaAdSets {
		a := &domain.AdSet{
			MetaAdSetID:      mas.ID,
			CampaignID:       campaignLocalID,
			Name:             mas.Name,
			Status:           mas.Status,
			DailyBudget:      parseFloatPtr(mas.DailyBudget),
			OptimizationGoal: mas.OptimizationGoal,
			BillingEvent:     mas.BillingEvent,
		}
		if err := uc.adSets.Upsert(ctx, a); err != nil {
			slog.Error("sync: upsert ad set failed", "meta_id", mas.ID, "err", err)
			continue
		}

		// Sync ads for this ad set
		if err := uc.syncAds(ctx, token, a.ID, mas.ID); err != nil {
			slog.Error("sync: ads failed", "ad_set", mas.ID, "err", err)
		}
	}
	return nil
}

func (uc *CampaignUseCase) syncAds(ctx context.Context, token, adSetLocalID, metaAdSetID string) error {
	metaAds, err := uc.metaClient.GetAds(ctx, token, metaAdSetID)
	if err != nil {
		return fmt.Errorf("get ads: %w", err)
	}
	for _, ma := range metaAds {
		ad := &domain.Ad{
			MetaAdID:      ma.ID,
			AdSetID:       adSetLocalID,
			Name:          ma.Name,
			Status:        ma.Status,
			CreativeTitle: ma.Creative.Title,
			CreativeBody:  ma.Creative.Body,
			CTAType:       "",
		}
		if err := uc.ads.Upsert(ctx, ad); err != nil {
			slog.Error("sync: upsert ad failed", "meta_id", ma.ID, "err", err)
		}
	}
	return nil
}

func (uc *CampaignUseCase) syncInsights(ctx context.Context, token string, c *domain.Campaign) error {
	raw, err := uc.metaClient.GetInsights(ctx, token, c.MetaCampaignID, "last_30d")
	if err != nil {
		return err
	}
	for _, r := range raw {
		insight, parseErr := metaInsightToDomain(r, c.ID)
		if parseErr != nil {
			slog.Warn("sync: parse insight failed", "err", parseErr)
			continue
		}
		if err := uc.insights.Upsert(ctx, insight); err != nil {
			return err
		}
	}
	return nil
}

// AdSets returns all ad sets for a campaign.
func (uc *CampaignUseCase) AdSets(ctx context.Context, campaignID string) ([]*domain.AdSet, error) {
	return uc.adSets.ListByCampaign(ctx, campaignID)
}

// Ads returns all ads for an ad set.
func (uc *CampaignUseCase) Ads(ctx context.Context, adSetID string) ([]*domain.Ad, error) {
	return uc.ads.ListByAdSet(ctx, adSetID)
}

// GetAdsByCampaign returns all ads for a given campaign (via ad sets).
func (uc *CampaignUseCase) GetAdsByCampaign(ctx context.Context, campaignID string) ([]*domain.Ad, error) {
	adSets, err := uc.adSets.ListByCampaign(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	var all []*domain.Ad
	for _, as := range adSets {
		ads, err := uc.ads.ListByAdSet(ctx, as.ID)
		if err != nil {
			continue
		}
		all = append(all, ads...)
	}
	return all, nil
}

// ─── Converters ──────────────────────────────────────────────────────────────

func metaCampaignToDomain(mc metaads.MetaCampaign, userID, adAccountID string) *domain.Campaign {
	now := time.Now()
	c := &domain.Campaign{
		MetaCampaignID: mc.ID,
		UserID:         userID,
		AdAccountID:    adAccountID,
		Name:           mc.Name,
		Objective:      mc.Objective,
		Status:         mc.Status,
		HealthStatus:   domain.HealthHealthy,
		LastSyncedAt:   &now,
	}
	if mc.DailyBudget != "" {
		if v, err := strconv.ParseFloat(mc.DailyBudget, 64); err == nil {
			cents := v / 100
			c.DailyBudget = &cents
		}
	}
	if mc.LifetimeBudget != "" {
		if v, err := strconv.ParseFloat(mc.LifetimeBudget, 64); err == nil {
			cents := v / 100
			c.LifetimeBudget = &cents
		}
	}
	return c
}

func metaInsightToDomain(r metaads.MetaInsight, campaignID string) (*domain.CampaignInsight, error) {
	date, err := time.Parse("2006-01-02", r.DateStart)
	if err != nil {
		return nil, fmt.Errorf("parse date %s: %w", r.DateStart, err)
	}

	i := &domain.CampaignInsight{
		CampaignID:  campaignID,
		Date:        date,
		Spend:       parseF(r.Spend),
		Impressions: parseInt(r.Impressions),
		Clicks:      parseInt(r.Clicks),
		CTR:         parseF(r.CTR),
		CPC:         parseF(r.CPC),
		CPM:         parseF(r.CPM),
		Reach:       parseInt(r.Reach),
		Frequency:   parseF(r.Frequency),
	}

	for _, action := range r.Actions {
		switch action.ActionType {
		case "lead", "onsite_conversion.lead_grouped",
			"onsite_conversion.total_messaging_connection":
			i.Leads += parseInt(action.Value)
		case "purchase", "offsite_conversion.fb_pixel_purchase":
			i.Purchases += parseInt(action.Value)
		}
	}

	if i.Spend > 0 {
		for _, av := range r.Actions {
			if av.ActionType == "offsite_conversion.fb_pixel_purchase" {
				purchaseValue := parseF(av.Value)
				i.ROAS = purchaseValue / i.Spend
			}
		}
	}

	return i, nil
}

func parseF(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseInt(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func parseFloatPtr(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// decryptToken is a stub — real implementation delegates to config.Service.
func decryptToken(encrypted string) (string, error) {
	return encrypted, nil
}
