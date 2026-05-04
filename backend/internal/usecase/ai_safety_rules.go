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
	RulePauseCPLRatio     = "pause_cpl_ratio"
	RuleMinSpendToPause   = "min_spend_to_pause"
	RuleMinAgeHours       = "min_age_hours"
	RuleMaxPausePctPerDay = "max_pause_pct_per_day"
	RuleAlertCPLRatio     = "alert_cpl_ratio"
	RuleAlertCTRMin       = "alert_ctr_min"
	RuleAlertFreqMax      = "alert_freq_max"
)

// DefaultSafetyRules holds the hardcoded baseline thresholds. Per-user overrides
// from the ai_safety_rules table win when present.
var DefaultSafetyRules = map[string]float64{
	RulePauseCPLRatio:     3.0,
	RuleMinSpendToPause:   30.0,
	RuleMinAgeHours:       24.0,
	RuleMaxPausePctPerDay: 0.30,
	RuleAlertCPLRatio:     1.5,
	RuleAlertCTRMin:       0.01,
	RuleAlertFreqMax:      3.5,
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
	Ad             *domain.Ad
	Campaign       *domain.Campaign
	AccountMetaID  string
	UserID         string
	AdSpend7d      float64
	AdLeads7d      int64
	AdCTR7d        float64
	AdFreq7d       float64
	AccountAvgCPL  float64
	AdAgeHours     float64
}

// EvaluateAd applies the deterministic safety rules to one ad. Returns nil
// when the ad does not warrant any action. The returned AIAction is fully
// populated except for ID/CreatedAt — caller persists via the repository.
func EvaluateAd(eval *AdEvaluation, rules *EffectiveRules) *domain.AIAction {
	// No leads metric => no signal => skip.
	if eval.AdLeads7d <= 0 || eval.AccountAvgCPL <= 0 {
		// Still allow CTR/freq alerts even without lead data.
		return evaluateNonLeadAlerts(eval, rules)
	}

	cpl := eval.AdSpend7d / float64(eval.AdLeads7d)

	// Hard pause: too expensive AND minimum spend AND minimum age.
	pauseRatio := rules.Get(RulePauseCPLRatio)
	minSpend := rules.Get(RuleMinSpendToPause)
	minAge := rules.Get(RuleMinAgeHours)

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
