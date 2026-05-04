package handler

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
	"github.com/facebookads/backend/internal/usecase"
)

// MetaConnectV2Handler implements the new self-discovering connect flow:
// user pastes app_id + app_secret + access_token, backend exchanges for a
// long-lived token, validates it, then enumerates the entire BM hierarchy.
type MetaConnectV2Handler struct {
	sync     *usecase.SyncMetaAccount
	creds    repository.AppCredentialRepository
	tokens   repository.MetaTokenRepository
	bms      repository.BusinessManagerRepository
	accs     repository.MetaAdAccountRepository
	pages    repository.MetaPageRepository
	pixels   repository.MetaPixelRepository
	meta     metaads.Client
	cfg      *config.Service
}

func NewMetaConnectV2Handler(
	sync *usecase.SyncMetaAccount,
	creds repository.AppCredentialRepository,
	tokens repository.MetaTokenRepository,
	bms repository.BusinessManagerRepository,
	accs repository.MetaAdAccountRepository,
	pages repository.MetaPageRepository,
	pixels repository.MetaPixelRepository,
	meta metaads.Client,
	cfg *config.Service,
) *MetaConnectV2Handler {
	return &MetaConnectV2Handler{
		sync: sync, creds: creds, tokens: tokens, bms: bms,
		accs: accs, pages: pages, pixels: pixels, meta: meta, cfg: cfg,
	}
}

type connectV2Request struct {
	AppID       string `json:"app_id"`
	AppSecret   string `json:"app_secret"`
	AccessToken string `json:"access_token"`
}

// Connect handles POST /api/v1/auth/meta/connect-v2
func (h *MetaConnectV2Handler) Connect(c *fiber.Ctx) error {
	var body connectV2Request
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	body.AppID = strings.TrimSpace(body.AppID)
	body.AppSecret = strings.TrimSpace(body.AppSecret)
	body.AccessToken = strings.TrimSpace(body.AccessToken)
	if body.AppID == "" || body.AppSecret == "" || body.AccessToken == "" {
		return fiber.NewError(fiber.StatusBadRequest, "app_id, app_secret and access_token are required")
	}

	userID := middleware.UserID(c)
	ctx := c.UserContext()

	// 1) Exchange short-lived → long-lived (60d).
	longToken, expiresIn, err := h.meta.ExchangeForLongLived(ctx, body.AppID, body.AppSecret, body.AccessToken)
	if err != nil {
		slog.Warn("meta connect-v2: exchange failed", "user_id", userID, "err", err)
		return fiber.NewError(fiber.StatusBadRequest, "token exchange failed: "+err.Error())
	}

	// 2) Debug token to validate + extract user_id, scopes, expires_at.
	info, err := h.meta.DebugToken(ctx, longToken, body.AppID, body.AppSecret)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "debug_token failed: "+err.Error())
	}
	if !info.IsValid {
		return fiber.NewError(fiber.StatusBadRequest, "token reported as invalid by Meta")
	}

	// 3) Persist app credentials (encrypted secret).
	encSecret, err := h.cfg.Encrypt(body.AppSecret)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "encrypt app_secret failed")
	}
	cred := &domain.AppCredential{
		UserID: userID, AppID: body.AppID, EncryptedAppSecret: encSecret,
	}
	if err := h.creds.Upsert(ctx, cred); err != nil {
		return mapError(err)
	}

	// 4) Persist meta token (encrypted).
	encToken, err := h.cfg.Encrypt(longToken)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "encrypt token failed")
	}
	expAt := info.ExpiresAt
	if expAt.IsZero() && expiresIn > 0 {
		expAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	}
	var expPtr *time.Time
	if !expAt.IsZero() {
		expPtr = &expAt
	}
	tok := &domain.MetaToken{
		UserID:         userID,
		AppID:          body.AppID,
		MetaUserID:     info.UserID,
		EncryptedToken: encToken,
		TokenType:      domain.MetaTokenTypeUser,
		Scopes:         info.Scopes,
		ExpiresAt:      expPtr,
		IsActive:       true,
	}
	if err := h.tokens.Upsert(ctx, tok); err != nil {
		return mapError(err)
	}

	// 5) Run full sync synchronously.
	if err := h.sync.Run(ctx, userID); err != nil {
		slog.Warn("meta connect-v2: sync error (partial data may have been persisted)", "user_id", userID, "err", err)
	}

	// 6) Reply with summary.
	bms, _ := h.bms.ListByUser(ctx, userID)
	accs, _ := h.accs.ListByUser(ctx, userID)
	resp := fiber.Map{
		"meta_user_id":     info.UserID,
		"expires_at":       expPtr,
		"scopes":           info.Scopes,
		"businesses_count": len(bms),
		"accounts_count":   len(accs),
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": resp})
}

// Sync re-triggers the snapshot from the UI.
func (h *MetaConnectV2Handler) Sync(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	if err := h.sync.Run(c.UserContext(), userID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"status": "ok"}})
}

// Tree returns the persisted hierarchy for the user — fast, snapshot-only.
// Shape:
//
//	{ "data": {
//	    "businesses": [
//	      { "meta_id":"...", "name":"...", "accounts":[...], "pages":[...], "pixels":[...] }
//	    ],
//	    "personal_accounts":[...]
//	}}
func (h *MetaConnectV2Handler) Tree(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	ctx := c.UserContext()

	bms, err := h.bms.ListByUser(ctx, userID)
	if err != nil {
		return mapError(err)
	}

	type accountDTO struct {
		MetaID        string  `json:"meta_id"`
		Name          string  `json:"name"`
		Currency      string  `json:"currency"`
		AccountStatus int     `json:"account_status"`
		Balance       float64 `json:"balance"`
		AmountSpent   float64 `json:"amount_spent"`
		SpendCap      float64 `json:"spend_cap"`
		AccessKind    string  `json:"access_kind"`
	}
	type pageDTO struct {
		MetaID   string `json:"meta_id"`
		Name     string `json:"name"`
		Category string `json:"category"`
		FanCount int64  `json:"fan_count"`
	}
	type pixelDTO struct {
		MetaID    string     `json:"meta_id"`
		Name      string     `json:"name"`
		IsActive  bool       `json:"is_active"`
		LastFired *time.Time `json:"last_fired,omitempty"`
	}
	type bmDTO struct {
		MetaID             string       `json:"meta_id"`
		Name               string       `json:"name"`
		VerificationStatus string       `json:"verification_status"`
		Vertical           string       `json:"vertical"`
		Accounts           []accountDTO `json:"accounts"`
		Pages              []pageDTO    `json:"pages"`
		Pixels             []pixelDTO   `json:"pixels"`
	}

	mapAccount := func(a *domain.MetaAdAccount) accountDTO {
		return accountDTO{
			MetaID: a.MetaID, Name: a.Name, Currency: a.Currency,
			AccountStatus: a.AccountStatus, Balance: a.Balance,
			AmountSpent: a.AmountSpent, SpendCap: a.SpendCap,
			AccessKind: a.AccessKind,
		}
	}

	bmDTOs := make([]bmDTO, 0, len(bms))
	for _, b := range bms {
		entry := bmDTO{
			MetaID: b.MetaID, Name: b.Name,
			VerificationStatus: b.VerificationStatus,
			Vertical:           b.Vertical,
			Accounts:           []accountDTO{},
			Pages:              []pageDTO{},
			Pixels:             []pixelDTO{},
		}
		if accs, err := h.accs.ListByBM(ctx, b.MetaID); err == nil {
			for _, a := range accs {
				entry.Accounts = append(entry.Accounts, mapAccount(a))
			}
		}
		if pages, err := h.pages.ListByBM(ctx, b.MetaID); err == nil {
			for _, p := range pages {
				entry.Pages = append(entry.Pages, pageDTO{
					MetaID: p.MetaID, Name: p.Name,
					Category: p.Category, FanCount: p.FanCount,
				})
			}
		}
		if pixels, err := h.pixels.ListByBM(ctx, b.MetaID); err == nil {
			for _, p := range pixels {
				entry.Pixels = append(entry.Pixels, pixelDTO{
					MetaID: p.MetaID, Name: p.Name,
					IsActive: p.IsActive, LastFired: p.LastFired,
				})
			}
		}
		bmDTOs = append(bmDTOs, entry)
	}

	personal, _ := h.accs.ListPersonalByUser(ctx, userID)
	personalDTOs := make([]accountDTO, 0, len(personal))
	for _, a := range personal {
		personalDTOs = append(personalDTOs, mapAccount(a))
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"businesses":        bmDTOs,
		"personal_accounts": personalDTOs,
	}})
}
