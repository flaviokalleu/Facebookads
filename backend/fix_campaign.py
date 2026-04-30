import re

with open('internal/handler/campaign.go', 'r') as f:
    content = f.read()

# Find CreativeInsights to determine boundary
idx = content.find('func (h *CampaignHandler) CreativeInsights')
if idx < 0:
    idx = content.find('func (h *CampaignHandler) CreateFromPrompt')

# Keep everything before CreateFromPrompt
keep = content[:idx]

# Add clean implementation
new_code = '''func (h *CampaignHandler) CreateFromPrompt(c *fiber.Ctx) error {
	var body struct {
		Prompt string `json:"prompt"`
	}
	if err := c.BodyParser(&body); err != nil || body.Prompt == "" {
		return fiber.NewError(fiber.StatusBadRequest, "campo prompt é obrigatório")
	}

	resp, err := h.aiRouter.Complete(c.UserContext(), ai.TaskCreativeAnalysis, ai.CompletionRequest{
		SystemPrompt: "From user request, return JSON with: name, objective, daily_budget, min_age, max_age, gender, interests, cities, country. Valid objectives: OUTCOME_ENGAGEMENT,OUTCOME_LEADS,OUTCOME_SALES,OUTCOME_TRAFFIC. Default: objective=OUTCOME_ENGAGEMENT, budget=50, min_age=18, max_age=65, gender=all, country=BR. Return ONLY JSON.",
		UserPrompt:   body.Prompt,
		MaxTokens:    500,
		Temperature:  0.2,
		JSONMode:     true,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "IA falhou: "+err.Error())
	}

	var config struct {
		Name        string  `json:"name"`
		Objective   string  `json:"objective"`
		DailyBudget float64 `json:"daily_budget"`
		MinAge      int     `json:"min_age"`
		MaxAge      int     `json:"max_age"`
		Gender      string  `json:"gender"`
		Interests   string  `json:"interests"`
		Cities      string  `json:"cities"`
		Country     string  `json:"country"`
	}
	cleaned := cleanJSON(resp.Content)
	if err := json.Unmarshal([]byte(cleaned), &config); err != nil {
		return c.JSON(fiber.Map{"data": fiber.Map{
			"status": "precisa_ajuste",
			"raw":    cleaned,
		}})
	}
	if config.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "IA não conseguiu extrair o nome da campanha")
	}
	if config.MinAge == 0 { config.MinAge = 18 }
	if config.MaxAge == 0 { config.MaxAge = 65 }
	if config.Country == "" { config.Country = "BR" }
	if config.DailyBudget <= 0 { config.DailyBudget = 50 }

	budget := &config.DailyBudget
	campaign, err := h.uc.Create(c.UserContext(), middleware.UserID(c), usecase.CreateCampaignInput{
		Name:        config.Name,
		Objective:   config.Objective,
		AdAccountID: "1386606178278837",
		DailyBudget: budget,
	})
	if err != nil {
		return mapError(err)
	}

	// Create full funnel on Meta Ads
	var metaCampaignID, metaAdSetID, metaAdID string
	metaNote := "Campanha criada localmente."
	if tokens, err := h.tokenRepo.ListByUser(c.UserContext(), middleware.UserID(c)); err == nil && len(tokens) > 0 {
		tok := tokens[0].EncryptedToken
		adAccountID := tokens[0].AdAccountID
		budgetCents := int64(config.DailyBudget * 100)

		// Geo targeting
		geoLocations := map[string]any{"countries": []string{config.Country}}
		if config.Cities != "" {
			cities := strings.Split(config.Cities, ",")
			var customLoc []map[string]any
			for _, c := range cities {
				c = strings.TrimSpace(c)
				if c != "" {
					customLoc = append(customLoc, map[string]any{"name": c, "radius": 10, "distance_unit": "km"})
				}
			}
			if len(customLoc) > 0 {
				geoLocations["custom_locations"] = customLoc
			}
		}

		// Gender
		var genders []int
		switch config.Gender {
		case "male": genders = []int{1}
		case "female": genders = []int{2}
		default: genders = []int{0}
		}

		// 1. Create Campaign
		mid, cerr := h.metaClient.CreateCampaign(c.UserContext(), tok, adAccountID, map[string]any{
			"name":                config.Name,
			"objective":           mapObjective(config.Objective),
			"status":              "PAUSED",
			"daily_budget":        budgetCents,
			"special_ad_categories": []string{"NONE"},
		})
		if cerr != nil {
			slog.Error("meta: campaign failed", "err", cerr)
			metaNote = "Erro ao criar campanha: " + cerr.Error()
		} else {
			metaCampaignID = mid

			// 2. Create Ad Set
			targeting := map[string]any{
				"age_min":       config.MinAge,
				"age_max":       config.MaxAge,
				"genders":       genders,
				"geo_locations": geoLocations,
			}
			if config.Interests != "" {
				targeting["flexible_spec"] = []map[string]any{{
					"interests": []map[string]any{{"name": config.Interests}},
				}}
			}

			asid, aerr := h.metaClient.CreateAdSet(c.UserContext(), tok, adAccountID, map[string]any{
				"campaign_id":       mid,
				"name":              config.Name + " - Conjunto",
				"status":            "PAUSED",
				"daily_budget":      budgetCents,
				"billing_event":     "IMPRESSIONS",
				"optimization_goal": "REACH",
				"targeting":         targeting,
			})
			if aerr != nil {
				slog.Error("meta: ad set failed", "err", aerr)
				metaNote = "Campanha criada, mas conjunto falhou."
			} else {
				metaAdSetID = asid

				// 3. Create Ad
				cta := "WHATSAPP_MESSAGE"
				if strings.Contains(strings.ToLower(config.Objective), "lead") {
					cta = "SIGN_UP"
				}
				adid, aderr := h.metaClient.CreateAd(c.UserContext(), tok, adAccountID, map[string]any{
					"name":    config.Name + " - Anúncio",
					"adset_id": asid,
					"status":  "PAUSED",
					"creative": map[string]any{
						"title":              config.Name,
						"body":               "Confira " + config.Name + ". Saiba mais!",
						"call_to_action_type": cta,
					},
				})
				if aderr != nil {
					slog.Error("meta: ad failed", "err", aderr)
					metaNote = "Campanha e conjunto criados, anúncio falhou."
				} else {
					metaAdID = adid
					metaNote = "Campanha + Conjunto + Anúncio criados no Meta Ads (PAUSED)."
				}
			}
		}
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"campaign":         campaign,
		"meta_campaign_id": metaCampaignID,
		"meta_ad_set_id":   metaAdSetID,
		"meta_ad_id":       metaAdID,
		"model_used":       resp.Provider + "/" + resp.ModelUsed,
		"meta_status": fiber.Map{
			"created": metaCampaignID != "",
			"note":    metaNote,
		},
	}})
}

func (h *CampaignHandler) CreativeInsights(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"data": []any{}})
}

func toPtr(v *float64, def float64) float64 {
	if v != nil { return *v }
	return def
}

func mapObjective(obj string) string {
	m := map[string]string{
		"CONVERSIONS": "OUTCOME_ENGAGEMENT",
		"LEADS": "OUTCOME_LEADS",
		"SALES": "OUTCOME_SALES",
		"TRAFFIC": "OUTCOME_TRAFFIC",
		"REACH": "OUTCOME_AWARENESS",
		"ENGAGEMENT": "OUTCOME_ENGAGEMENT",
		"BRAND_AWARENESS": "OUTCOME_AWARENESS",
		"APP_INSTALLS": "OUTCOME_APP_PROMOTION",
	}
	if v, ok := m[obj]; ok { return v }
	return obj
}

func cleanJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	return strings.TrimSpace(raw)
}
'''

with open('internal/handler/campaign.go', 'w') as f:
    f.write(keep + new_code)

print("Done")
