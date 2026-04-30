package handler

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/orchestrator"
	"github.com/facebookads/backend/internal/repository"
)

type AdminHandler struct {
	cfg      *config.Service
	llmUsage repository.LLMUsageRepository
	router   *ai.Router
	sched    *orchestrator.Scheduler
}

func NewAdminHandler(cfg *config.Service, llmUsage repository.LLMUsageRepository, router *ai.Router, sched *orchestrator.Scheduler) *AdminHandler {
	return &AdminHandler{cfg: cfg, llmUsage: llmUsage, router: router, sched: sched}
}

func (h *AdminHandler) ListConfig(c *fiber.Ctx) error {
	entries, err := h.cfg.List(c.UserContext())
	if err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": entries})
}

func (h *AdminHandler) SetConfig(c *fiber.Ctx) error {
	key := c.Params("key")
	var body struct {
		Value    string `json:"value"`
		IsSecret bool   `json:"is_secret"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	if err := h.cfg.Set(c.UserContext(), key, body.Value, body.IsSecret); err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"key": key, "updated": true}})
}

func (h *AdminHandler) AIUsage(c *fiber.Ctx) error {
	from := time.Now().AddDate(0, 0, -30)
	to := time.Now()
	summary, err := h.llmUsage.SummaryByProvider(c.UserContext(), "", from, to)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": summary})
}

func (h *AdminHandler) AIUsageDaily(c *fiber.Ctx) error {
	daily, err := h.llmUsage.DailyCost(c.UserContext(), "", 30)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(fiber.Map{"data": daily})
}

func (h *AdminHandler) Providers(c *fiber.Ctx) error {
	if h.router == nil {
		return c.JSON(fiber.Map{"data": []any{}})
	}
	infos := h.router.ProviderInfos()
	return c.JSON(fiber.Map{"data": infos})
}

func (h *AdminHandler) TestProvider(c *fiber.Ctx) error {
	name := c.Params("name")
	if h.router == nil {
		return c.JSON(fiber.Map{"data": fiber.Map{"name": name, "available": false, "error": "no router configured"}})
	}

	available := false
	var errMsg string
	for _, p := range h.router.ProviderInfos() {
		if strings.EqualFold(p.Name, name) || strings.EqualFold(p.ModelID, name) {
			ctx, cancel := context.WithTimeout(c.UserContext(), 5*time.Second)
			defer cancel()
			available = ai.CheckAvailable(ctx, nil)
			if !available {
				errMsg = "provider unavailable or timeout"
			}
			return c.JSON(fiber.Map{"data": fiber.Map{"name": name, "available": available, "error": errMsg}})
		}
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"name": name, "available": false, "error": "provider not found"}})
}

func (h *AdminHandler) RoutingTable(c *fiber.Ctx) error {
	if h.router == nil {
		return c.JSON(fiber.Map{"data": fiber.Map{}})
	}
	return c.JSON(fiber.Map{"data": h.router.RoutingTable()})
}

func (h *AdminHandler) OverrideRouting(c *fiber.Ctx) error {
	if h.router == nil {
		return fiber.NewError(fiber.StatusServiceUnavailable, "no router configured")
	}
	var body struct {
		Task   string   `json:"task"`
		Models []string `json:"models"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	h.router.OverrideRouting(ai.TaskType(body.Task), body.Models)
	return c.JSON(fiber.Map{"data": fiber.Map{"task": body.Task, "models": body.Models, "updated": true}})
}

func (h *AdminHandler) SchedulerStatus(c *fiber.Ctx) error {
	if h.sched == nil {
		return c.JSON(fiber.Map{"data": []any{}})
	}
	return c.JSON(fiber.Map{"data": h.sched.Status()})
}
