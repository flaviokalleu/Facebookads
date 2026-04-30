package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
)

type CreativeTestHandler struct {
	metaClient metaads.Client
	tokenRepo  repository.UserTokenRepository
}

func NewCreativeTestHandler(metaClient metaads.Client, tokenRepo repository.UserTokenRepository) *CreativeTestHandler {
	return &CreativeTestHandler{metaClient: metaClient, tokenRepo: tokenRepo}
}

// Start133 cria 1 campanha, 3 conjuntos, 3+ anúncios para teste de criativos
func (h *CreativeTestHandler) Start133(c *fiber.Ctx) error {
	var in struct {
		Name      string  `json:"name"`
		Niche     string  `json:"niche"`
		Budget    float64 `json:"budget"`
		MinAge    int     `json:"min_age"`
		MaxAge    int     `json:"max_age"`
		Gender    string  `json:"gender"`
		Country   string  `json:"country"`
		PageID    string  `json:"page_id"`
		Creatives []struct {
			Headline string `json:"headline"`
			Body     string `json:"body"`
			CTA      string `json:"cta"`
		} `json:"creatives"`
	}
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "body invalido")
	}
	if in.Name == "" { in.Name = "Teste 133 - " + in.Niche }
	if in.Budget <= 0 { in.Budget = 50 }
	if in.MinAge == 0 { in.MinAge = 18 }
	if in.MaxAge == 0 { in.MaxAge = 65 }
	if in.Country == "" { in.Country = "BR" }
	if len(in.Creatives) == 0 {
		// Default creatives if none provided
		in.Creatives = []struct {
			Headline string `json:"headline"`
			Body     string `json:"body"`
			CTA      string `json:"cta"`
		}{
			{Headline: "Oferta Especial", Body: "Aproveite esta oportunidade unica!", CTA: "LEARN_MORE"},
			{Headline: "Voce merece isso", Body: "Descubra o que podemos fazer por voce.", CTA: "LEARN_MORE"},
			{Headline: "Nao perca tempo", Body: "Resultados reais para voce. Confira!", CTA: "WHATSAPP_MESSAGE"},
		}
	}

	userID := middleware.UserID(c)
	tokens, err := h.tokenRepo.ListByUser(c.UserContext(), userID)
	if err != nil || len(tokens) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "sem conta Meta conectada")
	}
	tok := tokens[0].EncryptedToken
	adAccountID := "4318540458418505"
	budgetCents := int64(in.Budget * 100)

	// 3 audiences diferentes
	audiences := []struct {
		Name     string
		MinAge   int
		MaxAge   int
		Gender   []int
		Interests string
	}{
		{Name: "Publico 1 - Geral", MinAge: in.MinAge, MaxAge: in.MaxAge, Gender: []int{0}},
		{Name: "Publico 2 - Interesses", MinAge: in.MinAge, MaxAge: in.MaxAge, Gender: []int{0}, Interests: in.Niche + ", casa propria, financiamento"},
		{Name: "Publico 3 - Amplo", MinAge: 18, MaxAge: 60, Gender: []int{0}},
	}

	// 1. Criar Campanha com CBO
	campID, cerr := h.metaClient.CreateCampaign(c.UserContext(), tok, adAccountID, map[string]any{
		"name":                in.Name,
		"objective":           "OUTCOME_ENGAGEMENT",
		"status":              "PAUSED",
		"daily_budget":        budgetCents,
		"special_ad_categories": []string{"NONE"},
	})
	if cerr != nil {
		return fiber.NewError(fiber.StatusBadRequest, "campanha: "+cerr.Error())
	}

	type result struct {
		AdSetID string `json:"ad_set_id"`
		Name    string `json:"name"`
		AdID    string `json:"ad_id"`
		Error   string `json:"error,omitempty"`
	}
	var results []result

	// 2. Para cada público, criar 1 conjunto + 1 anúncio
	for i, aud := range audiences {
		targeting := map[string]any{
			"age_min": aud.MinAge, "age_max": aud.MaxAge,
			"genders": aud.Gender,
			"geo_locations": map[string]any{"countries": []string{in.Country}},
			"targeting_automation": map[string]any{"advantage_audience": 0},
		}
		if aud.Interests != "" {
			targeting["flexible_spec"] = []map[string]any{{"interests": []map[string]any{{"name": aud.Interests}}}}
		}

		asName := fmt.Sprintf("%s - %s", in.Name, aud.Name)
		asid, aerr := h.metaClient.CreateAdSet(c.UserContext(), tok, adAccountID, map[string]any{
			"campaign_id": campID, "name": asName, "status": "PAUSED",
			"billing_event": "IMPRESSIONS", "optimization_goal": "REACH",
			"bid_amount": 100, "targeting": targeting,
		})
		if aerr != nil {
			results = append(results, result{Name: asName, Error: "conjunto: " + aerr.Error()})
			continue
		}

		// 3. Criar 1 anúncio para este conjunto
		creative := in.Creatives[i%len(in.Creatives)]
		cta := creative.CTA
		if cta == "" { cta = "LEARN_MORE" }

		adID, aderr := h.metaClient.CreateAd(c.UserContext(), tok, adAccountID, map[string]any{
			"name": asName + " - Anuncio", "adset_id": asid, "status": "PAUSED",
			"creative": map[string]any{
				"object_story_spec": map[string]any{
					"page_id": in.PageID,
					"link_data": map[string]any{
						"link": "https://www.facebook.com/" + in.PageID,
						"message": creative.Body,
						"call_to_action": map[string]any{"type": cta},
					},
				},
			},
		})
		if aderr != nil {
			results = append(results, result{AdSetID: asid, Name: asName, Error: "anuncio: " + aderr.Error()})
		} else {
			results = append(results, result{AdSetID: asid, AdID: adID, Name: asName})
		}
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"campaign_id": campID,
		"strategy":    "133",
		"results":     results,
	}})
}

var _ = strings.TrimSpace
