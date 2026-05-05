package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/ai/agents"
	"github.com/facebookads/backend/internal/ai/providers"
	"github.com/facebookads/backend/internal/config"
	"github.com/facebookads/backend/internal/handler"
	"github.com/facebookads/backend/internal/metaads"
	"github.com/facebookads/backend/internal/middleware"
	"github.com/facebookads/backend/internal/orchestrator"
	"github.com/facebookads/backend/internal/repository/postgres"
	"github.com/facebookads/backend/internal/usecase"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Load .env — only DATABASE_URL should be here
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	// ─── Database ────────────────────────────────────────────────────────────
	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		slog.Error("database ping failed", "err", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// ─── Config Service ───────────────────────────────────────────────────────
	// Master key bootstrapped from env once — after that everything lives in DB
	masterKey := os.Getenv("CONFIG_MASTER_KEY")
	if masterKey == "" {
		masterKey = "default-dev-key-change-in-production!" // 32+ chars
	}
	cfg, err := config.NewService(db, masterKey)
	if err != nil {
		slog.Error("config service init failed", "err", err)
		os.Exit(1)
	}
	slog.Info("config service loaded")

	// ─── Redis ────────────────────────────────────────────────────────────────
	redisAddr := cfg.Get("redis.addr")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Warn("redis not available — caching and job locks disabled", "err", err)
	}

	// ─── Repositories ─────────────────────────────────────────────────────────
	campaignRepo    := postgres.NewCampaignRepo(db)
	insightRepo     := postgres.NewInsightRepo(db)
	userRepo        := postgres.NewUserRepo(db)
	userTokenRepo   := postgres.NewUserTokenRepo(db)
	anomalyRepo     := postgres.NewAnomalyRepo(db)
	recommendRepo   := postgres.NewRecommendationRepo(db)
	budgetRepo      := postgres.NewBudgetSuggestionRepo(db)
	llmUsageRepo    := postgres.NewLLMUsageRepo(db)
	adSetRepo       := postgres.NewAdSetRepo(db)
	adRepo          := postgres.NewAdRepo(db)

	// Phase F1 — Business Manager hierarchy + Imovel catalog
	credsRepo       := postgres.NewAppCredentialRepo(db, cfg)
	metaTokensRepo  := postgres.NewMetaTokenRepo(db, cfg)
	bmRepo          := postgres.NewBusinessManagerRepo(db)
	metaAccRepo     := postgres.NewMetaAdAccountRepo(db)
	metaPageRepo    := postgres.NewMetaPageRepo(db, cfg)
	metaPixelRepo   := postgres.NewMetaPixelRepo(db)
	imovelRepo      := postgres.NewImovelRepo(db)

	// Phase F2/F5/F6 — autonomous AI optimization agent
	aiActionRepo    := postgres.NewAIActionRepo(db)
	aiSafetyRepo    := postgres.NewAISafetyRuleRepo(db)

	// ─── Meta Ads client ──────────────────────────────────────────────────────
	metaClient := metaads.NewClient(cfg.Get("meta.api_version"))

	// ─── AI Router ────────────────────────────────────────────────────────────
	aiProviders := []ai.Provider{
		providers.NewAnthropic(cfg.GetSecret("ai.anthropic.api_key"), "claude-opus-4-7"),
		providers.NewAnthropic(cfg.GetSecret("ai.anthropic.api_key"), "claude-sonnet-4-6"),
		providers.NewOpenAI(cfg.GetSecret("ai.openai.api_key"), "gpt-5-4"),
		providers.NewOpenAI(cfg.GetSecret("ai.openai.api_key"), "gpt-4o-mini"),
		providers.NewDeepSeek(cfg.GetSecret("ai.deepseek.api_key"), "deepseek-v4-pro"),
		providers.NewDeepSeek(cfg.GetSecret("ai.deepseek.api_key"), "deepseek-r2"),
		providers.NewZhipu(cfg.GetSecret("ai.zhipu.api_key"), "glm-5"),
		providers.NewMoonshot(cfg.GetSecret("ai.moonshot.api_key"), "kimi-2-6"),
		providers.NewAlibaba(cfg.GetSecret("ai.alibaba.api_key"), "qwen-max"),
		providers.NewXAI(cfg.GetSecret("ai.xai.api_key"), "grok-3"),
	}
	var validProviders []ai.Provider
	for _, p := range aiProviders {
		if p.IsAvailable(context.Background()) {
			validProviders = append(validProviders, p)
		}
	}
	aiRouter := ai.NewRouter(validProviders)
	slog.Info("ai router ready", "providers", len(validProviders))

	// ─── Use Cases ────────────────────────────────────────────────────────────
	authUC      := usecase.NewAuthUseCase(userRepo, cfg)
	campaignUC  := usecase.NewCampaignUseCase(campaignRepo, insightRepo, userTokenRepo, adSetRepo, adRepo, metaClient)
	dashboardUC := usecase.NewDashboardUseCase(campaignRepo, insightRepo, anomalyRepo, budgetRepo, llmUsageRepo, recommendRepo)
	syncMetaUC  := usecase.NewSyncMetaAccount(metaClient, bmRepo, metaAccRepo, metaPageRepo, metaPixelRepo, metaTokensRepo, credsRepo, cfg)

	// F2 — sync from auto-discovered accounts
	syncCampaignsUC := usecase.NewSyncMetaCampaigns(metaClient, metaTokensRepo, metaAccRepo, campaignRepo, adSetRepo, adRepo, insightRepo)
	autoPilotV2     := usecase.NewAutoPilotV2(db, metaClient, metaTokensRepo, metaAccRepo, aiActionRepo, aiSafetyRepo)
	safetyRulesSvc  := usecase.NewSafetyRulesService(aiSafetyRepo)
	strategist      := agents.NewStrategist(db, cfg.GetSecret("ai.deepseek.api_key"), metaTokensRepo, metaAccRepo, aiActionRepo, llmUsageRepo)

	// ─── Orchestrator Scheduler ───────────────────────────────────────────────
	sched := orchestrator.New(db, rdb, aiRouter)
	sched.SetSyncCampaignsJob(func(ctx context.Context) error {
		users, err := listActiveTokenUsers(ctx, db)
		if err != nil {
			return err
		}
		for _, uid := range users {
			if err := syncCampaignsUC.Run(ctx, uid); err != nil {
				slog.Warn("sync_meta_campaigns: user failed", "user_id", uid, "err", err)
			}
		}
		return nil
	})
	sched.SetAutoPilotV2Job(func(ctx context.Context) error {
		return autoPilotV2.Run(ctx)
	})
	sched.SetStrategistJob(func(ctx context.Context) error {
		return strategist.Run(ctx)
	})
	go sched.Start(context.Background())

	// ─── Fiber app ────────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.Get("cors.allowed_origins"),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PUT,PATCH,DELETE",
	}))

	// ─── Routes ───────────────────────────────────────────────────────────────
	api := app.Group("/api/v1")

	// Auth (public)
	authH := handler.NewAuthHandler(authUC)
	api.Post("/auth/register", authH.Register)
	api.Post("/auth/login", authH.Login)

	// Protected routes
	protected := api.Use(middleware.JWT(cfg.GetSecret("jwt.secret")))

	// Meta connect
	metaAuthH := handler.NewMetaAuthHandler(userTokenRepo, metaClient, cfg)
	protected.Post("/auth/meta/connect", metaAuthH.Connect)
	protected.Get("/auth/meta/status", metaAuthH.Status)
		protected.Get("/auth/meta/accounts", metaAuthH.ListAdAccounts)
	protected.Post("/auth/meta/refresh-token", handler.NewTokenHandler(userTokenRepo, cfg).Refresh)
	protected.Post("/auth/meta/refresh-token", handler.NewTokenHandler(userTokenRepo, cfg).Refresh)

	// Phase F1 — auto-discovery connect + BM hierarchy tree
	metaConnectV2H := handler.NewMetaConnectV2Handler(syncMetaUC, credsRepo, metaTokensRepo, bmRepo, metaAccRepo, metaPageRepo, metaPixelRepo, metaClient, cfg)
	protected.Post("/auth/meta/connect-v2", metaConnectV2H.Connect)
	protected.Get("/businesses", metaConnectV2H.Tree)
	protected.Post("/businesses/sync", metaConnectV2H.Sync)

	// Campaigns
	campaignH := handler.NewCampaignHandler(campaignUC, aiRouter, metaClient, userTokenRepo)
	protected.Get("/campaigns", campaignH.List)
	protected.Get("/campaigns/creatives", campaignH.CreativeInsights)
	protected.Post("/campaigns/sync", campaignH.Sync)
	protected.Post("/campaigns/create-full", handler.NewCreateCampaignHandler(campaignUC, aiRouter, userTokenRepo, metaClient).CreateFull)
		protected.Post("/campaigns/:id/rules", handler.NewRulesHandler(cfg, campaignUC, metaClient, userTokenRepo).Save)
		protected.Get("/campaigns/:id/rules", handler.NewRulesHandler(cfg, campaignUC, metaClient, userTokenRepo).Get)
		protected.Post("/campaigns/:id/ab-test", handler.NewRulesHandler(cfg, campaignUC, metaClient, userTokenRepo).ABTest)
	protected.Post("/campaigns/test-133", handler.NewCreativeTestHandler(metaClient, userTokenRepo).Start133)
	protected.Get("/locations/search", handler.NewLocationHandler(userTokenRepo, metaClient).Search)
	protected.Get("/campaigns/:id", campaignH.Get)
	protected.Post("/campaigns/:id/auto-optimize", campaignH.AutoOptimize)
		protected.Post("/campaigns/:id/optimize", campaignH.Optimize)
		protected.Get("/campaigns/:id/recommendations", campaignH.GetRecommendations)
		protected.Post("/campaigns", campaignH.Create)
		protected.Patch("/campaigns/:id", campaignH.Update)
		protected.Delete("/campaigns/:id", campaignH.Delete)
		protected.Post("/recommendations/:id/apply", campaignH.ApplyRecommendation)
		protected.Post("/budget-suggestions/:id/apply", campaignH.ApplyBudgetSuggestion)

	// Dashboard
	dashH := handler.NewDashboardHandler(dashboardUC)
	protected.Get("/dashboard/summary", dashH.Summary)

	// Aggregated overview across ALL discovered ad accounts.
	dashOverviewH := handler.NewDashboardOverviewHandler(db)
	protected.Get("/dashboard/overview", dashOverviewH.Overview)

	// Free-form AI chat about the user's accounts.
	aiChatH := handler.NewAIChatHandler(db, cfg)
	protected.Get("/ai/chat", aiChatH.History)
	protected.Post("/ai/chat", aiChatH.Send)
	protected.Delete("/ai/chat", aiChatH.Clear)
	protected.Get("/ai/chat/suggestions", aiChatH.Suggest)

	// Cross-account campaigns list with windowed insights.
	campaignsListH := handler.NewCampaignsListHandler(db)
	protected.Get("/campanhas", campaignsListH.List)

	// Campaign detail (PT route — returns campaign + adsets + ads + daily + ai actions).
	campaignDetailH := handler.NewCampaignDetailHandler(db)
	protected.Get("/campanhas/:id", campaignDetailH.Get)

	// Token health (F1 flow — meta_tokens + app_credentials)
	tokenHealthH := handler.NewTokenHealthHandler(metaTokensRepo, credsRepo, credsRepo, metaClient, cfg)
	protected.Get("/auth/meta/token/health", tokenHealthH.Health)
	protected.Post("/auth/meta/token/refresh", tokenHealthH.Refresh)

	// Públicos / Custom Audiences cross-account
	publicosH := handler.NewPublicosHandler(metaTokensRepo, metaAccRepo, metaClient)
	protected.Get("/publicos", publicosH.List)

	// Imoveis (catálogo multi-segmento)
	imoveisH := handler.NewImoveisHandler(imovelRepo)
	protected.Get("/imoveis", imoveisH.List)
	protected.Post("/imoveis", imoveisH.Create)
	protected.Get("/imoveis/:id", imoveisH.Get)
	protected.Patch("/imoveis/:id", imoveisH.Update)
	protected.Delete("/imoveis/:id", imoveisH.Delete)
	protected.Get("/dashboard/campaigns", dashH.Campaigns)
	protected.Get("/dashboard/campaigns/:id/insights", dashH.CampaignInsights)
	protected.Get("/dashboard/top-creatives", dashH.TopCreatives)
	protected.Get("/dashboard/anomalies", dashH.Anomalies)
	protected.Get("/dashboard/budget-advisor", dashH.BudgetAdvisor)
		// Creatives
		creativeH := handler.NewCreativeHandler(campaignUC, adRepo, adSetRepo, insightRepo, aiRouter)
		protected.Get("/creatives", creativeH.List)
		protected.Post("/creatives/analyze", creativeH.Analyze)
		protected.Post("/creatives/improve", creativeH.Improve)
	protected.Get("/dashboard/recommendations", dashH.Recommendations)

	// Auth — me endpoint (protected)
	protected.Get("/auth/me", authH.Me)

	// Admin
	adminH := handler.NewAdminHandler(cfg, llmUsageRepo, aiRouter, sched)
	admin := protected.Use(middleware.AdminOnly())
	admin.Get("/admin/config", adminH.ListConfig)
	admin.Put("/admin/config/:key", adminH.SetConfig)
	admin.Get("/admin/providers", adminH.Providers)
	admin.Post("/admin/providers/:name/test", adminH.TestProvider)
	admin.Get("/admin/providers/routing", adminH.RoutingTable)
	admin.Put("/admin/providers/routing", adminH.OverrideRouting)
	admin.Get("/admin/ai-usage", adminH.AIUsage)
	admin.Get("/admin/ai-usage/daily", adminH.AIUsageDaily)
	admin.Get("/admin/scheduler/status", adminH.SchedulerStatus)

	// Account detail (per-account KPIs + campaign list)
	accDetailH := handler.NewAccountDetailHandler(db, cfg, metaTokensRepo, metaClient)
	protected.Get("/contas/:account_id", accDetailH.Get)
	protected.Get("/contas/:account_id/campanhas", accDetailH.ListCampaigns)
	protected.Get("/contas/:account_id/insights/daily", accDetailH.DailyInsights)
	protected.Get("/contas/:account_id/breakdowns", accDetailH.Breakdowns)
	protected.Get("/contas/:account_id/analysis", accDetailH.GetAnalysis)
	protected.Post("/contas/:account_id/analyze", accDetailH.Analyze)

	// AI actions (autonomous optimization agent)
	aiActionsH := handler.NewAIActionsHandler(aiActionRepo, autoPilotV2, safetyRulesSvc)
	protected.Get("/ai/actions", aiActionsH.List)
	protected.Post("/ai/actions/:id/approve", aiActionsH.Approve)
	protected.Post("/ai/actions/:id/reject", aiActionsH.Reject)
	protected.Post("/ai/actions/:id/revert", aiActionsH.Revert)
	protected.Get("/ai/safety-rules", aiActionsH.ListSafetyRules)
	protected.Put("/ai/safety-rules/:rule_key", aiActionsH.UpsertSafetyRule)

	// Webhooks (public — signature verified internally)
	webhookH := handler.NewMetaWebhookHandler(db, cfg.GetSecret("meta.app_secret"))
	api.Get("/webhooks/meta", handler.MetaWebhookVerify)
	api.Post("/webhooks/meta", webhookH.Handle)

	// ─── Start ────────────────────────────────────────────────────────────────
	port := cfg.Get("server.port")
	if port == "" {
		port = "8080"
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "port", port)
		if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
			slog.Error("server error", "err", err)
		}
	}()

	<-quit
	slog.Info("graceful shutdown...")
	if err := app.ShutdownWithContext(context.Background()); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

func listActiveTokenUsers(ctx context.Context, db *pgxpool.Pool) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT DISTINCT user_id::text FROM meta_tokens WHERE is_active = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal_error"

	var fe *fiber.Error
	if errors.As(err, &fe) {
		code = fe.Code
		msg = fe.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    msg,
			"message": msg,
		},
	})
}
