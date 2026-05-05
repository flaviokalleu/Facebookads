package handler

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/sync/errgroup"

	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
)

// PublicosHandler aggregates Meta Custom Audiences (and Lookalikes) across all
// of the user's accounts. Pulls live with a short in-memory cache to keep the
// page snappy without hammering the Graph API.
type PublicosHandler struct {
	tokens   repository.MetaTokenRepository
	accounts repository.MetaAdAccountRepository
	meta     metaads.Client

	mu    sync.Mutex
	cache map[string]publicosCacheEntry
}

type publicosCacheEntry struct {
	rows      []audienceRow
	expiresAt time.Time
}

type audienceRow struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Subtype            string `json:"subtype"`
	SubtypeLabel       string `json:"subtype_label"`
	Description        string `json:"description"`
	CountLow           int64  `json:"count_low"`
	CountHigh          int64  `json:"count_high"`
	DeliveryStatusCode int    `json:"delivery_status_code"`
	DeliveryStatusText string `json:"delivery_status_text"`
	OperationStatus    string `json:"operation_status"`
	TimeCreated        int64  `json:"time_created"`
	TimeUpdated        int64  `json:"time_updated"`
	AccountMetaID      string `json:"account_meta_id"`
	AccountName        string `json:"account_name"`
	BMName             string `json:"bm_name"`
}

func NewPublicosHandler(
	tokens repository.MetaTokenRepository,
	accounts repository.MetaAdAccountRepository,
	meta metaads.Client,
) *PublicosHandler {
	return &PublicosHandler{
		tokens: tokens, accounts: accounts, meta: meta,
		cache: make(map[string]publicosCacheEntry),
	}
}

// List handles GET /api/v1/publicos
// Pulls every active account's custom audiences and merges. Cache: 10min.
func (h *PublicosHandler) List(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	ctx := c.UserContext()

	// Cache check.
	h.mu.Lock()
	if entry, ok := h.cache[userID]; ok && time.Now().Before(entry.expiresAt) {
		rows := entry.rows
		h.mu.Unlock()
		return c.JSON(fiber.Map{"data": fiber.Map{"rows": rows, "cached_until": entry.expiresAt}})
	}
	h.mu.Unlock()

	tok, err := h.tokens.GetActiveByUser(ctx, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "nenhum token Meta ativo")
	}
	access := tok.PlainToken

	accs, err := h.accounts.ListByUser(ctx, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	type accInfo struct {
		MetaID, Name, BMName string
	}
	infoByID := make(map[string]accInfo, len(accs))
	for _, a := range accs {
		bm := ""
		if a.BMID != nil {
			// BMID is the local UUID; we'd need a join. For now skip name lookup —
			// audience cards link back to the account anyway.
		}
		infoByID[a.MetaID] = accInfo{MetaID: a.MetaID, Name: a.Name, BMName: bm}
	}

	out := make([]audienceRow, 0, 64)
	var outMu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(6)
	for _, a := range accs {
		if a.AccountStatus != 1 {
			continue
		}
		acc := a
		g.Go(func() error {
			audiences, err := h.meta.GetCustomAudiences(gctx, access, acc.MetaID)
			if err != nil {
				slog.Warn("publicos: account failed", "account", acc.MetaID, "err", err)
				return nil // tolerate per-account failure
			}
			converted := make([]audienceRow, 0, len(audiences))
			for _, ca := range audiences {
				converted = append(converted, audienceRow{
					ID:                 ca.ID,
					Name:               ca.Name,
					Subtype:            ca.Subtype,
					SubtypeLabel:       subtypeLabel(ca.Subtype),
					Description:        ca.Description,
					CountLow:           ca.ApproximateCountLowerBound,
					CountHigh:          ca.ApproximateCountUpperBound,
					DeliveryStatusCode: ca.DeliveryStatus.Code,
					DeliveryStatusText: ca.DeliveryStatus.Description,
					OperationStatus:    ca.OperationStatus.Description,
					TimeCreated:        ca.TimeCreated,
					TimeUpdated:        ca.TimeUpdated,
					AccountMetaID:      acc.MetaID,
					AccountName:        acc.Name,
				})
			}
			outMu.Lock()
			out = append(out, converted...)
			outMu.Unlock()
			return nil
		})
	}
	_ = g.Wait()

	// Sort: bigger audiences first.
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CountHigh > out[j].CountHigh
	})

	expires := time.Now().Add(10 * time.Minute)
	h.mu.Lock()
	h.cache[userID] = publicosCacheEntry{rows: out, expiresAt: expires}
	h.mu.Unlock()

	return c.JSON(fiber.Map{"data": fiber.Map{
		"rows":         out,
		"cached_until": expires,
	}})
}

func subtypeLabel(s string) string {
	return map[string]string{
		"CUSTOM":              "Lista personalizada",
		"WEBSITE":             "Visitantes do site",
		"APP":                 "Usuários do app",
		"OFFLINE_CONVERSION":  "Conversões offline",
		"ENGAGEMENT":          "Quem engajou",
		"VIDEO":               "Quem viu vídeo",
		"LOOKALIKE":           "Sósia (Lookalike)",
		"FOX":                 "Catálogo (DPA)",
		"BAG_OF_ACCOUNTS":     "Lista de contas",
		"LEAD_AD":             "Quem preencheu lead form",
		"REGULATED_CATEGORIES": "Setor regulado",
		"PARTNER":             "De parceiro",
	}[s]
}

// Make sure we don't import context twice indirectly; keep a sentinel use.
var _ = context.Background
