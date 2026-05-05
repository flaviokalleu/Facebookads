package metaads

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// CustomAudience mirrors the rows the Marketing API returns from
// /act_{id}/customaudiences.
type CustomAudience struct {
	ID                          string `json:"id"`
	Name                        string `json:"name"`
	Subtype                     string `json:"subtype"`
	Description                 string `json:"description"`
	ApproximateCountLowerBound  int64  `json:"approximate_count_lower_bound"`
	ApproximateCountUpperBound  int64  `json:"approximate_count_upper_bound"`
	DeliveryStatus              struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"delivery_status"`
	OperationStatus struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"operation_status"`
	TimeCreated int64  `json:"time_created"`
	TimeUpdated int64  `json:"time_updated"`
	Rule        string `json:"rule"`
}

// GetCustomAudiences pulls every custom audience for the given account
// (numeric id, with or without `act_` prefix). Includes lookalikes — they live
// in the same endpoint with subtype=LOOKALIKE.
func (c *httpClient) GetCustomAudiences(ctx context.Context, accessToken, accountID string) ([]CustomAudience, error) {
	accountID = strings.TrimPrefix(accountID, "act_")

	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("fields", "id,name,subtype,description,approximate_count_lower_bound,approximate_count_upper_bound,delivery_status,operation_status,time_created,time_updated,rule")
	params.Set("limit", "200")

	var out []CustomAudience
	if err := c.get(ctx, fmt.Sprintf("act_%s/customaudiences", accountID), params, &out); err != nil {
		return nil, fmt.Errorf("get custom audiences: %w", err)
	}
	return out, nil
}
