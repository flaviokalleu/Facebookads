package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"
)

// routingTable maps each task to an ordered list of providers to try.
// Primary → Fallback 1 → Fallback 2
var routingTable = map[TaskType][]string{
	TaskClassification:   {"deepseek-v4-pro", "glm-5", "gemini-2-5-flash"},
	TaskAnomalyDetection: {"gemini-2-5-pro", "claude-opus-4-7", "gpt-5-4"},
	TaskOptimization: {"deepseek-v4-pro", "deepseek-r2", "gemini-2-5-pro"},
	TaskBudgetAdvisor: {"deepseek-r2", "deepseek-v4-pro", "gemini-2-5-pro"},
	TaskCreativeAnalysis: {"deepseek-v4-pro", "deepseek-r2", "gemini-2-5-flash"},
	TaskSummary:          {"glm-5", "gemini-2-5-flash", "gpt-4o-mini"},
}

// Router selects the best available provider for a given task and falls back gracefully.
type Router struct {
	providers map[string]Provider // key = provider name (e.g. "claude-opus-4-7")
}

func NewRouter(providers []Provider) *Router {
	m := make(map[string]Provider, len(providers))
	for _, p := range providers {
		m[p.ModelID()] = p
	}
	return &Router{providers: m}
}

// Complete routes the request to the best available provider for the task.
func (r *Router) Complete(ctx context.Context, task TaskType, req CompletionRequest) (CompletionResponse, error) {
	chain, ok := routingTable[task]
	if !ok {
		return CompletionResponse{}, fmt.Errorf("unknown task type: %s", task)
	}

	var lastErr error
	for _, modelID := range chain {
		p, exists := r.providers[modelID]
		if !exists {
			slog.Debug("router: provider not configured", "model", modelID, "task", task)
			continue
		}
		if !CheckAvailable(ctx, p) {
			slog.Warn("router: provider unavailable", "model", modelID, "task", task)
			continue
		}

		start := time.Now()
		resp, err := p.Complete(ctx, req)
		if err != nil {
			slog.Error("router: provider failed", "model", modelID, "task", task, "err", err)
			lastErr = err
			continue
		}
		resp.LatencyMs = time.Since(start).Milliseconds()
		slog.Info("router: task completed",
			"task", task,
			"provider", resp.Provider,
			"model", resp.ModelUsed,
			"latency_ms", resp.LatencyMs,
			"cost_usd", resp.CostUSD,
		)
		return resp, nil
	}

	if lastErr != nil {
		return CompletionResponse{}, fmt.Errorf("all providers failed for task %s: %w", task, lastErr)
	}
	return CompletionResponse{}, fmt.Errorf("no providers configured for task %s", task)
}

// OverrideRouting allows admin to change the routing table at runtime (from DB config).
func (r *Router) OverrideRouting(task TaskType, modelIDs []string) {
	routingTable[task] = modelIDs
}

// RoutingTable returns the current routing table (for admin API).
func (r *Router) RoutingTable() map[TaskType][]string {
	result := make(map[TaskType][]string, len(routingTable))
	for k, v := range routingTable {
		cp := make([]string, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}

// ActiveProviders returns all configured provider model IDs.
func (r *Router) ActiveProviders() []string {
	names := make([]string, 0, len(r.providers))
	for k := range r.providers {
		names = append(names, k)
	}
	return names
}

// ProviderInfo is a summary of a provider returned to the admin UI.
type ProviderInfo struct {
	Name             string  `json:"name"`
	ModelID          string  `json:"model_id"`
	Available        bool    `json:"available"`
	CostPer1MInput   float64 `json:"cost_per_1m_input,omitempty"`
	CostPer1MOutput  float64 `json:"cost_per_1m_output,omitempty"`
}

// ProviderInfos returns status info for all configured providers.
func (r *Router) ProviderInfos() []ProviderInfo {
	infos := make([]ProviderInfo, 0, len(r.providers))
	for _, p := range r.providers {
		cost := ModelCosts[p.ModelID()]
		infos = append(infos, ProviderInfo{
			Name:            p.Name(),
			ModelID:         p.ModelID(),
			Available:       p.IsAvailable(context.Background()),
			CostPer1MInput:  cost.InputPer1M,
			CostPer1MOutput: cost.OutputPer1M,
		})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name < infos[j].Name })
	return infos
}
