package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/repository"
)

// SyncMetaCampaigns pulls campaigns / ad sets / ads / insights from each
// auto-discovered Meta ad account and upserts them into the legacy tables
// (`campaigns`, `ad_sets`, `ads`, `campaign_insights`). The existing AI
// agents read those tables, so they keep working unchanged.
type SyncMetaCampaigns struct {
	meta       metaads.Client
	tokens     repository.MetaTokenRepository
	accounts   repository.MetaAdAccountRepository
	campaigns  repository.CampaignRepository
	adSets     repository.AdSetRepository
	ads        repository.AdRepository
	insights   repository.InsightRepository
}

func NewSyncMetaCampaigns(
	meta metaads.Client,
	tokens repository.MetaTokenRepository,
	accounts repository.MetaAdAccountRepository,
	campaigns repository.CampaignRepository,
	adSets repository.AdSetRepository,
	ads repository.AdRepository,
	insights repository.InsightRepository,
) *SyncMetaCampaigns {
	return &SyncMetaCampaigns{
		meta: meta, tokens: tokens, accounts: accounts,
		campaigns: campaigns, adSets: adSets, ads: ads, insights: insights,
	}
}

// Run pulls every active ad account for the user. Per-account errors are
// logged and tolerated.
func (u *SyncMetaCampaigns) Run(ctx context.Context, userID string) error {
	tok, err := u.tokens.GetActiveByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("active meta token: %w", err)
	}
	access := tok.PlainToken
	if access == "" {
		return fmt.Errorf("meta token: empty plaintext")
	}

	accs, err := u.accounts.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list ad accounts: %w", err)
	}
	if len(accs) == 0 {
		slog.Info("sync_meta_campaigns: no accounts", "user_id", userID)
		return nil
	}

	// Token-scoped rate counter — sleep 1s after every 50 calls.
	var calls int64

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(4)
	for _, acc := range accs {
		acc := acc
		if acc.AccountStatus != 1 {
			continue
		}
		g.Go(func() error {
			if err := u.syncAccount(gctx, userID, access, acc, &calls); err != nil {
				slog.Warn("sync_meta_campaigns: account failed",
					"user_id", userID, "account", acc.MetaID, "err", err)
			}
			return nil
		})
	}
	return g.Wait()
}

func (u *SyncMetaCampaigns) syncAccount(ctx context.Context, userID, accessToken string, acc *domain.MetaAdAccount, calls *int64) error {
	numericID := strings.TrimPrefix(acc.MetaID, "act_")

	rateBump(ctx, calls)
	metaCampaigns, err := u.meta.GetCampaigns(ctx, accessToken, numericID)
	if err != nil {
		return fmt.Errorf("get campaigns: %w", err)
	}

	for _, mc := range metaCampaigns {
		c := metaCampaignToDomain(mc, userID, acc.MetaID)
		if err := u.campaigns.Upsert(ctx, c); err != nil {
			slog.Warn("sync_meta_campaigns: campaign upsert failed",
				"meta_id", mc.ID, "err", err)
			continue
		}

		rateBump(ctx, calls)
		metaAdSets, err := u.meta.GetAdSets(ctx, accessToken, mc.ID)
		if err != nil {
			slog.Warn("sync_meta_campaigns: ad sets failed",
				"campaign", mc.ID, "err", err)
		} else {
			for _, mas := range metaAdSets {
				as := &domain.AdSet{
					MetaAdSetID:      mas.ID,
					CampaignID:       c.ID,
					Name:             mas.Name,
					Status:           mas.Status,
					DailyBudget:      parseFloatPtr(mas.DailyBudget),
					OptimizationGoal: mas.OptimizationGoal,
					BillingEvent:     mas.BillingEvent,
				}
				if err := u.adSets.Upsert(ctx, as); err != nil {
					slog.Warn("sync_meta_campaigns: ad set upsert failed",
						"meta_id", mas.ID, "err", err)
					continue
				}
				rateBump(ctx, calls)
				metaAds, err := u.meta.GetAds(ctx, accessToken, mas.ID)
				if err != nil {
					slog.Warn("sync_meta_campaigns: ads failed",
						"ad_set", mas.ID, "err", err)
					continue
				}
				for _, ma := range metaAds {
					ad := &domain.Ad{
						MetaAdID:      ma.ID,
						AdSetID:       as.ID,
						Name:          ma.Name,
						Status:        ma.Status,
						CreativeTitle: ma.Creative.Title,
						CreativeBody:  ma.Creative.Body,
					}
					if err := u.ads.Upsert(ctx, ad); err != nil {
						slog.Warn("sync_meta_campaigns: ad upsert failed",
							"meta_id", ma.ID, "err", err)
					}
				}
			}
		}

		// Meta's "last_30d" exclui o dia corrente. Para refletir o que o usuário
		// vê no Gerenciador de Anúncios em tempo real, fazemos uma 2ª chamada
		// com preset "today" e mergiamos.
		rateBump(ctx, calls)
		hist, err := u.meta.GetInsights(ctx, accessToken, mc.ID, "last_30d")
		if err != nil {
			slog.Warn("sync_meta_campaigns: insights failed",
				"campaign", mc.ID, "err", err)
		}
		for _, r := range hist {
			ins, perr := metaInsightToDomain(r, c.ID)
			if perr != nil {
				continue
			}
			if err := u.insights.Upsert(ctx, ins); err != nil {
				slog.Warn("sync_meta_campaigns: insight upsert failed",
					"campaign", mc.ID, "err", err)
			}
		}

		rateBump(ctx, calls)
		today, err := u.meta.GetInsights(ctx, accessToken, mc.ID, "today")
		if err != nil {
			slog.Warn("sync_meta_campaigns: today insights failed",
				"campaign", mc.ID, "err", err)
			continue
		}
		for _, r := range today {
			ins, perr := metaInsightToDomain(r, c.ID)
			if perr != nil {
				continue
			}
			if err := u.insights.Upsert(ctx, ins); err != nil {
				slog.Warn("sync_meta_campaigns: today upsert failed",
					"campaign", mc.ID, "err", err)
			}
		}
	}
	return nil
}

// rateBump increments the per-token counter. Every 50 calls it sleeps 1s to
// stay well under Meta's per-token rate limit.
func rateBump(ctx context.Context, calls *int64) {
	n := atomic.AddInt64(calls, 1)
	if n%50 == 0 {
		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
		}
	}
}
