package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
	"github.com/facebookads/backend/internal/usecase"
)

type AIActionsHandler struct {
	actions repository.AIActionRepository
	autopilot *usecase.AutoPilotV2
	rules     *usecase.SafetyRulesService
}

func NewAIActionsHandler(
	actions repository.AIActionRepository,
	autopilot *usecase.AutoPilotV2,
	rules *usecase.SafetyRulesService,
) *AIActionsHandler {
	return &AIActionsHandler{actions: actions, autopilot: autopilot, rules: rules}
}

// GET /api/v1/ai/actions?status=pending&limit=N
func (h *AIActionsHandler) List(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	if userID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.Query("limit", "100"))

	var (
		list []*domain.AIAction
		err  error
	)
	if status == "pending" {
		list, err = h.actions.ListPendingByUser(c.UserContext(), userID, limit)
	} else {
		list, err = h.actions.ListByUser(c.UserContext(), userID, status, limit)
	}
	if err != nil {
		return mapError(err)
	}
	if list == nil {
		list = []*domain.AIAction{}
	}
	return c.JSON(fiber.Map{"data": list})
}

// POST /api/v1/ai/actions/:id/approve
func (h *AIActionsHandler) Approve(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	id := c.Params("id")
	action, err := h.actions.GetByID(c.UserContext(), id)
	if err != nil {
		return mapError(err)
	}
	if action.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}
	if action.Status != domain.AIActionStatusPending {
		return fiber.NewError(fiber.StatusConflict, "action not pending")
	}
	if err := h.actions.MarkApproved(c.UserContext(), id); err != nil {
		return mapError(err)
	}
	// Refresh the action and execute against Meta.
	action, err = h.actions.GetByID(c.UserContext(), id)
	if err != nil {
		return mapError(err)
	}
	if err := h.autopilot.ExecuteAction(c.UserContext(), userID, action); err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fiber.Map{"code": "execute_failed", "message": err.Error()},
		})
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"id": id, "status": "executed"}})
}

// POST /api/v1/ai/actions/:id/reject
func (h *AIActionsHandler) Reject(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	id := c.Params("id")
	action, err := h.actions.GetByID(c.UserContext(), id)
	if err != nil {
		return mapError(err)
	}
	if action.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}
	if action.Status != domain.AIActionStatusPending {
		return fiber.NewError(fiber.StatusConflict, "action not pending")
	}
	if err := h.actions.MarkRejected(c.UserContext(), id); err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"id": id, "status": "rejected"}})
}

// POST /api/v1/ai/actions/:id/revert
func (h *AIActionsHandler) Revert(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	id := c.Params("id")
	action, err := h.actions.GetByID(c.UserContext(), id)
	if err != nil {
		return mapError(err)
	}
	if action.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "forbidden")
	}
	if err := h.autopilot.RevertAction(c.UserContext(), userID, action); err != nil {
		if errors.Is(err, domain.ErrValidation) {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error": fiber.Map{"code": "revert_failed", "message": err.Error()},
		})
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"id": id, "status": "reverted"}})
}

// GET /api/v1/ai/safety-rules
func (h *AIActionsHandler) ListSafetyRules(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	effective, overrides, err := h.rules.EffectiveForUser(c.UserContext(), userID)
	if err != nil {
		return mapError(err)
	}
	if overrides == nil {
		overrides = []*domain.AISafetyRule{}
	}
	return c.JSON(fiber.Map{"data": fiber.Map{
		"defaults":  usecase.DefaultSafetyRules,
		"effective": effective,
		"overrides": overrides,
	}})
}

// PUT /api/v1/ai/safety-rules/:rule_key
func (h *AIActionsHandler) UpsertSafetyRule(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	ruleKey := c.Params("rule_key")
	var body struct {
		Value         float64 `json:"value"`
		AccountMetaID *string `json:"account_meta_id,omitempty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if err := h.rules.Upsert(c.UserContext(), userID, ruleKey, body.Value, body.AccountMetaID); err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"rule_key": ruleKey, "value": body.Value, "account_meta_id": body.AccountMetaID}})
}
