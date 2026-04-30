package metaads

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"bytes"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// v25.0 is the latest stable Meta Marketing API version (Feb 2025+)
const defaultBaseURL = "https://graph.facebook.com"

// Client defines the Meta Marketing API contract.
type Client interface {
	GetAdAccounts(ctx context.Context, accessToken string) ([]AdAccount, error)
	GetCampaigns(ctx context.Context, accessToken, adAccountID string) ([]MetaCampaign, error)
	GetAdSets(ctx context.Context, accessToken, campaignID string) ([]MetaAdSet, error)
	GetAds(ctx context.Context, accessToken, adSetID string) ([]MetaAd, error)
	GetInsights(ctx context.Context, accessToken, campaignID string, datePreset string) ([]MetaInsight, error)
	UpdateAdSet(ctx context.Context, accessToken, adSetID string, updates map[string]any) error
	UpdateCampaign(ctx context.Context, accessToken, campaignID string, updates map[string]any) error
	UpdateAdSetTargeting(ctx context.Context, accessToken, adSetID string, targeting map[string]any) error
	CreateCampaign(ctx context.Context, accessToken, adAccountID string, params map[string]any) (string, error)
	CreateAdSet(ctx context.Context, accessToken, adAccountID string, params map[string]any) (string, error)
	CreateAd(ctx context.Context, accessToken, adAccountID string, params map[string]any) (string, error)
}

type httpClient struct {
	http       *http.Client
	baseURL    string
	apiVersion string
}

// NewClient creates a Meta API client. Config values come from the config service at call time.
func NewClient(apiVersion string) Client {
	if apiVersion == "" {
		apiVersion = "v25.0"
	}
	return &httpClient{
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:    defaultBaseURL,
		apiVersion: apiVersion,
	}
}

// ─── Response types ──────────────────────────────────────────────────────────

type AdAccount struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type MetaCampaign struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Objective      string  `json:"objective"`
	Status         string  `json:"status"`
	DailyBudget    string  `json:"daily_budget"`
	LifetimeBudget string  `json:"lifetime_budget"`
}

type MetaAdSet struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	DailyBudget      string `json:"daily_budget"`
	OptimizationGoal string `json:"optimization_goal"`
	BillingEvent     string `json:"billing_event"`
}

type MetaAd struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Creative MetaCreative `json:"creative"`
}

type MetaCreative struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type MetaInsight struct {
	CampaignID  string `json:"campaign_id"`
	DateStart   string `json:"date_start"`
	Spend       string `json:"spend"`
	Impressions string `json:"impressions"`
	Clicks      string `json:"clicks"`
	CTR         string `json:"ctr"`
	CPC         string `json:"cpc"`
	CPM         string `json:"cpm"`
	Reach       string `json:"reach"`
	Frequency   string `json:"frequency"`
	Actions     []MetaAction `json:"actions"`
}

type MetaAction struct {
	ActionType string `json:"action_type"`
	Value      string `json:"value"`
}

// ─── Meta API error ───────────────────────────────────────────────────────────

type MetaError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (e *MetaError) Error() string {
	return fmt.Sprintf("meta api error %d (%s): %s", e.Code, e.Type, e.Message)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (c *httpClient) get(ctx context.Context, path string, params url.Values, out any) error {
	endpoint := fmt.Sprintf("%s/%s/%s?%s", c.baseURL, c.apiVersion, path, params.Encode())

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			backoff(attempt)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			backoff(attempt)
			continue
		}

		// Meta returns errors in the body even on 200
		var envelope struct {
			Error *MetaError      `json:"error"`
			Data  json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(body, &envelope); err != nil {
			lastErr = fmt.Errorf("parse response: %w", err)
			backoff(attempt)
			continue
		}
		if envelope.Error != nil {
			// 4xx meta errors are not retryable
			return envelope.Error
		}

		// Some endpoints return a flat object (not data:[])
		if envelope.Data != nil {
			return json.Unmarshal(envelope.Data, out)
		}
		return json.Unmarshal(body, out)
	}
	return fmt.Errorf("meta api: all retries failed: %w", lastErr)
}

func backoff(attempt int) {
	wait := time.Duration(attempt*attempt) * 500 * time.Millisecond
	slog.Debug("meta api: retrying", "attempt", attempt, "wait_ms", wait.Milliseconds())
	time.Sleep(wait)
}

// ─── API calls ────────────────────────────────────────────────────────────────

func (c *httpClient) GetAdAccounts(ctx context.Context, accessToken string) ([]AdAccount, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,name,currency")

	var result []AdAccount
	if err := c.get(ctx, "me/adaccounts", params, &result); err != nil {
		return nil, fmt.Errorf("get ad accounts: %w", err)
	}
	return result, nil
}

func (c *httpClient) GetCampaigns(ctx context.Context, accessToken, adAccountID string) ([]MetaCampaign, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,name,objective,status,daily_budget,lifetime_budget")
	params.Set("limit", "500")

	var result []MetaCampaign
	if err := c.get(ctx, fmt.Sprintf("act_%s/campaigns", adAccountID), params, &result); err != nil {
		return nil, fmt.Errorf("get campaigns: %w", err)
	}
	return result, nil
}

func (c *httpClient) GetAdSets(ctx context.Context, accessToken, campaignID string) ([]MetaAdSet, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,name,status,daily_budget,optimization_goal,billing_event")
	params.Set("limit", "200")

	var result []MetaAdSet
	if err := c.get(ctx, fmt.Sprintf("%s/adsets", campaignID), params, &result); err != nil {
		return nil, fmt.Errorf("get ad sets: %w", err)
	}
	return result, nil
}

func (c *httpClient) GetAds(ctx context.Context, accessToken, adSetID string) ([]MetaAd, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,name,status,creative{title,body}")
	params.Set("limit", "200")

	var result []MetaAd
	if err := c.get(ctx, fmt.Sprintf("%s/ads", adSetID), params, &result); err != nil {
		return nil, fmt.Errorf("get ads: %w", err)
	}
	return result, nil
}

func (c *httpClient) GetInsights(ctx context.Context, accessToken, campaignID, datePreset string) ([]MetaInsight, error) {
	if datePreset == "" {
		datePreset = "last_30d"
	}
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "campaign_id,date_start,spend,impressions,clicks,ctr,cpc,cpm,reach,frequency,actions,action_values")
	params.Set("date_preset", datePreset)
	params.Set("time_increment", "1")
	params.Set("action_attribution_windows", "1d_click,7d_click,1d_view")
	params.Set("limit", "90")

	var result []MetaInsight
	if err := c.get(ctx, fmt.Sprintf("%s/insights", campaignID), params, &result); err != nil {
		return nil, fmt.Errorf("get insights: %w", err)
	}
	return result, nil
}

// ─── Write operations ─────────────────────────────────────────────────────────

// CreateCampaign creates a campaign on Meta Ads and returns the campaign ID.
func (c *httpClient) CreateCampaign(ctx context.Context, accessToken, adAccountID string, params map[string]any) (string, error) {
	body := map[string]any{"access_token": accessToken}
	for k, v := range params {
		body[k] = v
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := c.post(ctx, fmt.Sprintf("act_%s/campaigns", adAccountID), body, &result); err != nil {
		return "", fmt.Errorf("create campaign: %w", err)
	}
	return result.ID, nil
}

// CreateAdSet creates an ad set on Meta Ads and returns the ad set ID.
func (c *httpClient) CreateAdSet(ctx context.Context, accessToken, adAccountID string, params map[string]any) (string, error) {
	body := map[string]any{"access_token": accessToken}
	for k, v := range params {
		body[k] = v
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := c.post(ctx, fmt.Sprintf("act_%s/adsets", adAccountID), body, &result); err != nil {
		return "", fmt.Errorf("create ad set: %w", err)
	}
	return result.ID, nil
}

// CreateAdCreative creates an ad with creative on Meta Ads and returns the ad ID.
func (c *httpClient) CreateAd(ctx context.Context, accessToken, adAccountID string, params map[string]any) (string, error) {
	body := map[string]any{"access_token": accessToken}
	for k, v := range params {
		body[k] = v
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := c.post(ctx, fmt.Sprintf("act_%s/ads", adAccountID), body, &result); err != nil {
		return "", fmt.Errorf("create ad: %w", err)
	}
	return result.ID, nil
}

func (c *httpClient) UpdateAdSetTargeting(ctx context.Context, accessToken, adSetID string, targeting map[string]any) error {
	body := map[string]any{
		"access_token": accessToken,
		"targeting":    targeting,
	}
	return c.post(ctx, fmt.Sprintf("%s", adSetID), body, nil)
}

func (c *httpClient) UpdateAdSet(ctx context.Context, accessToken, adSetID string, updates map[string]any) error {
	updates["access_token"] = accessToken
	return c.post(ctx, fmt.Sprintf("%s", adSetID), updates, nil)
}

func (c *httpClient) UpdateCampaign(ctx context.Context, accessToken, campaignID string, updates map[string]any) error {
	updates["access_token"] = accessToken
	return c.post(ctx, fmt.Sprintf("%s", campaignID), updates, nil)
}

func (c *httpClient) post(ctx context.Context, path string, body map[string]any, out any) error {
	endpoint := fmt.Sprintf("%s/%s/%s", c.baseURL, c.apiVersion, path)
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("meta api post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var envelope struct {
		Error  *MetaError     `json:"error"`
		Data   json.RawMessage `json:"data"`
		ID     string          `json:"id"`
		Success bool           `json:"success"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if envelope.Error != nil {
		return envelope.Error
	}

	if out != nil {
		return json.Unmarshal(respBody, out)
	}
	return nil
}

// DatePresets lists all valid Meta date presets
var DatePresets = []string{
	"today", "yesterday", "this_month", "last_month",
	"this_quarter", "last_3d", "last_7d", "last_14d",
	"last_28d", "last_30d", "last_90d", "last_quarter",
	"last_year", "this_week_mon_today", "this_year", "maximum",
}
