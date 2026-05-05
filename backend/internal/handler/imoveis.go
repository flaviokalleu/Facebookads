package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository/postgres"
)

type ImoveisHandler struct {
	repo *postgres.ImovelRepo
}

func NewImoveisHandler(repo *postgres.ImovelRepo) *ImoveisHandler {
	return &ImoveisHandler{repo: repo}
}

var validSegments = map[string]bool{
	"mcmv": true, "medio": true, "alto": true,
	"comercial": true, "terreno": true, "lancamento": true,
}
var validTipologias = map[string]bool{
	"apartamento": true, "casa": true, "terreno": true,
	"sala": true, "galpao": true,
}
var validStatuses = map[string]bool{
	"rascunho": true, "ativo": true, "pausado": true, "vendido": true,
}

type imovelInput struct {
	Nome            string   `json:"nome"`
	Segmento        string   `json:"segmento"`
	Cidade          string   `json:"cidade"`
	Bairro          string   `json:"bairro"`
	PrecoMin        *float64 `json:"preco_min,omitempty"`
	PrecoMax        *float64 `json:"preco_max,omitempty"`
	Quartos         *int     `json:"quartos,omitempty"`
	AreaM2          *float64 `json:"area_m2,omitempty"`
	Tipologia       string   `json:"tipologia"`
	Diferenciais    []string `json:"diferenciais"`
	Fotos           []string `json:"fotos"`
	WhatsAppDestino string   `json:"whatsapp_destino"`
	LinkLanding     string   `json:"link_landing"`
	Status          string   `json:"status"`
}

func (in imovelInput) validate() error {
	if strings.TrimSpace(in.Nome) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nome é obrigatório")
	}
	if !validSegments[in.Segmento] {
		return fiber.NewError(fiber.StatusBadRequest, "segmento inválido (use mcmv, medio, alto, comercial, terreno, lancamento)")
	}
	if in.Tipologia != "" && !validTipologias[in.Tipologia] {
		return fiber.NewError(fiber.StatusBadRequest, "tipologia inválida")
	}
	if in.Status != "" && !validStatuses[in.Status] {
		return fiber.NewError(fiber.StatusBadRequest, "status inválido")
	}
	return nil
}

func (in imovelInput) toDomain(userID string) *domain.Imovel {
	status := in.Status
	if status == "" {
		status = domain.ImovelStatusRascunho
	}
	difer := in.Diferenciais
	if difer == nil {
		difer = []string{}
	}
	fotos := in.Fotos
	if fotos == nil {
		fotos = []string{}
	}
	return &domain.Imovel{
		UserID:          userID,
		Nome:            strings.TrimSpace(in.Nome),
		Segmento:        in.Segmento,
		Cidade:          in.Cidade,
		Bairro:          in.Bairro,
		PrecoMin:        in.PrecoMin,
		PrecoMax:        in.PrecoMax,
		Quartos:         in.Quartos,
		AreaM2:          in.AreaM2,
		Tipologia:       in.Tipologia,
		Diferenciais:    difer,
		Fotos:           fotos,
		WhatsAppDestino: in.WhatsAppDestino,
		LinkLanding:     in.LinkLanding,
		Status:          status,
	}
}

// List handles GET /api/v1/imoveis
func (h *ImoveisHandler) List(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	items, err := h.repo.ListByUser(c.UserContext(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"data": items})
}

// Get handles GET /api/v1/imoveis/:id
func (h *ImoveisHandler) Get(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	id := c.Params("id")
	im, err := h.repo.GetByID(c.UserContext(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "imóvel não encontrado")
	}
	if im.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "imóvel pertence a outro usuário")
	}
	return c.JSON(fiber.Map{"data": im})
}

// Create handles POST /api/v1/imoveis
func (h *ImoveisHandler) Create(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	var in imovelInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json inválido")
	}
	if err := in.validate(); err != nil {
		return err
	}
	im := in.toDomain(userID)
	if err := h.repo.Create(c.UserContext(), im); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": im})
}

// Update handles PATCH /api/v1/imoveis/:id
func (h *ImoveisHandler) Update(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	id := c.Params("id")
	existing, err := h.repo.GetByID(c.UserContext(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "imóvel não encontrado")
	}
	if existing.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "imóvel pertence a outro usuário")
	}
	var in imovelInput
	if err := c.BodyParser(&in); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "json inválido")
	}
	if err := in.validate(); err != nil {
		return err
	}
	im := in.toDomain(userID)
	im.ID = id
	if err := h.repo.Update(c.UserContext(), im); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	updated, _ := h.repo.GetByID(c.UserContext(), id)
	return c.JSON(fiber.Map{"data": updated})
}

// Delete handles DELETE /api/v1/imoveis/:id
func (h *ImoveisHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	id := c.Params("id")
	existing, err := h.repo.GetByID(c.UserContext(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "imóvel não encontrado")
	}
	if existing.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "imóvel pertence a outro usuário")
	}
	if err := h.repo.Delete(c.UserContext(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"deleted": true}})
}
