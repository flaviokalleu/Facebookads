package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
	"github.com/facebookads/backend/internal/usecase"
)

type CreativeHandler struct {
	campaignUC *usecase.CampaignUseCase
	ads        repository.AdRepository
	adSets     repository.AdSetRepository
	insights   repository.InsightRepository
	aiRouter   *ai.Router
}

func NewCreativeHandler(
	campaignUC *usecase.CampaignUseCase,
	ads repository.AdRepository,
	adSets repository.AdSetRepository,
	insights repository.InsightRepository,
	aiRouter *ai.Router,
) *CreativeHandler {
	return &CreativeHandler{
		campaignUC: campaignUC,
		ads:        ads,
		adSets:     adSets,
		insights:   insights,
		aiRouter:   aiRouter,
	}
}

type creativeItem struct {
	ID             string  `json:"id"`
	CampaignName   string  `json:"campaign_name"`
	CampaignID     string  `json:"campaign_id"`
	Headline       string  `json:"headline"`
	Body           string  `json:"body"`
	CTAType        string  `json:"cta_type"`
	Name           string  `json:"name"`
	FatigueScore   float64 `json:"fatigue_score"`
	CTR            float64 `json:"ctr"`
	Impressions    int64   `json:"impressions"`
	Frequency      float64 `json:"frequency"`
	Recommendation string  `json:"recommendation,omitempty"`
	Status         string  `json:"status"`
}

type creativeData struct {
	Name     string  `json:"name"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	CTA      string  `json:"cta"`
	CTR      float64 `json:"ctr"`
	Impressions int64 `json:"impressions"`
	Frequency float64 `json:"frequency"`
	Campaign  string  `json:"campaign"`
}

// List returns all creatives across user campaigns with fatigue scores.
func (h *CreativeHandler) List(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	campaigns, err := h.campaignUC.List(c.UserContext(), userID)
	if err != nil {
		return mapError(err)
	}

	var result []creativeItem
	for _, camp := range campaigns {
		adSets, _ := h.adSets.ListByCampaign(c.UserContext(), camp.ID)
		for _, as := range adSets {
			ads, _ := h.ads.ListByAdSet(c.UserContext(), as.ID)
			for _, ad := range ads {
				insights, _ := h.insights.ListByCampaign(c.UserContext(), camp.ID, time.Now().AddDate(0, 0, -14), time.Now())
				fatigue := computeFatigue(insights)
				ctr, impressions, freq := aggregateMetrics(insights)
				result = append(result, creativeItem{
					ID:           ad.ID,
					CampaignName: camp.Name,
					CampaignID:   camp.ID,
					Headline:     ad.CreativeTitle,
					Body:         ad.CreativeBody,
					CTAType:      ad.CTAType,
					Name:         ad.Name,
					FatigueScore: fatigue,
					CTR:          ctr,
					Impressions:  impressions,
					Frequency:    freq,
					Status:       ad.Status,
				})
			}
		}
	}
	if result == nil {
		result = []creativeItem{}
	}
	return c.JSON(fiber.Map{"data": result})
}

// Analyze uses AI to generate creative insights from performance data.
func (h *CreativeHandler) Analyze(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	campaigns, err := h.campaignUC.List(c.UserContext(), userID)
	if err != nil {
		return mapError(err)
	}

	var topCreatives, bottomCreatives []creativeData
	for _, camp := range campaigns {
		adSets, _ := h.adSets.ListByCampaign(c.UserContext(), camp.ID)
		for _, as := range adSets {
			ads, _ := h.ads.ListByAdSet(c.UserContext(), as.ID)
			for _, ad := range ads {
				insights, _ := h.insights.ListByCampaign(c.UserContext(), camp.ID, time.Now().AddDate(0, 0, -14), time.Now())
				ctr, impressions, freq := aggregateMetrics(insights)
				cd := creativeData{
					Name:        ad.Name,
					Headline:    ad.CreativeTitle,
					Body:        ad.CreativeBody,
					CTA:         ad.CTAType,
					CTR:         ctr,
					Impressions: impressions,
					Frequency:   freq,
					Campaign:    camp.Name,
				}
				if ctr > 0.01 {
					topCreatives = append(topCreatives, cd)
				} else {
					bottomCreatives = append(bottomCreatives, cd)
				}
			}
		}
	}

	if len(topCreatives) == 0 && len(bottomCreatives) == 0 {
		return c.JSON(fiber.Map{"data": fiber.Map{
			"creatives": []creativeData{},
			"ai_analyzed": false,
			"message": "No creatives found. Sync your campaigns first.",
		}})
	}

	sortByCTR(topCreatives, true)
	sortByCTR(bottomCreatives, false)
	if len(topCreatives) > 5 { topCreatives = topCreatives[:5] }
	if len(bottomCreatives) > 5 { bottomCreatives = bottomCreatives[:5] }

	topJSON, _ := json.Marshal(topCreatives)
	bottomJSON, _ := json.Marshal(bottomCreatives)

	prompt := fmt.Sprintf(`Analyze these Meta Ads creatives.

TOP PERFORMERS (by CTR):
%s

BOTTOM PERFORMERS (by CTR):
%s

Respond in JSON format:
{
  "winning_patterns": ["what works in top creatives"],
  "losing_patterns": ["what fails in bottom creatives"],
  "headline_insights": "analysis of headline patterns",
  "cta_insights": "which CTAs perform best",
  "recommendations": ["actionable improvement suggestions"]
}`, string(topJSON), string(bottomJSON))

	resp, err := h.aiRouter.Complete(c.UserContext(), ai.TaskCreativeAnalysis, ai.CompletionRequest{
		SystemPrompt: "You are a Meta Ads creative strategist. Respond only with valid JSON.",
		UserPrompt:   prompt,
		MaxTokens:    1500,
		Temperature:  0.3,
		JSONMode:     true,
	})
	if err != nil {
		return c.JSON(fiber.Map{"data": fiber.Map{
			"creatives":   append(topCreatives, bottomCreatives...),
			"ai_insights": nil,
			"ai_analyzed": false,
			"ai_error":    err.Error(),
		}})
	}

	var insights map[string]any
	json.Unmarshal([]byte(cleanJSON(resp.Content)), &insights)

	allCreatives := append(topCreatives, bottomCreatives...)
	return c.JSON(fiber.Map{"data": fiber.Map{
		"creatives":   allCreatives,
		"ai_insights": insights,
		"ai_analyzed": true,
		"model_used":  resp.Provider + "/" + resp.ModelUsed,
	}})
}

// Improve takes user instructions and generates AI-powered creative variations.
func (h *CreativeHandler) Improve(c *fiber.Ctx) error {
	var body struct {
		Instructions  string `json:"instructions"`
		CreativeID    string `json:"creative_id,omitempty"`
		CampaignName  string `json:"campaign_name,omitempty"`
		Headline      string `json:"headline,omitempty"`
		Body          string `json:"body,omitempty"`
		CTA           string `json:"cta,omitempty"`
		TargetAudience string `json:"target_audience,omitempty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if body.Instructions == "" {
		return fiber.NewError(fiber.StatusBadRequest, "instructions are required")
	}

	creativeInfo := fmt.Sprintf("Current Creative:\nHeadline: %s\nBody: %s\nCTA: %s\nCampaign: %s\nTarget: %s\n",
		body.Headline, body.Body, body.CTA, body.CampaignName, body.TargetAudience)

	prompt := fmt.Sprintf(`A Meta Ads advertiser wants to improve their creative.

%s

User instructions: %s

Generate 3 improved creative variations as JSON array:
[{
  "variant": 1,
  "headline": "improved headline",
  "primary_text": "improved body text",
  "cta": "improved call to action",
  "reasoning": "why this change helps",
  "expected_impact": "what metrics this should improve"
}]`, creativeInfo, body.Instructions)

	resp, err := h.aiRouter.Complete(c.UserContext(), ai.TaskCreativeAnalysis, ai.CompletionRequest{
		SystemPrompt: "You are a Meta Ads creative optimization expert. Generate practical, high-impact creative improvements. Respond only with valid JSON.",
		UserPrompt:   prompt,
		MaxTokens:    2000,
		Temperature:  0.7,
		JSONMode:     true,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "ai analysis failed: "+err.Error())
	}

	var variations []map[string]any
	if err := json.Unmarshal([]byte(cleanJSON(resp.Content)), &variations); err != nil {
		return c.JSON(fiber.Map{"data": fiber.Map{
			"variations": []map[string]any{},
			"raw":        resp.Content,
			"model_used": resp.Provider + "/" + resp.ModelUsed,
		}})
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"variations": variations,
		"model_used": resp.Provider + "/" + resp.ModelUsed,
	}})
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func computeFatigue(insights []*domain.CampaignInsight) float64 {
	if len(insights) == 0 {
		return 0
	}
	var totalFreq float64
	for _, i := range insights {
		totalFreq += i.Frequency
	}
	avgFreq := totalFreq / float64(len(insights))

	var totalCTR float64
	for _, i := range insights {
		totalCTR += i.CTR
	}
	avgCTR := totalCTR / float64(len(insights))

	freqScore := min(avgFreq/5.0, 1.0) * 50
	ctrScore := (1.0 - min(avgCTR*50, 1.0)) * 50
	return freqScore + ctrScore
}

func aggregateMetrics(insights []*domain.CampaignInsight) (ctr float64, impressions int64, freq float64) {
	if len(insights) == 0 {
		return 0, 0, 0
	}
	var ctrSum, freqSum float64
	var impSum int64
	for _, i := range insights {
		ctrSum += i.CTR
		impSum += i.Impressions
		freqSum += i.Frequency
	}
	n := float64(len(insights))
	return ctrSum / n, impSum, freqSum / n
}

func sortByCTR(list []creativeData, desc bool) {
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if desc && list[j].CTR > list[i].CTR {
				list[i], list[j] = list[j], list[i]
			} else if !desc && list[j].CTR < list[i].CTR {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
}

var _ = context.Background
