package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/facebookads/backend/internal/ai"
	"github.com/facebookads/backend/internal/ai/agents"
)

// JobRun records the last execution of a background job.
type JobRun struct {
	Name       string     `json:"name"`
	LastRun    *time.Time `json:"last_run"`
	LastStatus string     `json:"last_status"` // success, failed, running
	ErrorCount int        `json:"error_count"`
	ModelUsed  string     `json:"model_used,omitempty"`
	DurationMs int64      `json:"duration_ms"`
}

// Scheduler manages background goroutines with Redis distributed locking.
type Scheduler struct {
	db      *pgxpool.Pool
	rdb     *redis.Client
	router  *ai.Router

	mu   sync.RWMutex
	jobs []JobRun
}

func New(db *pgxpool.Pool, rdb *redis.Client, router *ai.Router) *Scheduler {
	return &Scheduler{
		db:     db,
		rdb:    rdb,
		router: router,
		jobs: []JobRun{
			{Name: "sync_insights"},
			{Name: "classify_campaigns"},
			{Name: "detect_anomalies"},
			{Name: "generate_recommendations"},
			{Name: "budget_advisor"},
			{Name: "creative_analysis"},
			{Name: "provider_health_check"},
			{Name: "auto_pilot"},
		},
	}
}

// Start launches all background jobs with their intervals.
func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("scheduler: starting background jobs")

	go s.runLoop(ctx, "sync_insights", 6*time.Hour, s.syncInsights)
	go s.runLoop(ctx, "classify_campaigns", 12*time.Hour, s.classifyCampaigns)
	go s.runLoop(ctx, "detect_anomalies", 2*time.Hour, s.detectAnomalies)
	go s.runLoop(ctx, "generate_recommendations", 24*time.Hour, s.generateRecommendations)
	go s.runLoop(ctx, "budget_advisor", 24*time.Hour, s.budgetAdvisor)
	go s.runLoop(ctx, "creative_analysis", 24*time.Hour, s.creativeAnalysis)
	go s.runLoop(ctx, "provider_health_check", 15*time.Minute, s.providerHealthCheck)
		go s.runLoop(ctx, "auto_pilot", 6*time.Hour, s.autoPilot)

	slog.Info("scheduler: all jobs started")
}

// Status returns the current state of all jobs (for admin API).
func (s *Scheduler) Status() []JobRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]JobRun, len(s.jobs))
	copy(result, s.jobs)
	return result
}

// runLoop executes a job at the given interval. It skips if a lock is already held.
func (s *Scheduler) runLoop(ctx context.Context, name string, interval time.Duration, fn func(context.Context) error) {
	// Run immediately on start
	s.runJob(ctx, name, fn)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler: job stopping", "name", name)
			return
		case <-ticker.C:
			s.runJob(ctx, name, fn)
		}
	}
}

func (s *Scheduler) runJob(ctx context.Context, name string, fn func(context.Context) error) {
	lockKey := fmt.Sprintf("scheduler:lock:%s", name)
	if s.rdb != nil {
		ok, err := s.rdb.SetNX(ctx, lockKey, "1", 30*time.Minute).Result()
		if err != nil || !ok {
			slog.Debug("scheduler: skipped (locked)", "job", name)
			return
		}
		defer s.rdb.Del(ctx, lockKey)
	}

	start := time.Now()
	s.setJobStatus(name, "running", 0, "", 0)

	err := fn(ctx)

	duration := time.Since(start)
	status := "success"
	if err != nil {
		status = "failed"
		slog.Error("scheduler: job failed", "job", name, "err", err, "duration_ms", duration.Milliseconds())
	} else {
		slog.Info("scheduler: job completed", "job", name, "duration_ms", duration.Milliseconds())
	}

	s.updateJob(name, status, duration)
}

func (s *Scheduler) setJobStatus(name, status string, errCount int, model string, durationMs int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, j := range s.jobs {
		if j.Name == name {
			now := time.Now()
			s.jobs[i].LastRun = &now
			s.jobs[i].LastStatus = status
			s.jobs[i].ErrorCount = errCount
			s.jobs[i].ModelUsed = model
			s.jobs[i].DurationMs = durationMs
			return
		}
	}
}

func (s *Scheduler) updateJob(name, status string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, j := range s.jobs {
		if j.Name == name {
			now := time.Now()
			s.jobs[i].LastRun = &now
			s.jobs[i].LastStatus = status
			s.jobs[i].DurationMs = duration.Milliseconds()
			if status == "failed" {
				s.jobs[i].ErrorCount++
				// Alert webhook if 3 consecutive failures
				if s.jobs[i].ErrorCount >= 3 {
					slog.Error("scheduler: job failed 3x consecutively", "job", name, "error_count", s.jobs[i].ErrorCount)
				}
			} else {
				s.jobs[i].ErrorCount = 0
			}
			return
		}
	}
}

// ─── Job implementations ────────────────────────────────────────────────────

func (s *Scheduler) syncInsights(ctx context.Context) error {
	// Fetch all users and trigger insights sync
	users, err := s.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	if len(users) == 0 {
		slog.Info("scheduler: no users for sync_insights")
		return nil
	}

	for _, userID := range users {
		if err := s.syncUserInsights(ctx, userID); err != nil {
			slog.Error("scheduler: sync insights failed for user", "user_id", userID, "err", err)
			continue
		}
	}
	return nil
}

func (s *Scheduler) syncUserInsights(ctx context.Context, userID string) error {
	campaigns, err := s.listCampaigns(ctx, userID)
	if err != nil {
		return err
	}
	for _, c := range campaigns {
		_ = c
		// TODO: Call metaClient.FetchInsights and upsert via InsightRepo
	}
	slog.Info("scheduler: insights synced", "user_id", userID, "campaigns", len(campaigns))
	return nil
}

func (s *Scheduler) classifyCampaigns(ctx context.Context) error {
	classifier := agents.NewClassifier(s.db, s.router)
	users, err := s.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	for _, userID := range users {
		if err := classifier.RunClassifyAll(ctx, userID); err != nil {
			slog.Error("scheduler: classify failed", "user_id", userID, "err", err)
		}
	}
	return nil
}

func (s *Scheduler) detectAnomalies(ctx context.Context) error {
	detector := agents.NewAnomalyDetector(s.db, s.router)
	users, err := s.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	for _, userID := range users {
		if err := detector.RunDetectAll(ctx, userID); err != nil {
			slog.Error("scheduler: anomaly detection failed", "user_id", userID, "err", err)
		}
	}
	return nil
}

func (s *Scheduler) generateRecommendations(ctx context.Context) error {
	optimizer := agents.NewOptimizer(s.db, s.router)
	users, err := s.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	for _, userID := range users {
		if err := optimizer.RunGenerateAll(ctx, userID); err != nil {
			slog.Error("scheduler: recommendations failed", "user_id", userID, "err", err)
		}
	}
	return nil
}

func (s *Scheduler) budgetAdvisor(ctx context.Context) error {
	advisor := agents.NewBudgetAdvisor(s.db, s.router)
	users, err := s.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	for _, userID := range users {
		if err := advisor.RunAnalyzeAll(ctx, userID); err != nil {
			slog.Error("scheduler: budget advisor failed", "user_id", userID, "err", err)
		}
	}
	return nil
}

func (s *Scheduler) creativeAnalysis(ctx context.Context) error {
	analyst := agents.NewCreativeAnalyst(s.db, s.router)
	users, err := s.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	for _, userID := range users {
		if err := analyst.RunAnalyzeAll(ctx, userID); err != nil {
			slog.Error("scheduler: creative analysis failed", "user_id", userID, "err", err)
		}
	}
	return nil
}

func (s *Scheduler) providerHealthCheck(ctx context.Context) error {
	if s.router == nil {
		return nil
	}
	providers := s.router.ActiveProviders()
	for _, name := range providers {
		// available := ai.CheckAvailable(ctx, nil)
		// _ = available
		slog.Debug("scheduler: provider health", "provider", name)
	}
	return nil
}

// ─── Auto-Pilot ────────────────────────────────────────────────────────────

func (s *Scheduler) autoPilot(ctx context.Context) error {
	users, err := s.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}
	for _, userID := range users {
		if err := s.runAutoPilotForUser(ctx, userID); err != nil {
			slog.Error("auto_pilot: failed for user", "user_id", userID, "err", err)
		}
	}
	return nil
}

func (s *Scheduler) runAutoPilotForUser(ctx context.Context, userID string) error {
	rows, err := s.db.Query(ctx, `
		SELECT c.id, c.name, COALESCE(AVG(ci.frequency),0) as avg_freq,
		       COALESCE(AVG(ci.ctr),0) as avg_ctr,
		       COALESCE(SUM(ci.spend),0) as total_spend
		FROM campaigns c
		JOIN campaign_insights ci ON ci.campaign_id = c.id
		WHERE c.user_id = $1 AND c.deleted_at IS NULL
		AND ci.date >= CURRENT_DATE - 7
		GROUP BY c.id, c.name
	`, userID)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name string
		var freq, ctr, spend float64
		if err := rows.Scan(&id, &name, &freq, &ctr, &spend); err != nil {
			continue
		}
		if freq > 5 {
			slog.Warn("auto_pilot: audience saturation", "campaign", name, "frequency", freq)
			s.db.Exec(ctx, `INSERT INTO anomalies (campaign_id, type, severity, description, is_active, detected_at)
				VALUES ($1,$2,$3,$4,true,now())`,
				id, "AUDIENCE_SATURATION", "HIGH",
				fmt.Sprintf("Frequência de %.1f indica saturação do público. Atualize os criativos ou expanda a segmentação.", freq))
		}
		if ctr < 0.005 && spend > 0 {
			slog.Warn("auto_pilot: low CTR", "campaign", name, "ctr", ctr)
			s.db.Exec(ctx, `INSERT INTO anomalies (campaign_id, type, severity, description, is_active, detected_at)
				VALUES ($1,$2,$3,$4,true,now())`,
				id, "CTR_DROP", "MEDIUM",
				fmt.Sprintf("CTR de %.2f%% está abaixo do ideal. Considere revisar criativos e segmentação.", ctr*100))
		}
	}
	return rows.Err()
}

// ─── DB helpers ──────────────────────────────────────────────────────────────

func (s *Scheduler) listUsers(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT id FROM users WHERE deleted_at IS NULL`)
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

func (s *Scheduler) listCampaigns(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT id FROM campaigns WHERE user_id = $1 AND deleted_at IS NULL`, userID)
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
