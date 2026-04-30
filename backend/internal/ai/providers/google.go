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

type GoogleProvider struct {
	apiKey  string
	modelID string
	http    *http.Client
}

func NewGoogle(apiKey, modelID string) *GoogleProvider {
	if modelID == "" {
		modelID = "gemini-2.5-pro"
	}
	return &GoogleProvider{
		apiKey:  apiKey,
		modelID: modelID,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *GoogleProvider) Name() string    { return "google" }
func (p *GoogleProvider) ModelID() string { return p.modelID }

func (p *GoogleProvider) IsAvailable(ctx context.Context) bool {
	return p.apiKey != ""
}

func (p *GoogleProvider) Complete(ctx context.Context, req ai.CompletionRequest) (ai.CompletionResponse, error) {
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}

	parts := []map[string]string{{"text": req.SystemPrompt + "\n\n" + req.UserPrompt}}

	body := map[string]any{
		"contents": []map[string]any{
			{"role": "user", "parts": parts},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": req.MaxTokens,
			"temperature":     req.Temperature,
		},
	}
	if req.JSONMode {
		body["generationConfig"].(map[string]any)["responseMimeType"] = "application/json"
	}

	data, err := json.Marshal(body)
	if err != nil {
		return ai.CompletionResponse{}, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		p.modelID, p.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return ai.CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.http.Do(httpReq)
	if err != nil {
		return ai.CompletionResponse{}, fmt.Errorf("google: request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return ai.CompletionResponse{}, fmt.Errorf("google: status %d: %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return ai.CompletionResponse{}, fmt.Errorf("google: parse response: %w", err)
	}

	text := ""
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		text = result.Candidates[0].Content.Parts[0].Text
	}

	in := result.UsageMetadata.PromptTokenCount
	out := result.UsageMetadata.CandidatesTokenCount

	return ai.CompletionResponse{
		Content:      text,
		InputTokens:  in,
		OutputTokens: out,
		ModelUsed:    p.modelID,
		Provider:     "google",
		CostUSD:      ai.CalcCost(p.modelID, in, out),
	}, nil
}
