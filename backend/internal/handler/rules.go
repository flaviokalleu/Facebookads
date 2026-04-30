package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/facebookads/backend/internal/config"

	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
	"github.com/facebookads/backend/internal/usecase"
)

type RulesHandler struct {
	cfg        *config.Service
	campaignUC *usecase.CampaignUseCase
	metaClient metaads.Client
	tokenRepo  repository.UserTokenRepository
}

func NewRulesHandler(cfg *config.Service, campaignUC *usecase.CampaignUseCase, metaClient metaads.Client, tokenRepo repository.UserTokenRepository) *RulesHandler {
	return &RulesHandler{cfg: cfg, campaignUC: campaignUC, metaClient: metaClient, tokenRepo: tokenRepo}
}

func (h *RulesHandler) Save(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	var in struct {
		Rules []map[string]any `json:"rules"`
	}
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "body invalido")
	}
	rulesJSON, _ := json.Marshal(in.Rules)
	key := "smart_rules_" + userID
	if err := h.cfg.Set(c.UserContext(), key, string(rulesJSON), false); err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"saved": true, "rules_count": len(in.Rules)}})
}

func (h *RulesHandler) Get(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	key := "smart_rules_" + userID
	val := h.cfg.Get(key)
	if val == "" {
		val = "[]"
	}
	var rules []map[string]any
	json.Unmarshal([]byte(val), &rules)
	return c.JSON(fiber.Map{"data": rules})
}

func (h *RulesHandler) ABTest(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	campaignID := c.Params("id")

	var in struct {
		TestType string `json:"test_type"`
		Name     string `json:"variant_name"`
		MinAge   int    `json:"min_age"`
		MaxAge   int    `json:"max_age"`
		Gender   string `json:"gender"`
		Interests string `json:"interests"`
		Budget   float64 `json:"budget"`
	}
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "body invalido")
	}

	campaign, err := h.campaignUC.Get(c.UserContext(), userID, campaignID)
	if err != nil {
		return mapError(err)
	}

	tokens, err := h.tokenRepo.ListByUser(c.UserContext(), userID)
	if err != nil || len(tokens) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "sem conta Meta")
	}

	tok := tokens[0].EncryptedToken
	adAccountID := tokens[0].AdAccountID

	if in.TestType == "audience" {
		// Create new ad set on Meta with variant targeting
		targeting := map[string]any{}
		if in.MinAge > 0 { targeting["age_min"] = in.MinAge }
		if in.MaxAge > 0 { targeting["age_max"] = in.MaxAge }
		if in.Gender != "" {
			switch in.Gender {
			case "male": targeting["genders"] = []int{1}
			case "female": targeting["genders"] = []int{2}
			default: targeting["genders"] = []int{0}
			}
		}
		targeting["geo_locations"] = map[string]any{"countries": []string{"BR"}}
		targeting["targeting_automation"] = map[string]any{"advantage_audience": 0}
		if in.Interests != "" {
			targeting["flexible_spec"] = []map[string]any{{"interests": []map[string]any{{"name": in.Interests}}}}
		}

		asid, err := h.metaClient.CreateAdSet(c.UserContext(), tok, adAccountID, map[string]any{
			"campaign_id": campaign.MetaCampaignID,
			"name":        in.Name,
			"status":      "PAUSED",
			"billing_event": "IMPRESSIONS",
			"optimization_goal": "REACH",
			"bid_amount":  100,
			"targeting":   targeting,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "criar conjunto: "+err.Error())
		}
		return c.JSON(fiber.Map{"data": fiber.Map{
			"test_type": "audience",
			"ad_set_id": asid,
			"status":    "criado",
			"note":      "Conjunto de teste criado (PAUSED). Ative no Gerenciador.",
		}})
	}

	return fiber.NewError(fiber.StatusBadRequest, "tipo de teste invalido: "+in.TestType)
}

