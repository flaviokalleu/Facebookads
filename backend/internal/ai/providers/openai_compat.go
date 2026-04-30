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

// OpenAICompatProvider works for any OpenAI-compatible API:
// OpenAI, DeepSeek, Zhipu GLM, Moonshot Kimi, Alibaba Qwen, xAI Grok
type OpenAICompatProvider struct {
	apiKey   string
	modelID  string
	name     string
	baseURL  string
	http     *http.Client
}

func NewOpenAI(apiKey, modelID string) *OpenAICompatProvider {
	return newCompat(apiKey, modelID, "openai", "https://api.openai.com/v1")
}

func NewDeepSeek(apiKey, modelID string) *OpenAICompatProvider {
	return newCompat(apiKey, modelID, "deepseek", "https://api.deepseek.com/v1")
}

func NewZhipu(apiKey, modelID string) *OpenAICompatProvider {
	return newCompat(apiKey, modelID, "zhipu", "https://open.bigmodel.cn/api/paas/v4")
}

func NewMoonshot(apiKey, modelID string) *OpenAICompatProvider {
	return newCompat(apiKey, modelID, "moonshot", "https://api.moonshot.cn/v1")
}

func NewAlibaba(apiKey, modelID string) *OpenAICompatProvider {
	return newCompat(apiKey, modelID, "alibaba", "https://dashscope.aliyuncs.com/compatible-mode/v1")
}

func NewXAI(apiKey, modelID string) *OpenAICompatProvider {
	return newCompat(apiKey, modelID, "xai", "https://api.x.ai/v1")
}

func newCompat(apiKey, modelID, name, baseURL string) *OpenAICompatProvider {
	return &OpenAICompatProvider{
		apiKey:  apiKey,
		modelID: modelID,
		name:    name,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAICompatProvider) Name() string    { return p.name }
func (p *OpenAICompatProvider) ModelID() string { return p.modelID }

func (p *OpenAICompatProvider) IsAvailable(ctx context.Context) bool {
	return p.apiKey != ""
}

func (p *OpenAICompatProvider) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	if req.Temperature == 0 {
		req.Temperature = 0.3
	}

	messages := []map[string]string{}
	if req.SystemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.SystemPrompt})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.UserPrompt})

	body := map[string]any{
		"model":       p.modelID,
		"messages":    messages,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}
	if req.JSONMode {
		body["response_format"] = map[string]string{"type": "json_object"}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return ai.CompletionResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return ai.CompletionResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return ai.CompletionResponse{}, fmt.Errorf("%s: request failed: %w", p.name, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ai.CompletionResponse{}, fmt.Errorf("%s: status %d: %s", p.name, resp.StatusCode, string(raw))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return ai.CompletionResponse{}, fmt.Errorf("%s: parse response: %w", p.name, err)
	}

	text := ""
	if len(result.Choices) > 0 {
		text = result.Choices[0].Message.Content
	}

	return ai.CompletionResponse{
		Content:      text,
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
		ModelUsed:    p.modelID,
		Provider:     p.name,
		CostUSD:      ai.CalcCost(p.modelID, result.Usage.PromptTokens, result.Usage.CompletionTokens),
	}, nil
}
