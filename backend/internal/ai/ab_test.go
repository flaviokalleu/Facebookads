package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ABTest controls A/B testing between two providers for a task type.
type ABTest struct {
	ID        string    `json:"id"`
	TaskType  string    `json:"task_type"`
	ProviderA string    `json:"provider_a"`
	ProviderB string    `json:"provider_b"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	IsActive  bool      `json:"is_active"`
}

// ABTestResult stores a single A/B test result entry.
type ABTestResult struct {
	ID                  string    `json:"id"`
	TestID              string    `json:"test_id"`
	Provider            string    `json:"provider"`
	ResponseQualityScore float64  `json:"response_quality_score"`
	LatencyMs           int       `json:"latency_ms"`
	CostUSD             float64   `json:"cost_usd"`
	CreatedAt           time.Time `json:"created_at"`
}

// ABTestManager manages A/B tests for the LLM router.
type ABTestManager struct {
	db   *pgxpool.Pool
	mu   sync.RWMutex
	tests map[string]*activeABTest // key: task_type
}

type activeABTest struct {
	test   *ABTest
	mu     sync.Mutex
	nextA  bool
}

// NewABTestManager creates a new A/B test manager.
func NewABTestManager(db *pgxpool.Pool) *ABTestManager {
	return &ABTestManager{
		db:    db,
		tests: make(map[string]*activeABTest),
	}
}

// LoadActiveTests loads all active A/B tests from the DB.
func (m *ABTestManager) LoadActiveTests(ctx context.Context) error {
	rows, err := m.db.Query(ctx, `
		SELECT id, task_type, provider_a, provider_b, start_date, end_date, is_active
		FROM llm_ab_tests WHERE is_active = true AND end_date > now()
	`)
	if err != nil {
		return fmt.Errorf("ab_test: load: %w", err)
	}
	defer rows.Close()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.tests = make(map[string]*activeABTest)
	for rows.Next() {
		var t ABTest
		if err := rows.Scan(&t.ID, &t.TaskType, &t.ProviderA, &t.ProviderB, &t.StartDate, &t.EndDate, &t.IsActive); err != nil {
			return err
		}
		m.tests[t.TaskType] = &activeABTest{test: &t}
		slog.Info("ab_test: loaded active test", "task_type", t.TaskType, "a", t.ProviderA, "b", t.ProviderB)
	}
	return rows.Err()
}

// PickProvider selects which provider to use for a task when an A/B test is active.
// Traffic split: 50/50 between ProviderA and ProviderB.
func (m *ABTestManager) PickProvider(taskType TaskType) (string, bool) {
	m.mu.RLock()
	abt, ok := m.tests[string(taskType)]
	m.mu.RUnlock()
	if !ok {
		return "", false
	}

	abt.mu.Lock()
	useA := abt.nextA
	abt.nextA = !abt.nextA
	abt.mu.Unlock()

	if useA {
		return abt.test.ProviderA, true
	}
	return abt.test.ProviderB, true
}

// CreateTest creates a new A/B test in the DB.
func (m *ABTestManager) CreateTest(ctx context.Context, taskType, providerA, providerB string, durationDays int) (*ABTest, error) {
	now := time.Now()
	t := &ABTest{
		TaskType:  taskType,
		ProviderA: providerA,
		ProviderB: providerB,
		StartDate: now,
		EndDate:   now.AddDate(0, 0, durationDays),
		IsActive:  true,
	}
	err := m.db.QueryRow(ctx, `
		INSERT INTO llm_ab_tests (task_type, provider_a, provider_b, start_date, end_date, is_active)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id
	`, t.TaskType, t.ProviderA, t.ProviderB, t.StartDate, t.EndDate, t.IsActive).Scan(&t.ID)
	if err != nil {
		return nil, fmt.Errorf("ab_test: create: %w", err)
	}

	m.mu.Lock()
	m.tests[t.TaskType] = &activeABTest{test: t}
	m.mu.Unlock()

	slog.Info("ab_test: created", "task_type", taskType, "id", t.ID)
	return t, nil
}

// RecordResult stores a response quality comparison.
func (m *ABTestManager) RecordResult(ctx context.Context, testID, provider string, qualityScore float64, latencyMs int, costUSD float64) error {
	_, err := m.db.Exec(ctx, `
		INSERT INTO llm_ab_results (test_id, provider, response_quality_score, latency_ms, cost_usd)
		VALUES ($1,$2,$3,$4,$5)
	`, testID, provider, qualityScore, latencyMs, costUSD)
	if err != nil {
		return fmt.Errorf("ab_test: record result: %w", err)
	}
	return nil
}

// GetResults returns aggregated results for a test.
func (m *ABTestManager) GetResults(ctx context.Context, testID string) (map[string]any, error) {
	rows, err := m.db.Query(ctx, `
		SELECT provider,
		       COUNT(*) as requests,
		       AVG(response_quality_score) as avg_quality,
		       AVG(latency_ms) as avg_latency,
		       SUM(cost_usd) as total_cost
		FROM llm_ab_results
		WHERE test_id = $1
		GROUP BY provider
	`, testID)
	if err != nil {
		return nil, fmt.Errorf("ab_test: get results: %w", err)
	}
	defer rows.Close()

	aResults := map[string]any{"provider_a": nil, "provider_b": nil}
	for rows.Next() {
		var provider string
		var requests int
		var avgQuality, avgLatency, totalCost float64
		if err := rows.Scan(&provider, &requests, &avgQuality, &avgLatency, &totalCost); err != nil {
			return nil, err
		}
		key := ""
		m.mu.RLock()
		if abt, ok := m.tests[string(TaskType(""))]; ok {
			_ = abt
		}
		m.mu.RUnlock()
		_ = key
		aResults[provider] = map[string]any{
			"requests":    requests,
			"avg_quality": avgQuality,
			"avg_latency": avgLatency,
			"total_cost":  totalCost,
		}
	}

	return aResults, rows.Err()
}

func init() {
	rand.Seed(time.Now().UnixNano())
	// ensure json is used
	_ = json.Marshal
}
