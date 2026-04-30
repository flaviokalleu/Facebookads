package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
)

type LocationHandler struct {
	tokenRepo  repository.UserTokenRepository
	metaClient metaads.Client
}

func NewLocationHandler(tokenRepo repository.UserTokenRepository, metaClient metaads.Client) *LocationHandler {
	return &LocationHandler{tokenRepo: tokenRepo, metaClient: metaClient}
}

func (h *LocationHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	if query == "" {
		return fiber.NewError(fiber.StatusBadRequest, "query param q is required")
	}

	// Get user's Meta token to proxy the request
	userID := middleware.UserID(c)
	tokens, err := h.tokenRepo.ListByUser(c.UserContext(), userID)
	if err != nil || len(tokens) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no Meta account connected")
	}
	tok := tokens[0].EncryptedToken

	// Call Meta Location Search API
	endpoint := fmt.Sprintf("https://graph.facebook.com/v25.0/search?type=adgeolocation&location_types=city&q=%s&access_token=%s",
		url.QueryEscape(query), url.QueryEscape(tok))

	resp, err := http.Get(endpoint)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var result struct {
		Data []fiber.Map `json:"data"`
		Error *struct { Message string `json:"message"` } `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if result.Error != nil {
		return fiber.NewError(fiber.StatusBadRequest, result.Error.Message)
	}

	if result.Data == nil {
		result.Data = []fiber.Map{}
	}
	return c.JSON(fiber.Map{"data": result.Data})
}
