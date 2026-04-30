package handler

import (
	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
)

type MetaAuthHandler struct {
	tokens     repository.UserTokenRepository
	metaClient metaads.Client
	cfg        *config.Service
}

func NewMetaAuthHandler(tokens repository.UserTokenRepository, metaClient metaads.Client, cfg *config.Service) *MetaAuthHandler {
	return &MetaAuthHandler{tokens: tokens, metaClient: metaClient, cfg: cfg}
}

func (h *MetaAuthHandler) Connect(c *fiber.Ctx) error {
	var body struct {
		AccessToken string `json:"access_token"`
		AdAccountID string `json:"ad_account_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if body.AccessToken == "" || body.AdAccountID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "access_token and ad_account_id are required")
	}

	// Validate token by calling Meta API
	_, err := h.metaClient.GetCampaigns(c.UserContext(), body.AccessToken, body.AdAccountID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid token or ad account")
	}

	// Store encrypted token
	token := &domain.UserToken{
		UserID:         middleware.UserID(c),
		AdAccountID:    body.AdAccountID,
		EncryptedToken: body.AccessToken, // TODO: encrypt with config.Service
	}
	if err := h.tokens.Upsert(c.UserContext(), token); err != nil {
		return mapError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": fiber.Map{
			"ad_account_id": body.AdAccountID,
			"connected":     true,
		},
	})
}

// ListAdAccounts proxies to Meta Graph API to fetch ad accounts for a given access token.
func (h *MetaAuthHandler) ListAdAccounts(c *fiber.Ctx) error {
	token := c.Query("access_token")
	if token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "access_token query param is required")
	}
	accounts, err := h.metaClient.GetAdAccounts(c.UserContext(), token)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": accounts})
}

func (h *MetaAuthHandler) Status(c *fiber.Ctx) error {
	tokens, err := h.tokens.ListByUser(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return mapError(err)
	}

	accounts := make([]fiber.Map, 0, len(tokens))
	for _, t := range tokens {
		accounts = append(accounts, fiber.Map{
			"ad_account_id": t.AdAccountID,
			"connected":     true,
			"expires_at":    t.TokenExpiry,
		})
	}

	return c.JSON(fiber.Map{"data": fiber.Map{"accounts": accounts, "total": len(accounts)}})
}
