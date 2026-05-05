package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/repository"
)

// Safety rule keys
const (
	RulePauseCPLRatio          = "pause_cpl_ratio"
	RuleMinSpendToPause        = "min_spend_to_pause"
	RuleMinAgeHours            = "min_age_hours"
	RuleMinConversionsToDecide = "min_conversions_to_decide"
	RuleRespectLearningPhase   = "respect_learning_phase"
	RuleMaxPausePctPerDay      = "max_pause_pct_per_day"
	RuleAlertCPLRatio          = "alert_cpl_ratio"
	RuleAlertCTRMin            = "alert_ctr_min"
	RuleAlertFreqMax           = "alert_freq_max"
	RuleScaleCPLRatio          = "scale_cpl_ratio"
	RuleScaleMinCTR            = "scale_min_ctr"
	RuleScaleMaxFreq           = "scale_max_freq"
	RuleScaleFactor            = "scale_factor"
)

// DefaultSafetyRules holds the hardcoded baseline thresholds. Per-user overrides
// from the ai_safety_rules table win when present.
//
// Maturity guardrail: nothing fires for campaigns younger than 7 days OR with
// fewer than 50 conversions in the last 7 days (Meta's learning-phase exit
// threshold). This avoids acting on noise.
var DefaultSafetyRules = map[string]float64{
	RulePauseCPLRatio:          3.0,
	RuleMinSpendToPause:        30.0,
	RuleMinAgeHours:            168.0, // 7 dias — campanha precisa estar madura
	RuleMinConversionsToDecide: 50.0,  // Meta exige ~50 eventos/7d pra sair do learning
	RuleRespectLearningPhase:   1.0,   // 1 = on, 0 = off
	RuleMaxPausePctPerDay:      0.30,
	RuleAlertCPLRatio:          1.5,
	RuleAlertCTRMin:            0.01,
	RuleAlertFreqMax:           3.5,
	RuleScaleCPLRatio:          0.6, // CPL ≤ 0.6× média = winner
	RuleScaleMinCTR:            0.025,
	RuleScaleMaxFreq:           2.5,
	RuleScaleFactor:            1.2, // sugere +20% de verba
}

// EffectiveRules merges defaults with per-user overrides.
type EffectiveRules struct {
	values map[string]float64
}

func NewEffectiveRules(overrides []*domain.AISafetyRule, accountMetaID string) *EffectiveRules {
	v := make(map[string]float64, len(DefaultSafetyRules))
	for k, val := range DefaultSafetyRules {
		v[k] = val
	}
	// Two-pass: first apply user-global overrides, then per-account ones.
	for _, r := range overrides {
		if r.AccountMetaID == nil {
			v[r.RuleKey] = r.RuleValue
		}
	}
	for _, r := range overrides {
		if r.AccountMetaID != nil && *r.AccountMetaID == accountMetaID {
			v[r.RuleKey] = r.RuleValue
		}
	}
	return &EffectiveRules{values: v}
}

func (r *EffectiveRules) Get(key string) float64 {
	if v, ok := r.values[key]; ok {
		return v
	}
	return DefaultSafetyRules[key]
}

func (r *EffectiveRules) All() map[string]float64 {
	out := make(map[string]float64, len(r.values))
	for k, v := range r.values {
		out[k] = v
	}
	return out
}

// AdEvaluation is the input for EvaluateAd. We pass the data from the DB
// instead of fetching it inside, so the caller can batch / cache.
type AdEvaluation struct {
	Ad                  *domain.Ad
	Campaign            *domain.Campaign
	AdSetMetaID         string  // for scale_budget target
	AccountMetaID       string
	UserID              string
	AdSpend7d           float64
	AdLeads7d           int64
	AdCTR7d             float64
	AdFreq7d            float64
	AdSetDailyBudget    float64 // BRL — used to compute proposed scale-up
	AccountAvgCPL       float64
	AdAgeHours          float64 // age of the ad itself
	CampaignAgeHours    float64 // age of the parent campaign
	CampaignLeads7d     int64   // proxy for "out of learning phase"
}

// EvaluateAd applies the deterministic safety rules to one ad. Returns nil
// when the ad does not warrant any action. The returned AIAction is fully
// populated except for ID/CreatedAt — caller persists via the repository.
//
// Maturity gate: never fire on a campaign that is younger than min_age_hours
// (default 7 days) OR still in Meta's learning phase (proxy: campaign accumulated
// fewer than min_conversions_to_decide events in the last 7d).
func EvaluateAd(eval *AdEvaluation, rules *EffectiveRules) *domain.AIAction {
	minAge := rules.Get(RuleMinAgeHours)
	minConv := rules.Get(RuleMinConversionsToDecide)
	respectLearning := rules.Get(RuleRespectLearningPhase) >= 0.5

	campaignAge := eval.CampaignAgeHours
	if campaignAge == 0 {
		campaignAge = eval.AdAgeHours
	}
	if campaignAge < minAge {
		return nil // ainda em fase inicial — não interferir
	}
	if respectLearning && eval.CampaignLeads7d > 0 && float64(eval.CampaignLeads7d) < minConv {
		return nil // ainda em aprendizagem — Meta exige ~50 eventos/7d
	}

	// No leads metric => no signal => skip.
	if eval.AdLeads7d <= 0 || eval.AccountAvgCPL <= 0 {
		// Still allow CTR/freq alerts even without lead data.
		return evaluateNonLeadAlerts(eval, rules)
	}

	cpl := eval.AdSpend7d / float64(eval.AdLeads7d)

	// Winner: CPL muito abaixo da média + CTR alto + freq sob controle => sugere escalar verba.
	if scale := evaluateScaleUp(eval, rules, cpl); scale != nil {
		return scale
	}

	// Hard pause: too expensive AND minimum spend AND minimum age.
	pauseRatio := rules.Get(RulePauseCPLRatio)
	minSpend := rules.Get(RuleMinSpendToPause)

	if cpl >= pauseRatio*eval.AccountAvgCPL &&
		eval.AdSpend7d >= minSpend &&
		eval.AdAgeHours >= minAge {

		snap := metricSnapshot(eval, cpl)
		change := proposedChange("status", "ACTIVE", "PAUSED")
		return &domain.AIAction{
			UserID:         eval.UserID,
			AccountMetaID:  eval.AccountMetaID,
			ActionType:     domain.AIActionTypePauseAd,
			TargetMetaID:   eval.Ad.MetaAdID,
			TargetKind:     domain.AIActionTargetAd,
			Reason: fmt.Sprintf(
				"CPL R$ %.2f está %.1fx acima da média da conta (R$ %.2f). Spend R$ %.2f em 7d.",
				cpl, cpl/eval.AccountAvgCPL, eval.AccountAvgCPL, eval.AdSpend7d,
			),
			MetricSnapshot: snap,
			ProposedChange: change,
			Source:         domain.AIActionSourceRules,
			Mode:           domain.AIActionModeAuto,
			Status:         domain.AIActionStatusPending,
		}
	}

	// Soft alert: less than 3x but > alert ratio — propose for review.
	alertRatio := rules.Get(RuleAlertCPLRatio)
	if cpl >= alertRatio*eval.AccountAvgCPL && eval.AdSpend7d >= minSpend {
		snap := metricSnapshot(eval, cpl)
		return &domain.AIAction{
			UserID:        eval.UserID,
			AccountMetaID: eval.AccountMetaID,
			ActionType:    domain.AIActionTypeAlert,
			TargetMetaID:  eval.Ad.MetaAdID,
			TargetKind:    domain.AIActionTargetAd,
			Reason: fmt.Sprintf(
				"CPL R$ %.2f está %.1fx acima da média (R$ %.2f). Avalie revisar criativo ou segmentação.",
				cpl, cpl/eval.AccountAvgCPL, eval.AccountAvgCPL,
			),
			MetricSnapshot: snap,
			Source:         domain.AIActionSourceRules,
			Mode:           domain.AIActionModePropose,
			Status:         domain.AIActionStatusPending,
		}
	}

	if alert := evaluateNonLeadAlerts(eval, rules); alert != nil {
		return alert
	}
	return nil
}

// evaluateScaleUp proposes a budget increase for clear winners.
// Conditions: CPL <= scale_cpl_ratio * account avg, CTR >= scale_min_ctr,
// freq <= scale_max_freq, has an adset with a daily budget set.
func evaluateScaleUp(eval *AdEvaluation, rules *EffectiveRules, cpl float64) *domain.AIAction {
	if eval.AdSetMetaID == "" || eval.AdSetDailyBudget <= 0 {
		return nil
	}
	scaleRatio := rules.Get(RuleScaleCPLRatio)
	minCTR := rules.Get(RuleScaleMinCTR)
	maxFreq := rules.Get(RuleScaleMaxFreq)
	factor := rules.Get(RuleScaleFactor)
	if factor < 1.05 || factor > 2.0 {
		factor = 1.2
	}

	if cpl > scaleRatio*eval.AccountAvgCPL {
		return nil
	}
	if eval.AdCTR7d > 0 && eval.AdCTR7d < minCTR {
		return nil
	}
	if eval.AdFreq7d > maxFreq {
		return nil
	}

	from := eval.AdSetDailyBudget
	to := from * factor
	change := proposedChange("daily_budget", from, to)
	snap := metricSnapshot(eval, cpl)
	return &domain.AIAction{
		UserID:        eval.UserID,
		AccountMetaID: eval.AccountMetaID,
		ActionType:    domain.AIActionTypeScaleBudget,
		TargetMetaID:  eval.AdSetMetaID,
		TargetKind:    domain.AIActionTargetAdSet,
		Reason: fmt.Sprintf(
			"Vencedor maduro: custo R$ %.2f (%.0f%% da média R$ %.2f), CTR %.2f%%, freq %.2f. Sugiro escalar +%.0f%% (R$ %.2f → R$ %.2f).",
			cpl, (cpl/eval.AccountAvgCPL)*100, eval.AccountAvgCPL,
			eval.AdCTR7d*100, eval.AdFreq7d, (factor-1)*100, from, to,
		),
		MetricSnapshot: snap,
		ProposedChange: change,
		Source:         domain.AIActionSourceRules,
		Mode:           domain.AIActionModePropose,
		Status:         domain.AIActionStatusPending,
	}
}

func evaluateNonLeadAlerts(eval *AdEvaluation, rules *EffectiveRules) *domain.AIAction {
	ctrMin := rules.Get(RuleAlertCTRMin)
	freqMax := rules.Get(RuleAlertFreqMax)

	if eval.AdSpend7d < rules.Get(RuleMinSpendToPause) {
		return nil
	}

	if eval.AdCTR7d > 0 && eval.AdCTR7d < ctrMin {
		snap := metricSnapshot(eval, 0)
		return &domain.AIAction{
			UserID:        eval.UserID,
			AccountMetaID: eval.AccountMetaID,
			ActionType:    domain.AIActionTypeAlert,
			TargetMetaID:  eval.Ad.MetaAdID,
			TargetKind:    domain.AIActionTargetAd,
			Reason: fmt.Sprintf(
				"CTR %.2f%% abaixo do mínimo (%.2f%%). Spend R$ %.2f. Avalie criativo.",
				eval.AdCTR7d*100, ctrMin*100, eval.AdSpend7d,
			),
			MetricSnapshot: snap,
			Source:         domain.AIActionSourceRules,
			Mode:           domain.AIActionModePropose,
			Status:         domain.AIActionStatusPending,
		}
	}
	if eval.AdFreq7d >= freqMax {
		snap := metricSnapshot(eval, 0)
		return &domain.AIAction{
			UserID:        eval.UserID,
			AccountMetaID: eval.AccountMetaID,
			ActionType:    domain.AIActionTypeAlert,
			TargetMetaID:  eval.Ad.MetaAdID,
			TargetKind:    domain.AIActionTargetAd,
			Reason: fmt.Sprintf(
				"Frequência %.2f acima do limite (%.2f). Audiência saturada.",
				eval.AdFreq7d, freqMax,
			),
			MetricSnapshot: snap,
			Source:         domain.AIActionSourceRules,
			Mode:           domain.AIActionModePropose,
			Status:         domain.AIActionStatusPending,
		}
	}
	return nil
}

func metricSnapshot(eval *AdEvaluation, cpl float64) json.RawMessage {
	m := map[string]any{
		"cpl":             cpl,
		"account_avg_cpl": eval.AccountAvgCPL,
		"spend_7d":        eval.AdSpend7d,
		"leads_7d":        eval.AdLeads7d,
		"ctr_7d":          eval.AdCTR7d,
		"freq_7d":         eval.AdFreq7d,
		"age_hours":       eval.AdAgeHours,
		"snapshot_at":     time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.Marshal(m)
	return b
}

func proposedChange(field string, from, to any) json.RawMessage {
	m := map[string]any{
		"field": field,
		"from":  from,
		"to":    to,
	}
	b, _ := json.Marshal(m)
	return b
}

// SafetyRulesService bundles the repos so handlers can list / update overrides.
type SafetyRulesService struct {
	rules repository.AISafetyRuleRepository
}

func NewSafetyRulesService(rules repository.AISafetyRuleRepository) *SafetyRulesService {
	return &SafetyRulesService{rules: rules}
}

// EffectiveForUser returns defaults + the user's overrides, indexed by rule_key.
func (s *SafetyRulesService) EffectiveForUser(ctx context.Context, userID string) (map[string]float64, []*domain.AISafetyRule, error) {
	overrides, err := s.rules.ListByUser(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	out := make(map[string]float64, len(DefaultSafetyRules))
	for k, v := range DefaultSafetyRules {
		out[k] = v
	}
	for _, r := range overrides {
		if r.AccountMetaID == nil {
			out[r.RuleKey] = r.RuleValue
		}
	}
	return out, overrides, nil
}

// Upsert validates the rule_key and persists the override.
func (s *SafetyRulesService) Upsert(ctx context.Context, userID, ruleKey string, value float64, accountMetaID *string) error {
	if _, ok := DefaultSafetyRules[ruleKey]; !ok {
		return fmt.Errorf("%w: unknown rule_key %q", domain.ErrValidation, ruleKey)
	}
	rule := &domain.AISafetyRule{
		UserID:        userID,
		AccountMetaID: accountMetaID,
		RuleKey:       ruleKey,
		RuleValue:     value,
		Enabled:       true,
	}
	return s.rules.Upsert(ctx, rule)
}
