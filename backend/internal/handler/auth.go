package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/usecase"
)

type AuthHandler struct {
	uc *usecase.AuthUseCase
}

func NewAuthHandler(uc *usecase.AuthUseCase) *AuthHandler {
	return &AuthHandler{uc: uc}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var in usecase.RegisterInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	out, err := h.uc.Register(c.UserContext(), in)
	if err != nil {
		return mapError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": out})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var in usecase.LoginInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	out, err := h.uc.Login(c.UserContext(), in)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	user, err := h.uc.GetUser(c.UserContext(), userID)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": user})
}

// mapError converts domain errors to HTTP errors (sanitized — no internal details exposed).
func mapError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, "not_found")
	case errors.Is(err, domain.ErrUnauthorized):
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	case errors.Is(err, domain.ErrForbidden):
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	case errors.Is(err, domain.ErrConflict):
		return fiber.NewError(fiber.StatusConflict, "conflict")
	case errors.Is(err, domain.ErrValidation):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, "internal_error")
	}
}
