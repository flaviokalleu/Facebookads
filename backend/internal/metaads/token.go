package metaads

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// TokenInfo mirrors the payload returned by /debug_token.
type TokenInfo struct {
	AppID       string    `json:"app_id"`
	Application string    `json:"application"`
	Type        string    `json:"type"`
	UserID      string    `json:"user_id"`
	Scopes      []string  `json:"scopes"`
	ExpiresAt   time.Time `json:"expires_at"`
	IsValid     bool      `json:"is_valid"`
}

// AppSecretProof returns hex(HMAC-SHA256(access_token, app_secret)).
// Meta requires this on every Graph API call when "Require App Secret Proof"
// is enabled on the app.
func (c *httpClient) AppSecretProof(accessToken, appSecret string) string {
	if accessToken == "" || appSecret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(accessToken))
	return hex.EncodeToString(mac.Sum(nil))
}

// ExchangeForLongLived swaps a short-lived user token for a 60-day token via
// GET /oauth/access_token?grant_type=fb_exchange_token. The endpoint does NOT
// wrap its response under "data", so we read the raw body.
func (c *httpClient) ExchangeForLongLived(ctx context.Context, appID, appSecret, shortToken string) (string, int, error) {
	params := url.Values{}
	params.Set("grant_type", "fb_exchange_token")
	params.Set("client_id", appID)
	params.Set("client_secret", appSecret)
	params.Set("fb_exchange_token", shortToken)

	var out struct {
		AccessToken string     `json:"access_token"`
		TokenType   string     `json:"token_type"`
		ExpiresIn   int        `json:"expires_in"`
		Error       *MetaError `json:"error"`
	}
	if err := c.getRaw(ctx, "oauth/access_token", params, &out); err != nil {
		return "", 0, fmt.Errorf("exchange long-lived: %w", err)
	}
	if out.Error != nil {
		return "", 0, out.Error
	}
	if out.AccessToken == "" {
		return "", 0, fmt.Errorf("exchange long-lived: empty token in response")
	}
	return out.AccessToken, out.ExpiresIn, nil
}

// DebugToken hits /debug_token to validate and inspect a token. The required
// "access_token" param must be the app token: "{app_id}|{app_secret}".
func (c *httpClient) DebugToken(ctx context.Context, inputToken, appID, appSecret string) (*TokenInfo, error) {
	params := url.Values{}
	params.Set("input_token", inputToken)
	params.Set("access_token", fmt.Sprintf("%s|%s", appID, appSecret))

	// debug_token returns { "data": { ...fields... } } — a single object, not array.
	var out struct {
		Data struct {
			AppID       string   `json:"app_id"`
			Application string   `json:"application"`
			Type        string   `json:"type"`
			UserID      string   `json:"user_id"`
			Scopes      []string `json:"scopes"`
			ExpiresAt   int64    `json:"expires_at"`
			IsValid     bool     `json:"is_valid"`
		} `json:"data"`
		Error *MetaError `json:"error"`
	}
	if err := c.getRaw(ctx, "debug_token", params, &out); err != nil {
		return nil, fmt.Errorf("debug_token: %w", err)
	}
	if out.Error != nil {
		return nil, out.Error
	}

	info := &TokenInfo{
		AppID:       out.Data.AppID,
		Application: out.Data.Application,
		Type:        out.Data.Type,
		UserID:      out.Data.UserID,
		Scopes:      out.Data.Scopes,
		IsValid:     out.Data.IsValid,
	}
	if out.Data.ExpiresAt > 0 {
		info.ExpiresAt = time.Unix(out.Data.ExpiresAt, 0)
	}
	return info, nil
}

// getRaw is like get() but does NOT unwrap the "data" envelope. Used for
// endpoints (oauth/access_token, debug_token) where the response shape differs.
func (c *httpClient) getRaw(ctx context.Context, path string, params url.Values, out any) error {
	endpoint := fmt.Sprintf("%s/%s/%s?%s", c.baseURL, c.apiVersion, path, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// withProof returns the params with appsecret_proof appended when appSecret
// is set. Caller already provided access_token in params.
func (c *httpClient) withProof(params url.Values, accessToken, appSecret string) url.Values {
	if appSecret == "" || accessToken == "" {
		return params
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("appsecret_proof", c.AppSecretProof(accessToken, appSecret))
	return params
}

