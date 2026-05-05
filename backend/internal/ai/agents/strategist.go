package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/ai/prompts"
	"github.com/facebookads/backend/internal/ai/providers"
	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/repository"
)

// Strategist is the DeepSeek-driven daily strategic-plan agent. It is pinned
// to deepseek-reasoner — DeepSeek is ~10x cheaper than Anthropic for the kind
// of long, structured reasoning we want here.
type Strategist struct {
	db       *pgxpool.Pool
	provider ai.Provider
	tokens   repository.MetaTokenRepository
	accounts repository.MetaAdAccountRepository
	actions  repository.AIActionRepository
	llmUsage repository.LLMUsageRepository
}

func NewStrategist(
	db *pgxpool.Pool,
	deepseekAPIKey string,
	tokens repository.MetaTokenRepository,
	accounts repository.MetaAdAccountRepository,
	actions repository.AIActionRepository,
	llmUsage repository.LLMUsageRepository,
) *Strategist {
	prov := providers.NewDeepSeek(deepseekAPIKey, "deepseek-reasoner")
	return &Strategist{
		db: db, provider: prov, tokens: tokens, accounts: accounts,
		actions: actions, llmUsage: llmUsage,
	}
}

// strategistSystemPrompt agora vive em internal/ai/prompts/imobiliario.go
// (StrategistSystemPrompt) com benchmarks de mercado e playbook detalhado.
var strategistSystemPrompt = prompts.StrategistSystemPrompt

type strategistToolCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type strategistResponse struct {
	ToolCalls []strategistToolCall `json:"tool_calls"`
}

// Run iterates every user with an active token and analyzes each of their
// active accounts (where signal is sufficient).
func (s *Strategist) Run(ctx context.Context) error {
	if s.provider == nil || !s.provider.IsAvailable(ctx) {
		slog.Warn("strategist: deepseek provider not available — skipping")
		return nil
	}

	rows, err := s.db.Query(ctx, `SELECT DISTINCT user_id::text FROM meta_tokens WHERE is_active = true`)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	var users []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		users = append(users, id)
	}
	rows.Close()

	for _, userID := range users {
		if err := s.runForUser(ctx, userID); err != nil {
			slog.Error("strategist: user failed", "user_id", userID, "err", err)
		}
	}
	return nil
}

func (s *Strategist) runForUser(ctx context.Context, userID string) error {
	accs, err := s.accounts.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list accounts: %w", err)
	}
	for _, acc := range accs {
		if acc.AccountStatus != 1 {
			continue
		}
		if err := s.runForAccount(ctx, userID, acc); err != nil {
			slog.Warn("strategist: account failed",
				"user_id", userID, "account", acc.MetaID, "err", err)
		}
	}
	return nil
}

type campaignSnapshot struct {
	CampaignID    string  `json:"campaign_id"`
	MetaID        string  `json:"meta_campaign_id"`
	Name          string  `json:"name"`
	Objective     string  `json:"objective"`
	Spend7d       float64 `json:"spend_7d"`
	Leads7d       int64   `json:"leads_7d"`
	CPL           float64 `json:"cpl"`
	AvgCTR        float64 `json:"avg_ctr"`
	AvgFreq       float64 `json:"avg_freq"`
	ActiveAds     int     `json:"active_ads"`
	HealthStatus  string  `json:"health_status"`
}

type adsetSnapshot struct {
	AdSetID       string  `json:"adset_id"`
	MetaID        string  `json:"meta_adset_id"`
	CampaignID    string  `json:"campaign_id"`
	Name          string  `json:"name"`
	Status        string  `json:"status"`
	DailyBudget   float64 `json:"daily_budget"`
	Spend7d       float64 `json:"spend_7d_parent"`
	Leads7d       int64   `json:"leads_7d_parent"`
}

func (s *Strategist) runForAccount(ctx context.Context, userID string, acc *domain.MetaAdAccount) error {
	// Cost cap: <R$10 spent in 7d OR <3 active ads => skip.
	var spent7d float64
	var activeAds int
	if err := s.db.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(ci.spend),0)::float8,
		  (SELECT COUNT(*) FROM ads ad
		     JOIN ad_sets aset ON aset.id = ad.ad_set_id
		     JOIN campaigns c2 ON c2.id = aset.campaign_id
		     WHERE c2.user_id = $1 AND c2.ad_account_id = $2
		       AND ad.status = 'ACTIVE' AND ad.deleted_at IS NULL)::int
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id = $1 AND c.ad_account_id = $2
		  AND ci.date >= CURRENT_DATE - 7
	`, userID, acc.MetaID).Scan(&spent7d, &activeAds); err != nil {
		return fmt.Errorf("signal check: %w", err)
	}
	if spent7d < 10 || activeAds < 3 {
		slog.Info("strategist: account below cost cap, skipping",
			"user_id", userID, "account", acc.MetaID, "spend_7d", spent7d, "active_ads", activeAds)
		return nil
	}

	camps, err := s.campaignSnapshots(ctx, userID, acc.MetaID)
	if err != nil {
		return fmt.Errorf("campaign snapshots: %w", err)
	}
	adsets, err := s.adsetSnapshots(ctx, userID, acc.MetaID)
	if err != nil {
		return fmt.Errorf("adset snapshots: %w", err)
	}

	prompt := buildStrategistPrompt(acc, camps, adsets)

	start := time.Now()
	resp, err := s.provider.Complete(ctx, ai.CompletionRequest{
		SystemPrompt: strategistSystemPrompt,
		UserPrompt:   prompt,
		MaxTokens:    3000,
		Temperature:  0.2,
		JSONMode:     true,
	})
	resp.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		s.logUsage(ctx, userID, resp, err)
		return fmt.Errorf("deepseek: %w", err)
	}
	s.logUsage(ctx, userID, resp, nil)

	cleaned := cleanJSON(resp.Content)
	var parsed strategistResponse
	if jerr := json.Unmarshal([]byte(cleaned), &parsed); jerr != nil {
		return fmt.Errorf("parse response: %w (raw: %s)", jerr, truncate(cleaned, 300))
	}

	persisted := 0
	for _, tc := range parsed.ToolCalls {
		action := s.toolCallToAction(userID, acc.MetaID, tc, camps, adsets)
		if action == nil {
			continue
		}
		if err := s.actions.Create(ctx, action); err != nil {
			slog.Warn("strategist: insert action failed", "err", err)
			continue
		}
		persisted++
	}
	slog.Info("strategist: ran for account",
		"user_id", userID, "account", acc.MetaID,
		"tool_calls", len(parsed.ToolCalls), "persisted", persisted,
		"input_tokens", resp.InputTokens, "output_tokens", resp.OutputTokens)
	return nil
}

func (s *Strategist) campaignSnapshots(ctx context.Context, userID, accountMetaID string) ([]campaignSnapshot, error) {
	rows, err := s.db.Query(ctx, `
		SELECT c.id, c.meta_campaign_id, c.name, COALESCE(c.objective,''), c.health_status,
		       COALESCE(s.spend,0)::float8, COALESCE(s.leads,0)::bigint,
		       COALESCE(s.ctr,0)::float8, COALESCE(s.freq,0)::float8,
		       COALESCE(ad_count.n,0)::int
		FROM campaigns c
		LEFT JOIN LATERAL (
		  SELECT SUM(spend) AS spend, SUM(leads) AS leads,
		         AVG(ctr) AS ctr, AVG(frequency) AS freq
		  FROM campaign_insights WHERE campaign_id = c.id AND date >= CURRENT_DATE - 7
		) s ON true
		LEFT JOIN LATERAL (
		  SELECT COUNT(*) AS n FROM ads ad
		    JOIN ad_sets ase ON ase.id = ad.ad_set_id
		    WHERE ase.campaign_id = c.id AND ad.status = 'ACTIVE' AND ad.deleted_at IS NULL
		) ad_count ON true
		WHERE c.user_id = $1 AND c.ad_account_id = $2 AND c.deleted_at IS NULL
		ORDER BY s.spend DESC NULLS LAST
		LIMIT 30
	`, userID, accountMetaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []campaignSnapshot
	for rows.Next() {
		var c campaignSnapshot
		if err := rows.Scan(&c.CampaignID, &c.MetaID, &c.Name, &c.Objective, &c.HealthStatus,
			&c.Spend7d, &c.Leads7d, &c.AvgCTR, &c.AvgFreq, &c.ActiveAds); err != nil {
			return nil, err
		}
		if c.Leads7d > 0 {
			c.CPL = c.Spend7d / float64(c.Leads7d)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Strategist) adsetSnapshots(ctx context.Context, userID, accountMetaID string) ([]adsetSnapshot, error) {
	rows, err := s.db.Query(ctx, `
		SELECT aset.id, aset.meta_ad_set_id, aset.campaign_id, aset.name, aset.status,
		       COALESCE(aset.daily_budget,0)::float8,
		       COALESCE(s.spend,0)::float8, COALESCE(s.leads,0)::bigint
		FROM ad_sets aset
		JOIN campaigns c ON c.id = aset.campaign_id
		LEFT JOIN LATERAL (
		  SELECT SUM(spend) AS spend, SUM(leads) AS leads
		  FROM campaign_insights WHERE campaign_id = c.id AND date >= CURRENT_DATE - 7
		) s ON true
		WHERE c.user_id = $1 AND c.ad_account_id = $2
		  AND aset.deleted_at IS NULL AND c.deleted_at IS NULL
		ORDER BY s.spend DESC NULLS LAST
		LIMIT 80
	`, userID, accountMetaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []adsetSnapshot
	for rows.Next() {
		var a adsetSnapshot
		if err := rows.Scan(&a.AdSetID, &a.MetaID, &a.CampaignID, &a.Name, &a.Status,
			&a.DailyBudget, &a.Spend7d, &a.Leads7d); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func buildStrategistPrompt(acc *domain.MetaAdAccount, camps []campaignSnapshot, adsets []adsetSnapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Conta %s — saldo R$ %.2f — moeda %s — última semana:\n",
		acc.MetaID, acc.Balance/100, acc.Currency)
	fmt.Fprintf(&b, "\n=== CAMPANHAS ===\n")
	for _, c := range camps {
		fmt.Fprintf(&b,
			"- meta_id=%s  nome=%q  objetivo=%s  spend=R$%.2f  leads=%d  CPL=R$%.2f  CTR=%.2f%%  freq=%.2f  ads_ativos=%d  health=%s\n",
			c.MetaID, c.Name, c.Objective, c.Spend7d, c.Leads7d, c.CPL, c.AvgCTR*100, c.AvgFreq, c.ActiveAds, c.HealthStatus)
	}
	fmt.Fprintf(&b, "\n=== AD SETS ===\n")
	for _, a := range adsets {
		fmt.Fprintf(&b,
			"- meta_id=%s  nome=%q  status=%s  daily_budget=R$%.2f  spend_campanha=R$%.2f  leads_campanha=%d\n",
			a.MetaID, a.Name, a.Status, a.DailyBudget, a.Spend7d, a.Leads7d)
	}
	fmt.Fprintf(&b, "\nGere o plano estratégico em JSON conforme o formato.")
	return b.String()
}

func (s *Strategist) toolCallToAction(userID, accountMetaID string, tc strategistToolCall, _ []campaignSnapshot, _ []adsetSnapshot) *domain.AIAction {
	args := tc.Args
	getStr := func(k string) string { v, _ := args[k].(string); return v }
	getNum := func(k string) float64 {
		switch v := args[k].(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case json.Number:
			f, _ := v.Float64()
			return f
		}
		return 0
	}

	base := func(actionType, targetID, targetKind, reason string) *domain.AIAction {
		snap, _ := json.Marshal(map[string]any{"raw_args": args})
		change, _ := json.Marshal(args)
		return &domain.AIAction{
			UserID:         userID,
			AccountMetaID:  accountMetaID,
			ActionType:     actionType,
			TargetMetaID:   targetID,
			TargetKind:     targetKind,
			Reason:         reason,
			MetricSnapshot: snap,
			ProposedChange: change,
			Source:         domain.AIActionSourceDeepSeek,
			Mode:           domain.AIActionModePropose,
			Status:         domain.AIActionStatusPending,
		}
	}

	switch tc.Name {
	case "pause_adset":
		id := getStr("adset_id")
		if id == "" {
			return nil
		}
		return base(domain.AIActionTypePauseAdSet, id, domain.AIActionTargetAdSet, getStr("reason"))
	case "scale_budget":
		id := getStr("adset_id")
		factor := getNum("factor")
		if id == "" || factor <= 0 {
			return nil
		}
		if factor < 0.5 {
			factor = 0.5
		}
		if factor > 2.0 {
			factor = 2.0
		}
		args["factor"] = factor
		return base(domain.AIActionTypeScaleBudget, id, domain.AIActionTargetAdSet, getStr("reason"))
	case "duplicate_adset":
		id := getStr("adset_id")
		if id == "" {
			return nil
		}
		return base(domain.AIActionTypeDuplicateAdSet, id, domain.AIActionTargetAdSet, getStr("reason"))
	case "rotate_creative":
		id := getStr("ad_id")
		if id == "" {
			return nil
		}
		return base(domain.AIActionTypeRotateCreative, id, domain.AIActionTargetAd, getStr("reason"))
	case "alert":
		id := getStr("target_id")
		kind := getStr("target_kind")
		if id == "" {
			return nil
		}
		if kind == "" {
			kind = domain.AIActionTargetAd
		}
		return base(domain.AIActionTypeAlert, id, kind, getStr("reason"))
	case "propose_only":
		summary := getStr("plan_summary")
		if summary == "" {
			return nil
		}
		return base(domain.AIActionTypeAlert, accountMetaID, domain.AIActionTargetCampaign, summary)
	}
	return nil
}

func (s *Strategist) logUsage(ctx context.Context, userID string, resp ai.CompletionResponse, err error) {
	if s.llmUsage == nil {
		return
	}
	uid := userID
	usage := &domain.LLMUsage{
		UserID:       &uid,
		TaskType:     "strategist",
		Provider:     resp.Provider,
		Model:        resp.ModelUsed,
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
		CostUSD:      resp.CostUSD,
		LatencyMs:    int(resp.LatencyMs),
		Success:      err == nil,
	}
	if err != nil {
		usage.ErrorMessage = err.Error()
	}
	if usage.Provider == "" {
		usage.Provider = "deepseek"
	}
	if usage.Model == "" {
		usage.Model = "deepseek-reasoner"
	}
	if logErr := s.llmUsage.Create(ctx, usage); logErr != nil {
		slog.Warn("strategist: log usage failed", "err", logErr)
	}
}
