package metaads

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
)

// ─── BM hierarchy types ───────────────────────────────────────────────────────

type Business struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	VerificationStatus string `json:"verification_status"`
	TimezoneID         int    `json:"timezone_id"`
	Vertical           string `json:"vertical"`
}

type BusinessRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AdAccountFull struct {
	ID            string       `json:"id"`         // act_xxxx
	AccountID     string       `json:"account_id"` // numeric only
	Name          string       `json:"name"`
	Currency      string       `json:"currency"`
	TimezoneName  string       `json:"timezone_name"`
	AccountStatus int          `json:"account_status"`
	DisableReason int          `json:"disable_reason"`
	SpendCap      string       `json:"spend_cap"`
	AmountSpent   string       `json:"amount_spent"`
	Balance       string       `json:"balance"`
	Business      *BusinessRef `json:"business,omitempty"`
}

type Page struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	FanCount    int64  `json:"fan_count"`
	AccessToken string `json:"access_token"`
}

type Pixel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	LastFiredTime string `json:"last_fired_time"`
	IsUnavailable bool   `json:"is_unavailable"`
}

type InstagramAccount struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	ProfilePic string `json:"profile_pic"`
}

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	listLimit       = "200"
	adAccountFields = "id,account_id,name,currency,timezone_name,account_status,disable_reason,spend_cap,amount_spent,balance,business"
)

// ─── Business Manager listing ─────────────────────────────────────────────────

func (c *httpClient) GetBusinesses(ctx context.Context, accessToken, appSecret string) ([]Business, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,name,verification_status,timezone_id,vertical")
	params.Set("limit", listLimit)
	params = c.withProof(params, accessToken, appSecret)

	var result []Business
	if err := c.get(ctx, "me/businesses", params, &result); err != nil {
		return nil, fmt.Errorf("get businesses: %w", err)
	}
	return result, nil
}

// GetMyAdAccounts returns ad accounts visible to the user that are NOT under
// a Business Manager (personal accounts) — Meta returns those at /me/adaccounts.
// pagination is left as a TODO; we cap at 200.
func (c *httpClient) GetMyAdAccounts(ctx context.Context, accessToken, appSecret string) ([]AdAccountFull, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", adAccountFields)
	params.Set("limit", listLimit)
	params = c.withProof(params, accessToken, appSecret)

	var result []AdAccountFull
	if err := c.get(ctx, "me/adaccounts", params, &result); err != nil {
		return nil, fmt.Errorf("get my adaccounts: %w", err)
	}
	return result, nil
}

func (c *httpClient) GetBMOwnedAccounts(ctx context.Context, accessToken, appSecret, bmID string) ([]AdAccountFull, error) {
	return c.bmAccounts(ctx, accessToken, appSecret, bmID, "owned_ad_accounts")
}

func (c *httpClient) GetBMClientAccounts(ctx context.Context, accessToken, appSecret, bmID string) ([]AdAccountFull, error) {
	return c.bmAccounts(ctx, accessToken, appSecret, bmID, "client_ad_accounts")
}

func (c *httpClient) bmAccounts(ctx context.Context, accessToken, appSecret, bmID, edge string) ([]AdAccountFull, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", adAccountFields)
	params.Set("limit", listLimit)
	params = c.withProof(params, accessToken, appSecret)

	var result []AdAccountFull
	if err := c.get(ctx, fmt.Sprintf("%s/%s", bmID, edge), params, &result); err != nil {
		slog.Warn("meta: bm accounts list failed", "bm_id", bmID, "edge", edge, "err", err)
		return nil, fmt.Errorf("get %s: %w", edge, err)
	}
	return result, nil
}

func (c *httpClient) GetBMPages(ctx context.Context, accessToken, appSecret, bmID string) ([]Page, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,name,category,fan_count,access_token")
	params.Set("limit", listLimit)
	params = c.withProof(params, accessToken, appSecret)

	var owned []Page
	if err := c.get(ctx, fmt.Sprintf("%s/owned_pages", bmID), params, &owned); err != nil {
		slog.Warn("meta: owned_pages failed", "bm_id", bmID, "err", err)
	}
	var clients []Page
	if err := c.get(ctx, fmt.Sprintf("%s/client_pages", bmID), params, &clients); err != nil {
		slog.Warn("meta: client_pages failed", "bm_id", bmID, "err", err)
	}
	return append(owned, clients...), nil
}

func (c *httpClient) GetBMPixels(ctx context.Context, accessToken, appSecret, bmID string) ([]Pixel, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,name,last_fired_time,is_unavailable")
	params.Set("limit", listLimit)
	params = c.withProof(params, accessToken, appSecret)

	var result []Pixel
	if err := c.get(ctx, fmt.Sprintf("%s/adspixels", bmID), params, &result); err != nil {
		return nil, fmt.Errorf("get adspixels: %w", err)
	}
	return result, nil
}

func (c *httpClient) GetBMInstagram(ctx context.Context, accessToken, appSecret, bmID string) ([]InstagramAccount, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,username,profile_pic")
	params.Set("limit", listLimit)
	params = c.withProof(params, accessToken, appSecret)

	var result []InstagramAccount
	if err := c.get(ctx, fmt.Sprintf("%s/owned_instagram_accounts", bmID), params, &result); err != nil {
		return nil, fmt.Errorf("get owned_instagram_accounts: %w", err)
	}
	return result, nil
}
