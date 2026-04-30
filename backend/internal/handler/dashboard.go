package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/usecase"
)

type DashboardHandler struct {
	uc *usecase.DashboardUseCase
}

func NewDashboardHandler(uc *usecase.DashboardUseCase) *DashboardHandler {
	return &DashboardHandler{uc: uc}
}

func (h *DashboardHandler) Summary(c *fiber.Ctx) error {
	summary, err := h.uc.Summary(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": summary})
}

func (h *DashboardHandler) Campaigns(c *fiber.Ctx) error {
	campaigns, err := h.uc.CampaignsWithMetrics(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": campaigns, "meta": fiber.Map{"total": len(campaigns)}})
}

func (h *DashboardHandler) CampaignInsights(c *fiber.Ctx) error {
	from := time.Now().AddDate(0, 0, -30)
	to := time.Now()
	rows, err := h.uc.CampaignInsights(c.UserContext(), c.Params("id"), from, to)
	if err != nil {
		return mapError(err)
	}
	if rows == nil {
		rows = []*domain.CampaignInsight{}
	}
	return c.JSON(fiber.Map{"data": rows, "meta": fiber.Map{"campaign_id": c.Params("id")}})
}

func (h *DashboardHandler) TopCreatives(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"data": fiber.Map{"top": []any{}, "bottom": []any{}}})
}

func (h *DashboardHandler) Anomalies(c *fiber.Ctx) error {
	anomalies, err := h.uc.Anomalies(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return mapError(err)
	}
	if anomalies == nil {
		anomalies = []*domain.Anomaly{}
	}
	return c.JSON(fiber.Map{"data": anomalies, "meta": fiber.Map{"total": len(anomalies)}})
}

func (h *DashboardHandler) BudgetAdvisor(c *fiber.Ctx) error {
	suggestions, err := h.uc.BudgetSuggestions(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return mapError(err)
	}
	if suggestions == nil {
		suggestions = []*domain.BudgetSuggestion{}
	}
	return c.JSON(fiber.Map{"data": suggestions})
}

func (h *DashboardHandler) Recommendations(c *fiber.Ctx) error {
	recs, err := h.uc.Recommendations(c.UserContext(), middleware.UserID(c))
	if err != nil {
		return mapError(err)
	}
	if recs == nil {
		recs = []*domain.Recommendation{}
	}
	return c.JSON(fiber.Map{"data": recs})
}
