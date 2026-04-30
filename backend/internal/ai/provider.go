package ai

import (
	"context"
	"time"
)

// TaskType identifies what kind of AI work is being done,
// so the router can pick the best model for each job.
type TaskType string

const (
	TaskClassification   TaskType = "classification"
	TaskAnomalyDetection TaskType = "anomaly_detection"
	TaskOptimization     TaskType = "optimization"
	TaskBudgetAdvisor    TaskType = "budget_advisor"
	TaskCreativeAnalysis TaskType = "creative_analysis"
	TaskSummary          TaskType = "summary"
)

// CompletionRequest is the unified input for all providers.
type CompletionRequest struct {
	SystemPrompt string
	UserPrompt   string
	MaxTokens    int
	Temperature  float64
	JSONMode     bool
}

// CompletionResponse is the unified output from all providers.
type CompletionResponse struct {
	Content      string
	InputTokens  int
	OutputTokens int
	LatencyMs    int64
	ModelUsed    string
	Provider     string
	CostUSD      float64
}

// Provider is the interface every LLM adapter must implement.
type Provider interface {
	Name() string
	ModelID() string
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	IsAvailable(ctx context.Context) bool
}

// ─── Cost constants (per 1M tokens, USD) ─────────────────────────────────────

type ModelCost struct {
	InputPer1M  float64
	OutputPer1M float64
}

var ModelCosts = map[string]ModelCost{
	"claude-opus-4-7":     {15.00, 75.00},
	"claude-sonnet-4-6":   {3.00, 15.00},
	"gpt-5-4":             {10.00, 30.00},
	"gpt-4o-mini":         {0.15, 0.60},
	"gemini-2-5-pro":      {7.00, 21.00},
	"gemini-2-5-flash":    {0.15, 0.60},
	"deepseek-v4-pro":     {0.27, 1.10},
	"deepseek-r2":         {0.55, 2.20},
	"glm-5-reasoning":     {0.20, 0.80},
	"glm-5":               {0.10, 0.40},
	"grok-4":              {8.00, 24.00},
	"kimi-2-6":            {1.00, 3.00},
	"qwen3-5-235b":        {0.20, 0.60},
}

func CalcCost(model string, inputTokens, outputTokens int) float64 {
	cost, ok := ModelCosts[model]
	if !ok {
		return 0
	}
	return (float64(inputTokens)/1_000_000)*cost.InputPer1M +
		(float64(outputTokens)/1_000_000)*cost.OutputPer1M
}

// ─── Availability check helper ────────────────────────────────────────────────

func CheckAvailable(ctx context.Context, p Provider) bool {
	ctx2, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return p.IsAvailable(ctx2)
}
