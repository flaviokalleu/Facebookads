package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/ai/prompts"
	"github.com/facebookads/backend/internal/ai/providers"
	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/middleware"
)

type AIChatHandler struct {
	db  *pgxpool.Pool
	cfg *config.Service
}

func NewAIChatHandler(db *pgxpool.Pool, cfg *config.Service) *AIChatHandler {
	return &AIChatHandler{db: db, cfg: cfg}
}

type chatMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// History handles GET /api/v1/ai/chat — returns last 50 messages.
func (h *AIChatHandler) History(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	ctx := c.UserContext()

	rows, err := h.db.Query(ctx, `
		SELECT id, role, content, created_at
		FROM ai_chat_messages
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	out := make([]chatMessage, 0)
	for rows.Next() {
		var m chatMessage
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			continue
		}
		out = append(out, m)
	}
	// reverse so frontend gets chronological order
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return c.JSON(fiber.Map{"data": out})
}

// Send handles POST /api/v1/ai/chat with body { "message": "..." }.
func (h *AIChatHandler) Send(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	ctx := c.UserContext()

	var body struct {
		Message string `json:"message"`
	}
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.Message) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "mensagem vazia")
	}
	body.Message = strings.TrimSpace(body.Message)

	apiKey := h.cfg.GetSecret("ai.deepseek.api_key")
	if apiKey == "" {
		return fiber.NewError(fiber.StatusBadRequest, "configure a chave da DeepSeek em Ajustes → Chaves de IA")
	}

	// Persist user message immediately.
	var userMsgID string
	if err := h.db.QueryRow(ctx, `
		INSERT INTO ai_chat_messages (user_id, role, content)
		VALUES ($1, 'user', $2)
		RETURNING id
	`, userID, body.Message).Scan(&userMsgID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "persist user msg: "+err.Error())
	}

	// Build account snapshot for the system prompt context.
	snapshot, err := h.buildUserSnapshot(ctx, userID)
	if err != nil {
		slog.Warn("ai chat: snapshot failed", "user_id", userID, "err", err)
		snapshot = "(sem dados sincronizados ainda)"
	}

	// Pull last 10 message pairs for short-term memory.
	historyRows, err := h.db.Query(ctx, `
		SELECT role, content FROM (
		  SELECT role, content, created_at
		  FROM ai_chat_messages
		  WHERE user_id = $1 AND id <> $2
		  ORDER BY created_at DESC
		  LIMIT 20
		) t ORDER BY created_at ASC
	`, userID, userMsgID)
	type histMsg struct{ Role, Content string }
	var hist []histMsg
	if err == nil {
		defer historyRows.Close()
		for historyRows.Next() {
			var m histMsg
			if err := historyRows.Scan(&m.Role, &m.Content); err == nil {
				hist = append(hist, m)
			}
		}
	}

	// Compose user prompt: snapshot + history + new message.
	var sb strings.Builder
	sb.WriteString("=== SNAPSHOT DAS CONTAS DO USUÁRIO (atualizado agora) ===\n")
	sb.WriteString(snapshot)
	if len(hist) > 0 {
		sb.WriteString("\n\n=== CONVERSA ANTERIOR (mais antiga primeiro) ===\n")
		for _, m := range hist {
			label := "Usuário"
			if m.Role == "assistant" {
				label = "Você (IA)"
			}
			fmt.Fprintf(&sb, "%s: %s\n", label, m.Content)
		}
	}
	sb.WriteString("\n\n=== NOVA PERGUNTA DO USUÁRIO ===\n")
	sb.WriteString(body.Message)

	provider := providers.NewDeepSeek(apiKey, "deepseek-chat")
	start := time.Now()
	resp, err := provider.Complete(ctx, ai.CompletionRequest{
		SystemPrompt: prompts.ChatSystemPrompt,
		UserPrompt:   sb.String(),
		MaxTokens:    900,
		Temperature:  0.5,
	})
	latency := time.Since(start).Milliseconds()
	if err != nil {
		slog.Error("ai chat: deepseek failed", "user_id", userID, "err", err)
		return fiber.NewError(fiber.StatusBadGateway, "deepseek: "+err.Error())
	}

	// Persist assistant message.
	var assistID string
	var assistCreated time.Time
	if err := h.db.QueryRow(ctx, `
		INSERT INTO ai_chat_messages
		  (user_id, role, content, model_used, input_tokens, output_tokens, cost_usd, latency_ms)
		VALUES ($1, 'assistant', $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`, userID, resp.Content, resp.ModelUsed, resp.InputTokens, resp.OutputTokens, resp.CostUSD, latency).Scan(&assistID, &assistCreated); err != nil {
		slog.Warn("ai chat: persist assistant failed", "err", err)
	}

	return c.JSON(fiber.Map{"data": fiber.Map{
		"reply": fiber.Map{
			"id":         assistID,
			"role":       "assistant",
			"content":    resp.Content,
			"created_at": assistCreated,
		},
		"latency_ms": latency,
		"cost_usd":   resp.CostUSD,
	}})
}

// Clear handles DELETE /api/v1/ai/chat — wipes the user's chat history.
func (h *AIChatHandler) Clear(c *fiber.Ctx) error {
	userID := middleware.UserID(c)
	if _, err := h.db.Exec(c.UserContext(),
		`DELETE FROM ai_chat_messages WHERE user_id = $1`, userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"cleared": true}})
}

// buildUserSnapshot creates a structured PT-BR snapshot of the user's accounts
// to inject into the chat system prompt.
func (h *AIChatHandler) buildUserSnapshot(ctx context.Context, userID string) (string, error) {
	var sb strings.Builder

	// Aggregated KPIs (last 7d).
	var spend, leads float64
	var leadsCount int64
	var imps, clk int64
	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(ci.spend),0), COALESCE(SUM(ci.leads),0)::bigint,
		       COALESCE(SUM(ci.impressions),0)::bigint, COALESCE(SUM(ci.clicks),0)::bigint
		FROM campaign_insights ci
		JOIN campaigns c ON c.id = ci.campaign_id
		WHERE c.user_id = $1 AND c.deleted_at IS NULL
		  AND ci.date > CURRENT_DATE - 7
	`, userID).Scan(&spend, &leadsCount, &imps, &clk); err != nil {
		return "", err
	}
	leads = float64(leadsCount)
	avgCPL := 0.0
	if leads > 0 {
		avgCPL = spend / leads
	}

	fmt.Fprintf(&sb, "Janela: últimos 7 dias\n")
	fmt.Fprintf(&sb, "Total: gasto R$ %.2f, %d contatos, custo médio R$ %.2f, %d cliques, %d impressões\n",
		spend, leadsCount, avgCPL, clk, imps)

	// Per-account roll-up (top 10 by spend).
	rows, err := h.db.Query(ctx, `
		SELECT a.meta_id, a.name, a.balance, COALESCE(b.name,'') AS bm,
		       COALESCE(SUM(ci.spend),0)::float8,
		       COALESCE(SUM(ci.leads),0)::bigint,
		       COALESCE(SUM(ci.impressions),0)::bigint,
		       COALESCE(SUM(ci.clicks),0)::bigint
		FROM meta_ad_accounts a
		LEFT JOIN business_managers b ON b.id = a.bm_id
		LEFT JOIN campaigns c ON c.user_id = a.user_id AND c.ad_account_id = a.meta_id AND c.deleted_at IS NULL
		LEFT JOIN campaign_insights ci ON ci.campaign_id = c.id AND ci.date > CURRENT_DATE - 7
		WHERE a.user_id = $1
		GROUP BY a.meta_id, a.name, a.balance, b.name
		ORDER BY COALESCE(SUM(ci.spend),0) DESC
		LIMIT 15
	`, userID)
	if err != nil {
		return sb.String(), nil
	}
	defer rows.Close()

	sb.WriteString("\nContas (top 15 por gasto, 7d):\n")
	for rows.Next() {
		var metaID, name, bm string
		var balanceCents, accSpend float64
		var accLeads, accImps, accClk int64
		if err := rows.Scan(&metaID, &name, &balanceCents, &bm, &accSpend, &accLeads, &accImps, &accClk); err != nil {
			continue
		}
		balance := balanceCents / 100.0
		cpl := 0.0
		if accLeads > 0 {
			cpl = accSpend / float64(accLeads)
		}
		ctr := 0.0
		if accImps > 0 {
			ctr = float64(accClk) / float64(accImps) * 100
		}
		bmDisplay := bm
		if bmDisplay == "" {
			bmDisplay = "(pessoal)"
		}
		fmt.Fprintf(&sb, "- %s [%s] (%s) — saldo R$ %.2f, gasto R$ %.2f, %d contatos, custo R$ %.2f, CTR %.2f%%\n",
			name, metaID, bmDisplay, balance, accSpend, accLeads, cpl, ctr)
	}

	// Recent AI actions (executed in last 24h).
	actRows, err := h.db.Query(ctx, `
		SELECT action_type, target_meta_id, status, reason, created_at
		FROM ai_actions_log
		WHERE user_id = $1 AND created_at > now() - INTERVAL '24 hours'
		ORDER BY created_at DESC
		LIMIT 10
	`, userID)
	if err == nil {
		defer actRows.Close()
		var lines []string
		for actRows.Next() {
			var typ, tgt, status, reason string
			var ts time.Time
			if err := actRows.Scan(&typ, &tgt, &status, &reason, &ts); err == nil {
				lines = append(lines, fmt.Sprintf("- [%s] %s sobre %s: %s", status, typ, tgt, reason))
			}
		}
		if len(lines) > 0 {
			sb.WriteString("\nAções da IA nas últimas 24h:\n")
			sb.WriteString(strings.Join(lines, "\n"))
		}
	}

	return sb.String(), nil
}

// Suggest is a tiny endpoint that returns canned starter questions. Used
// by the frontend's chip suggestions when chat is empty.
func (h *AIChatHandler) Suggest(c *fiber.Ctx) error {
	suggestions := []string{
		"Qual conta está com pior desempenho hoje?",
		"Compare AVAK MCMV com Programa Casa Verde Amarelo.",
		"Quais contas têm saldo acabando?",
		"O que devo fazer agora pra melhorar o resultado?",
		"Resuma a semana das 5 maiores contas.",
		"Tem alguma campanha pronta pra escalar?",
	}
	return c.JSON(fiber.Map{"data": suggestions})
}

// Avoid unused import when JSON helpers aren't used directly.
var _ = json.Marshal
