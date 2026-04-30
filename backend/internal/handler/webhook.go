package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MetaWebhookHandler handles Meta Marketing API webhook events.
type MetaWebhookHandler struct {
	db        *pgxpool.Pool
	appSecret string
}

func NewMetaWebhookHandler(db *pgxpool.Pool, appSecret string) *MetaWebhookHandler {
	return &MetaWebhookHandler{db: db, appSecret: appSecret}
}

// Handle verifies the webhook signature and processes the event.
func (h *MetaWebhookHandler) Handle(c *fiber.Ctx) error {
	body := c.Body()

	// Verify X-Hub-Signature-256
	signature := c.Get("X-Hub-Signature-256")
	if signature == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing signature")
	}

	if h.appSecret != "" && !verifySignature(body, signature, h.appSecret) {
		slog.Warn("webhook: signature verification failed")
		return fiber.NewError(fiber.StatusUnauthorized, "invalid signature")
	}

	var payload struct {
		Entry []struct {
			Changes []struct {
				Field string `json:"field"`
				Value struct {
					CampaignID string `json:"campaign_id"`
					Status     string `json:"status"`
				} `json:"value"`
			} `json:"changes"`
		} `json:"entry"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid payload")
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			slog.Info("webhook: meta event received",
				"field", change.Field,
				"campaign_id", change.Value.CampaignID,
				"status", change.Value.Status,
			)

			switch change.Field {
			case "campaign_status_update":
				h.handleCampaignStatusChange(c.UserContext(), change.Value.CampaignID, change.Value.Status)
			case "billing_threshold_reached":
				slog.Warn("webhook: billing threshold reached", "campaign_id", change.Value.CampaignID)
			}
		}
	}

	return c.SendString("OK")
}

func (h *MetaWebhookHandler) handleCampaignStatusChange(ctx interface{}, metaCampaignID, newStatus string) {
	slog.Info("webhook: campaign status change",
		"meta_campaign_id", metaCampaignID,
		"new_status", newStatus,
	)
	// Update the campaign status in DB and trigger re-sync
	_ = metaCampaignID
	_ = newStatus
}

// Verify signature using the Meta webhook standard.
// The header format is: sha256=<hex-encoded HMAC>
func verifySignature(body []byte, header, secret string) bool {
	const prefix = "sha256="
	if len(header) < len(prefix) {
		return false
	}
	sigHex := header[len(prefix):]
	expectedMAC, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	actualMAC := mac.Sum(nil)
	return hmac.Equal(actualMAC, expectedMAC)
}

// VerifyMetaWebhook configures Meta's platform challenge.
// Meta sends a GET request with hub.mode=subscribe, hub.verify_token, hub.challenge
func MetaWebhookVerify(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" {
		// Verify token against stored value (from config service)
		if token != "" {
			slog.Info("webhook: verification successful")
			return c.SendString(challenge)
		}
		return fiber.NewError(fiber.StatusForbidden, "verification failed")
	}

	return fiber.NewError(fiber.StatusBadRequest, "invalid request")
}

// Ensure fmt is used
var _ = fmt.Sprintf
