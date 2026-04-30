package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/facebookads/backend/internal/ai"
)

type AnthropicProvider struct {
	apiKey  string
	modelID string
	http    *http.Client
}

func NewAnthropic(apiKey, modelID string) *AnthropicProvider {
	if modelID == "" {
		modelID = "claude-opus-4-7"
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		modelID: modelID,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *AnthropicProvider) Name() string    { return "anthropic" }
func (p *AnthropicProvider) ModelID() string { return p.modelID }

func (p *AnthropicProvider) IsAvailable(ctx context.Context) bool {
	return p.apiKey != ""
}

func (p *AnthropicProvider) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	if req.Temperature == 0 {
		req.Temperature = 0.3
	}

	body := map[string]any{
		"model":      p.modelID,
		"max_tokens": req.MaxTokens,
		"system":     req.SystemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": req.UserPrompt},
		},
	}

	// Use prompt caching for system prompt (reduces cost up to 90%)
	if len(req.SystemPrompt) > 1024 {
		body["system"] = []map[string]any{
			{
				"type": "text",
				"text": req.SystemPrompt,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return ai.CompletionResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(data))
	if err != nil {
		return ai.CompletionResponse{}, err
	}
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return ai.CompletionResponse{}, fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return ai.CompletionResponse{}, fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return ai.CompletionResponse{}, fmt.Errorf("anthropic: parse response: %w", err)
	}

	text := ""
	if len(result.Content) > 0 {
		text = result.Content[0].Text
	}

	return ai.CompletionResponse{
		Content:      text,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		ModelUsed:    p.modelID,
		Provider:     "anthropic",
		CostUSD:      ai.CalcCost(p.modelID, result.Usage.InputTokens, result.Usage.OutputTokens),
	}, nil
}
