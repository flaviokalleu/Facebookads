package handler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
	"github.com/facebookads/backend/internal/usecase"
	"log/slog"
)

type CreateCampaignHandler struct {
	uc         *usecase.CampaignUseCase
	aiRouter   *ai.Router
	tokenRepo  repository.UserTokenRepository
	metaClient metaads.Client
}

func NewCreateCampaignHandler(uc *usecase.CampaignUseCase, aiRouter *ai.Router, tokenRepo repository.UserTokenRepository, metaClient metaads.Client) *CreateCampaignHandler {
	return &CreateCampaignHandler{uc: uc, aiRouter: aiRouter, tokenRepo: tokenRepo, metaClient: metaClient}
}

func (h *CreateCampaignHandler) findToken() func([]*domain.UserToken) (string, string) {
	return func(list []*domain.UserToken) (string, string) {
		for _, t := range list {
			if t.AdAccountID == "1386606178278837" {
				return t.EncryptedToken, t.AdAccountID
			}
		}
		if len(list) > 0 {
			return list[0].EncryptedToken, list[0].AdAccountID
		}
		return "", ""
	}
}

func makeGeo(country, cityDetails string) map[string]any {
	geo := map[string]any{"countries": []string{country}}
	if cityDetails != "" {
		var raw []map[string]any
		if json.Unmarshal([]byte(cityDetails), &raw) == nil && len(raw) > 0 {
			var locs []map[string]any
			for _, item := range raw {
				loc := map[string]any{"radius": 10, "distance_unit": "km"}
				if k, ok := item["key"]; ok && k != nil && fmt.Sprintf("%v", k) != "" {
					loc["key"] = k
				} else if n, ok := item["name"]; ok && n != nil && fmt.Sprintf("%v", n) != "" {
					loc["name"] = n
				}
				if r, ok := item["radius"]; ok { loc["radius"] = r }
				locs = append(locs, loc)
			}
			if len(locs) > 0 { geo["custom_locations"] = locs }
		}
	}
	return geo
}

func (h *CreateCampaignHandler) CreateFull(c *fiber.Ctx) error {
	var in struct {
		Name        string  `json:"name"`
		Objective   string  `json:"objective"`
		Budget      float64 `json:"budget"`
		MinAge      int     `json:"min_age"`
		MaxAge      int     `json:"max_age"`
		Gender      string  `json:"gender"`
		Interests   string  `json:"interests"`
		Cities      string  `json:"cities"`
		Country     string  `json:"country"`
		CityDetails string  `json:"city_details"`
		PageID      string  `json:"page_id"`
	}
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "body invalido")
	}
	if in.Name == "" { return fiber.NewError(fiber.StatusBadRequest, "nome obrigatorio") }
	if in.MinAge == 0 { in.MinAge = 18 }
	if in.MaxAge == 0 { in.MaxAge = 65 }
	if in.Country == "" { in.Country = "BR" }
	if in.Budget <= 0 { in.Budget = 50 }
	if in.Objective == "" { in.Objective = "OUTCOME_ENGAGEMENT" }

	budget := in.Budget
	camp, err := h.uc.Create(c.UserContext(), middleware.UserID(c), usecase.CreateCampaignInput{
		Name: in.Name, Objective: in.Objective, AdAccountID: "1386606178278837", DailyBudget: &budget,
	})
	if err != nil { return mapError(err) }

	var mCampID, mAdSetID, mAdID string
	note := "Criada localmente."
	uid := middleware.UserID(c)

	tokens, err := h.tokenRepo.ListByUser(c.UserContext(), uid)
	if err == nil && len(tokens) > 0 {
		tok, acct := h.findToken()(tokens)
		cents := int64(in.Budget * 100)
		geo := makeGeo(in.Country, in.CityDetails)

		var genders []int
		switch in.Gender {
		case "male": genders = []int{1}
		case "female": genders = []int{2}
		default: genders = []int{0}
		}

		// Campaign
		mid, e := h.metaClient.CreateCampaign(c.UserContext(), tok, acct, map[string]any{
			"name": in.Name, "objective": mapObjective(in.Objective),
			"status": "PAUSED", "daily_budget": cents, "special_ad_categories": []string{"NONE"},
		})
		if e != nil { return c.JSON(fiber.Map{"data": fiber.Map{"error": e.Error()}}) }
		mCampID = mid

		// Ad Set
		tgt := map[string]any{"age_min": in.MinAge, "age_max": in.MaxAge, "genders": genders, "geo_locations": geo, "targeting_automation": map[string]any{"advantage_audience": 0}}
		if in.Interests != "" {
			tgt["flexible_spec"] = []map[string]any{{"interests": []map[string]any{{"name": in.Interests}}}}
		}

		asid, e := h.metaClient.CreateAdSet(c.UserContext(), tok, acct, map[string]any{
			"campaign_id": mid, "name": in.Name + " - Conjunto", "status": "PAUSED",
			"billing_event": "IMPRESSIONS", "optimization_goal": "REACH", "bid_amount": 100,
			"targeting": tgt,
		})
		if e != nil {
			note = "Campanha criada, conjunto falhou: " + e.Error()
		} else {
			mAdSetID = asid
			if in.PageID != "" {
				cta := "WHATSAPP_MESSAGE"
				if strings.Contains(strings.ToLower(in.Objective), "lead") { cta = "SIGN_UP" }
				adid, e := h.metaClient.CreateAd(c.UserContext(), tok, acct, map[string]any{
					"name": in.Name + " - Anuncio", "adset_id": asid, "status": "PAUSED",
					"creative": map[string]any{"object_story_spec": map[string]any{
						"page_id": in.PageID,
						"link_data": map[string]any{
							"link": "https://www.facebook.com/" + in.PageID,
							"message": "Confira " + in.Name + "!",
							"call_to_action": map[string]any{"type": cta},
						},
					}},
				})
				if e != nil {
					slog.Error("ad fail", "err", e)
					note = "Campanha+Conjunto criados, anuncio falhou."
				} else {
					mAdID = adid
					note = "Campanha+Conjunto+Anuncio criados (PAUSED)."
				}
			} else {
				note = "Campanha+Conjunto criados. Vincule uma Pagina Facebook para criar anuncios."
			}
		}
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"campaign": camp, "meta_campaign_id": mCampID,
		"meta_ad_set_id": mAdSetID, "meta_ad_id": mAdID, "meta_note": note,
	}})
}

func mapObjective(obj string) string {
	m := map[string]string{
		"CONVERSIONS": "OUTCOME_ENGAGEMENT", "LEADS": "OUTCOME_LEADS",
		"SALES": "OUTCOME_SALES", "TRAFFIC": "OUTCOME_TRAFFIC",
		"REACH": "OUTCOME_AWARENESS", "ENGAGEMENT": "OUTCOME_ENGAGEMENT",
		"BRAND_AWARENESS": "OUTCOME_AWARENESS", "APP_INSTALLS": "OUTCOME_APP_PROMOTION",
	}
	if v, ok := m[obj]; ok { return v }
	return obj
}
