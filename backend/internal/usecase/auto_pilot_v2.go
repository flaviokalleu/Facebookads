package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/repository"
)

// AutoPilotV2 runs the deterministic safety rules across every active ad of
// every active account. mode='auto' actions execute immediately via Meta API;
// mode='propose' actions land in ai_actions_log for human review.
type AutoPilotV2 struct {
	db        *pgxpool.Pool
	meta      metaads.Client
	tokens    repository.MetaTokenRepository
	accounts  repository.MetaAdAccountRepository
	actions   repository.AIActionRepository
	rules     repository.AISafetyRuleRepository
}

func NewAutoPilotV2(
	db *pgxpool.Pool,
	meta metaads.Client,
	tokens repository.MetaTokenRepository,
	accounts repository.MetaAdAccountRepository,
	actions repository.AIActionRepository,
	rules repository.AISafetyRuleRepository,
) *AutoPilotV2 {
	return &AutoPilotV2{
		db: db, meta: meta, tokens: tokens, accounts: accounts,
		actions: actions, rules: rules,
	}
}

// Run iterates over every user with an active token.
func (a *AutoPilotV2) Run(ctx context.Context) error {
	users, err := a.listUsersWithActiveToken(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	for _, userID := range users {
		if err := a.runForUser(ctx, userID); err != nil {
			slog.Error("auto_pilot_v2: failed for user", "user_id", userID, "err", err)
		}
	}
	return nil
}

func (a *AutoPilotV2) runForUser(ctx context.Context, userID string) error {
	tok, err := a.tokens.GetActiveByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("active token: %w", err)
	}
	access := tok.PlainToken

	overrides, err := a.rules.ListByUser(ctx, userID)
	if err != nil {
		slog.Warn("auto_pilot_v2: load rule overrides", "user_id", userID, "err", err)
	}

	accs, err := a.accounts.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list accounts: %w", err)
	}

	for _, acc := range accs {
		if acc.AccountStatus != 1 {
			continue
		}
		if err := a.runForAccount(ctx, userID, access, acc, overrides); err != nil {
			slog.Warn("auto_pilot_v2: account failed",
				"user_id", userID, "account", acc.MetaID, "err", err)
		}
	}
	return nil
}

func (a *AutoPilotV2) runForAccount(ctx context.Context, userID, accessToken string, acc *domain.MetaAdAccount, overrides []*domain.AISafetyRule) error {
	rules := NewEffectiveRules(overrides, acc.MetaID)

	// Pull every active ad with last 7d aggregated insights from the parent campaign.
	type adRow struct {
		AdID, MetaAdID, AdName, AdStatus string
		AdCreatedAt                      time.Time
		Campaign                         *domain.Campaign
		Spend7d                          float64
		Leads7d                          int64
		AvgCTR                           float64
		AvgFreq                          float64
	}

	rows, err := a.db.Query(ctx, `
		SELECT
		  ads.id, ads.meta_ad_id, ads.name, ads.status, ads.created_at,
		  c.id, c.meta_campaign_id, c.user_id, c.ad_account_id, c.name, c.objective, c.status,
		  c.daily_budget, c.lifetime_budget, c.health_status,
		  COALESCE(s.spend_7d, 0)::float8,
		  COALESCE(s.leads_7d, 0)::bigint,
		  COALESCE(s.avg_ctr, 0)::float8,
		  COALESCE(s.avg_freq, 0)::float8
		FROM ads
		JOIN ad_sets aset ON aset.id = ads.ad_set_id AND aset.deleted_at IS NULL
		JOIN campaigns c ON c.id = aset.campaign_id AND c.deleted_at IS NULL
		LEFT JOIN LATERAL (
		  SELECT
		    SUM(spend) AS spend_7d,
		    SUM(leads) AS leads_7d,
		    AVG(ctr) AS avg_ctr,
		    AVG(frequency) AS avg_freq
		  FROM campaign_insights
		  WHERE campaign_id = c.id
		    AND date >= CURRENT_DATE - 7
		) s ON true
		WHERE c.user_id = $1
		  AND c.ad_account_id = $2
		  AND ads.status = 'ACTIVE'
		  AND ads.deleted_at IS NULL
	`, userID, acc.MetaID)
	if err != nil {
		return fmt.Errorf("query ads: %w", err)
	}
	defer rows.Close()

	var ads []adRow
	for rows.Next() {
		var r adRow
		var camp domain.Campaign
		var lastSynced *time.Time
		_ = lastSynced // not selected here
		if err := rows.Scan(
			&r.AdID, &r.MetaAdID, &r.AdName, &r.AdStatus, &r.AdCreatedAt,
			&camp.ID, &camp.MetaCampaignID, &camp.UserID, &camp.AdAccountID,
			&camp.Name, &camp.Objective, &camp.Status,
			&camp.DailyBudget, &camp.LifetimeBudget, &camp.HealthStatus,
			&r.Spend7d, &r.Leads7d, &r.AvgCTR, &r.AvgFreq,
		); err != nil {
			return fmt.Errorf("scan ad: %w", err)
		}
		r.Campaign = &camp
		ads = append(ads, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(ads) == 0 {
		return nil
	}

	// Account-wide CPL average over the last 7d.
	avgCPL, err := a.accountAvgCPL(ctx, userID, acc.MetaID)
	if err != nil {
		slog.Warn("auto_pilot_v2: avg cpl failed",
			"user_id", userID, "account", acc.MetaID, "err", err)
	}

	// Cap on pauses per account per day.
	maxPct := rules.Get(RuleMaxPausePctPerDay)
	maxPauses := int(float64(len(ads)) * maxPct)
	if maxPauses < 1 {
		maxPauses = 1
	}
	pausedSoFar, err := a.actions.CountPausesLast24h(ctx, userID, acc.MetaID)
	if err != nil {
		slog.Warn("auto_pilot_v2: count pauses failed", "err", err)
	}

	for _, r := range ads {
		ageHours := time.Since(r.AdCreatedAt).Hours()
		ad := &domain.Ad{
			ID:        r.AdID,
			MetaAdID:  r.MetaAdID,
			Name:      r.AdName,
			Status:    r.AdStatus,
			CreatedAt: r.AdCreatedAt,
		}
		eval := &AdEvaluation{
			Ad:            ad,
			Campaign:      r.Campaign,
			AccountMetaID: acc.MetaID,
			UserID:        userID,
			AdSpend7d:     r.Spend7d,
			AdLeads7d:     r.Leads7d,
			AdCTR7d:       r.AvgCTR,
			AdFreq7d:      r.AvgFreq,
			AccountAvgCPL: avgCPL,
			AdAgeHours:    ageHours,
		}
		action := EvaluateAd(eval, rules)
		if action == nil {
			continue
		}

		// Auto-execute pause actions, respecting the daily cap.
		if action.Mode == domain.AIActionModeAuto && action.ActionType == domain.AIActionTypePauseAd {
			if pausedSoFar >= maxPauses {
				slog.Info("auto_pilot_v2: pause cap reached, downgrading to propose",
					"user_id", userID, "account", acc.MetaID,
					"paused_so_far", pausedSoFar, "max", maxPauses)
				action.Mode = domain.AIActionModePropose
				if err := a.actions.Create(ctx, action); err != nil {
					slog.Warn("auto_pilot_v2: action insert failed", "err", err)
				}
				continue
			}

			if err := a.actions.Create(ctx, action); err != nil {
				slog.Warn("auto_pilot_v2: action insert failed", "err", err)
				continue
			}
			if err := a.executePauseAd(ctx, accessToken, action); err != nil {
				slog.Error("auto_pilot_v2: pause execution failed",
					"action_id", action.ID, "err", err)
				resp, _ := json.Marshal(map[string]string{"error": err.Error()})
				_ = a.actions.MarkFailed(ctx, action.ID, resp)
				continue
			}
			pausedSoFar++
			slog.Info("auto_pilot_v2: ad auto-paused",
				"user_id", userID, "account", acc.MetaID,
				"ad", action.TargetMetaID, "reason", action.Reason)
			continue
		}

		// Otherwise: log as proposal.
		if err := a.actions.Create(ctx, action); err != nil {
			slog.Warn("auto_pilot_v2: action insert failed", "err", err)
		}
	}
	return nil
}

// executePauseAd flips the ad to PAUSED via Graph API and marks the action executed.
func (a *AutoPilotV2) executePauseAd(ctx context.Context, accessToken string, action *domain.AIAction) error {
	err := a.meta.UpdateAd(ctx, accessToken, action.TargetMetaID, map[string]any{
		"status": "PAUSED",
	})
	if err != nil {
		return err
	}
	resp, _ := json.Marshal(map[string]any{"success": true, "ts": time.Now().UTC().Format(time.RFC3339)})
	return a.actions.MarkExecuted(ctx, action.ID, resp)
}

// accountAvgCPL = simple mean of (spend / leads) per day across all the
// account's campaigns over the last 7 days.
func (a *AutoPilotV2) accountAvgCPL(ctx context.Context, userID, accountMetaID string) (float64, error) {
	var totalSpend float64
	var totalLeads int64
	err := a.db.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(ci.spend), 0)::float8,
		  COALESCE(SUM(ci.leads), 0)::bigint
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id = $1
		  AND c.ad_account_id = $2
		  AND ci.date >= CURRENT_DATE - 7
		  AND c.deleted_at IS NULL
	`, userID, accountMetaID).Scan(&totalSpend, &totalLeads)
	if err != nil {
		return 0, err
	}
	if totalLeads == 0 {
		return 0, nil
	}
	return totalSpend / float64(totalLeads), nil
}

func (a *AutoPilotV2) listUsersWithActiveToken(ctx context.Context) ([]string, error) {
	rows, err := a.db.Query(ctx, `
		SELECT DISTINCT user_id::text FROM meta_tokens WHERE is_active = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ─── Approve / revert helpers used by handlers ────────────────────────────────

// ExecuteAction performs the Meta-side mutation for a previously-approved
// action. Pause actions flip status; budget actions update the ad set;
// rotate_creative is a stub that returns an error (out of scope).
func (a *AutoPilotV2) ExecuteAction(ctx context.Context, userID string, action *domain.AIAction) error {
	tok, err := a.tokens.GetActiveByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("active token: %w", err)
	}
	access := tok.PlainToken

	switch action.ActionType {
	case domain.AIActionTypePauseAd:
		if err := a.meta.UpdateAd(ctx, access, action.TargetMetaID, map[string]any{
			"status": "PAUSED",
		}); err != nil {
			resp, _ := json.Marshal(map[string]string{"error": err.Error()})
			_ = a.actions.MarkFailed(ctx, action.ID, resp)
			return err
		}
	case domain.AIActionTypePauseAdSet:
		if err := a.meta.UpdateAdSet(ctx, access, action.TargetMetaID, map[string]any{
			"status": "PAUSED",
		}); err != nil {
			resp, _ := json.Marshal(map[string]string{"error": err.Error()})
			_ = a.actions.MarkFailed(ctx, action.ID, resp)
			return err
		}
	case domain.AIActionTypeScaleBudget:
		// proposed_change should hold {"to": <reais>, "field": "daily_budget"}
		var change map[string]any
		_ = json.Unmarshal(action.ProposedChange, &change)
		newBudget, _ := change["to"].(float64)
		if newBudget <= 0 {
			return fmt.Errorf("scale_budget: invalid new budget")
		}
		cents := int(newBudget * 100)
		if err := a.meta.UpdateAdSet(ctx, access, action.TargetMetaID, map[string]any{
			"daily_budget": cents,
		}); err != nil {
			resp, _ := json.Marshal(map[string]string{"error": err.Error()})
			_ = a.actions.MarkFailed(ctx, action.ID, resp)
			return err
		}
	case domain.AIActionTypeAlert:
		// Alert actions don't mutate Meta — approving just acknowledges.
	default:
		return fmt.Errorf("action_type %s: execution not supported", action.ActionType)
	}

	resp, _ := json.Marshal(map[string]any{"success": true, "ts": time.Now().UTC().Format(time.RFC3339)})
	return a.actions.MarkExecuted(ctx, action.ID, resp)
}

// RevertAction reverses a pause_ad. Returns ErrValidation for unsupported types.
func (a *AutoPilotV2) RevertAction(ctx context.Context, userID string, action *domain.AIAction) error {
	if action.ActionType != domain.AIActionTypePauseAd && action.ActionType != domain.AIActionTypePauseAdSet {
		return fmt.Errorf("%w: revert not supported for %s", domain.ErrValidation, action.ActionType)
	}
	if !strings.EqualFold(action.Status, domain.AIActionStatusExecuted) {
		return fmt.Errorf("%w: only executed actions can be reverted", domain.ErrValidation)
	}
	tok, err := a.tokens.GetActiveByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("active token: %w", err)
	}
	access := tok.PlainToken

	if action.ActionType == domain.AIActionTypePauseAd {
		if err := a.meta.UpdateAd(ctx, access, action.TargetMetaID, map[string]any{
			"status": "ACTIVE",
		}); err != nil {
			return err
		}
	} else {
		if err := a.meta.UpdateAdSet(ctx, access, action.TargetMetaID, map[string]any{
			"status": "ACTIVE",
		}); err != nil {
			return err
		}
	}
	return a.actions.MarkReverted(ctx, action.ID)
}
