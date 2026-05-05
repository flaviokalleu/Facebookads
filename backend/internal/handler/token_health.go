package handler

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/domain"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/repository"
	"github.com/facebookads/backend/internal/repository/postgres"
)

// TokenHealthHandler exposes Meta token state and refresh for the F1 flow
// (meta_tokens + app_credentials tables, not the legacy user_tokens).
type TokenHealthHandler struct {
	tokens     repository.MetaTokenRepository
	creds      repository.AppCredentialRepository
	credsRepo  *postgres.AppCredentialRepo // need this for DecryptAppSecret
	meta       metaads.Client
	cfg        *config.Service
}

func NewTokenHealthHandler(
	tokens repository.MetaTokenRepository,
	creds repository.AppCredentialRepository,
	credsRepo *postgres.AppCredentialRepo,
	meta metaads.Client,
	cfg *config.Service,
) *TokenHealthHandler {
	return &TokenHealthHandler{tokens: tokens, creds: creds, credsRepo: credsRepo, meta: meta, cfg: cfg}
}

// Health handles GET /api/v1/auth/meta/token/health
// Returns days remaining, scopes, validity. Does a fresh debug_token round-trip
// against Meta when possible — keeps the UI honest about whether the token
// still works.
func (h *TokenHealthHandler) Health(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	ctx := c.UserContext()

	tok, err := h.tokens.GetActiveByUser(ctx, userID)
	if err != nil {
		return c.JSON(fiber.Map{"data": fiber.Map{
			"connected": false,
		}})
	}

	resp := fiber.Map{
		"connected":     true,
		"meta_user_id":  tok.MetaUserID,
		"app_id":        tok.AppID,
		"token_type":    tok.TokenType,
		"scopes":        tok.Scopes,
		"is_active":     tok.IsActive,
		"last_refresh":  tok.LastRefresh,
	}
	if tok.ExpiresAt != nil {
		resp["expires_at"] = tok.ExpiresAt
		resp["days_remaining"] = int(time.Until(*tok.ExpiresAt).Hours() / 24)
	} else {
		resp["expires_at"] = nil
		resp["days_remaining"] = nil // system_user tokens never expire
	}

	// Live verification via debug_token when we have the app credentials.
	cred, err := h.creds.GetByUserAndAppID(ctx, userID, tok.AppID)
	if err == nil {
		appSecret, derr := h.credsRepo.DecryptAppSecret(cred)
		if derr == nil {
			info, ierr := h.meta.DebugToken(ctx, tok.PlainToken, tok.AppID, appSecret)
			if ierr == nil && info != nil {
				resp["live_valid"] = info.IsValid
				resp["live_scopes"] = info.Scopes
				if !info.ExpiresAt.IsZero() {
					resp["live_expires_at"] = info.ExpiresAt
					resp["days_remaining"] = int(time.Until(info.ExpiresAt).Hours() / 24)
				}
			} else if ierr != nil {
				slog.Warn("token health: debug_token failed", "err", ierr)
				resp["live_valid"] = false
				resp["live_error"] = ierr.Error()
			}
		}
	}

	return c.JSON(fiber.Map{"data": resp})
}

// Refresh handles POST /api/v1/auth/meta/token/refresh
// Re-runs ExchangeForLongLived against the stored short-lived (or expiring
// long-lived) token, persists the new one. Returns the new expiration.
func (h *TokenHealthHandler) Refresh(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	ctx := c.UserContext()

	tok, err := h.tokens.GetActiveByUser(ctx, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "nenhum token Meta ativo. Refazer onboarding.")
	}
	cred, err := h.creds.GetByUserAndAppID(ctx, userID, tok.AppID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "credenciais do app não encontradas. Refazer onboarding.")
	}
	appSecret, err := h.credsRepo.DecryptAppSecret(cred)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "decrypt app_secret: "+err.Error())
	}

	newToken, expiresIn, err := h.meta.ExchangeForLongLived(ctx, tok.AppID, appSecret, tok.PlainToken)
	if err != nil {
		slog.Warn("token refresh: exchange failed", "user", userID, "err", err)
		return fiber.NewError(fiber.StatusBadGateway, "Meta recusou: "+err.Error())
	}

	encNew, err := h.cfg.Encrypt(newToken)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "encrypt: "+err.Error())
	}

	expAt := time.Now().Add(time.Duration(expiresIn) * time.Second)
	tok.EncryptedToken = encNew
	tok.ExpiresAt = &expAt
	tok.LastRefresh = time.Now()
	tok.IsActive = true
	if expiresIn == 0 {
		// Some flows return zero — fall back to debug_token to learn the real expiry.
		if info, derr := h.meta.DebugToken(ctx, newToken, tok.AppID, appSecret); derr == nil && info != nil {
			if !info.ExpiresAt.IsZero() {
				tok.ExpiresAt = &info.ExpiresAt
			} else {
				tok.ExpiresAt = nil
			}
			if len(info.Scopes) > 0 {
				tok.Scopes = info.Scopes
			}
		}
	}

	if err := h.tokens.Upsert(ctx, &domain.MetaToken{
		UserID:         tok.UserID,
		AppID:          tok.AppID,
		MetaUserID:     tok.MetaUserID,
		EncryptedToken: tok.EncryptedToken,
		TokenType:      tok.TokenType,
		Scopes:         tok.Scopes,
		ExpiresAt:      tok.ExpiresAt,
		IsActive:       true,
	}); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	days := 0
	if tok.ExpiresAt != nil {
		days = int(time.Until(*tok.ExpiresAt).Hours() / 24)
	}
	return c.JSON(fiber.Map{"data": fiber.Map{
		"refreshed":      true,
		"expires_at":     tok.ExpiresAt,
		"days_remaining": days,
	}})
}
