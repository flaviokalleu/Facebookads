package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
)

type TokenHandler struct {
	tokenRepo repository.UserTokenRepository
	cfg       *config.Service
}

func NewTokenHandler(tokenRepo repository.UserTokenRepository, cfg *config.Service) *TokenHandler {
	return &TokenHandler{tokenRepo: tokenRepo, cfg: cfg}
}

// Refresh troca token atual por um longo (60 dias) usando app_secret
func (h *TokenHandler) Refresh(c *fiber.Ctx) error {
	userID := middleware.UserID(c)

	appID := h.cfg.Get("meta.app_id")
	appSecret := h.cfg.GetSecret("meta.app_secret")
	if appID == "" || appSecret == "" {
		return fiber.NewError(fiber.StatusBadRequest, "meta.app_id ou meta.app_secret não configurados")
	}

	tokens, err := h.tokenRepo.ListByUser(c.UserContext(), userID)
	if err != nil || len(tokens) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "nenhuma conta Meta conectada")
	}

	var results []fiber.Map
	for _, t := range tokens {
		// Troca o token atual por um longo
		refreshURL := fmt.Sprintf("https://graph.facebook.com/v25.0/oauth/access_token?grant_type=fb_exchange_token&client_id=%s&client_secret=%s&fb_exchange_token=%s",
			url.QueryEscape(appID), url.QueryEscape(appSecret), url.QueryEscape(t.EncryptedToken))

		resp, err := http.Get(refreshURL)
		if err != nil {
			results = append(results, fiber.Map{
				"account": t.AdAccountID, "status": "erro", "error": err.Error(),
			})
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var result struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
			Error       *struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		json.Unmarshal(body, &result)

		if result.Error != nil {
			results = append(results, fiber.Map{
				"account": t.AdAccountID, "status": "erro", "error": result.Error.Message,
			})
			continue
		}
		if result.AccessToken == "" {
			results = append(results, fiber.Map{
				"account": t.AdAccountID, "status": "erro", "error": "token vazio na resposta",
			})
			continue
		}

		// Atualiza o token no banco
		t.EncryptedToken = result.AccessToken
		if err := h.tokenRepo.Upsert(c.UserContext(), t); err != nil {
			results = append(results, fiber.Map{
				"account": t.AdAccountID, "status": "erro", "error": err.Error(),
			})
			continue
		}

		dias := result.ExpiresIn / 86400
		results = append(results, fiber.Map{
			"account": t.AdAccountID, "status": "renovado",
			"expira_em_dias": dias,
		})
	}

	return c.JSON(fiber.Map{"data": fiber.Map{"results": results}})
}
