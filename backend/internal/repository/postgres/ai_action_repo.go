package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/facebookads/backend/internal/domain"
)

type AIActionRepo struct {
	db *pgxpool.Pool
}

func NewAIActionRepo(db *pgxpool.Pool) *AIActionRepo {
	return &AIActionRepo{db: db}
}

func (r *AIActionRepo) Create(ctx context.Context, a *domain.AIAction) error {
	if a.MetricSnapshot == nil {
		a.MetricSnapshot = json.RawMessage("{}")
	}
	if a.ProposedChange == nil {
		a.ProposedChange = json.RawMessage("{}")
	}
	if a.Status == "" {
		a.Status = domain.AIActionStatusPending
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO ai_actions_log
		  (user_id, account_meta_id, action_type, target_meta_id, target_kind,
		   reason, metric_snapshot, proposed_change, source, mode, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, created_at
	`, a.UserID, a.AccountMetaID, a.ActionType, a.TargetMetaID, a.TargetKind,
		a.Reason, a.MetricSnapshot, a.ProposedChange, a.Source, a.Mode, a.Status,
	).Scan(&a.ID, &a.CreatedAt)
}

func (r *AIActionRepo) GetByID(ctx context.Context, id string) (*domain.AIAction, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, account_meta_id, action_type, target_meta_id, target_kind,
		       reason, metric_snapshot, proposed_change, source, mode, status,
		       meta_response, created_at, decided_at, executed_at
		FROM ai_actions_log WHERE id = $1
	`, id)
	return scanAIAction(row)
}

func (r *AIActionRepo) ListPendingByUser(ctx context.Context, userID string, limit int) ([]*domain.AIAction, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, account_meta_id, action_type, target_meta_id, target_kind,
		       reason, metric_snapshot, proposed_change, source, mode, status,
		       meta_response, created_at, decided_at, executed_at
		FROM ai_actions_log
		WHERE user_id = $1 AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.AIAction
	for rows.Next() {
		a, err := scanAIAction(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *AIActionRepo) ListByUser(ctx context.Context, userID, status string, limit int) ([]*domain.AIAction, error) {
	if limit <= 0 {
		limit = 100
	}
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = r.db.Query(ctx, `
			SELECT id, user_id, account_meta_id, action_type, target_meta_id, target_kind,
			       reason, metric_snapshot, proposed_change, source, mode, status,
			       meta_response, created_at, decided_at, executed_at
			FROM ai_actions_log
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		`, userID, limit)
	} else {
		rows, err = r.db.Query(ctx, `
			SELECT id, user_id, account_meta_id, action_type, target_meta_id, target_kind,
			       reason, metric_snapshot, proposed_change, source, mode, status,
			       meta_response, created_at, decided_at, executed_at
			FROM ai_actions_log
			WHERE user_id = $1 AND status = $2
			ORDER BY created_at DESC
			LIMIT $3
		`, userID, status, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.AIAction
	for rows.Next() {
		a, err := scanAIAction(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (r *AIActionRepo) MarkApproved(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE ai_actions_log SET status='approved', decided_at = now()
		WHERE id = $1 AND status = 'pending'
	`, id)
	return err
}

func (r *AIActionRepo) MarkRejected(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE ai_actions_log SET status='rejected', decided_at = now()
		WHERE id = $1 AND status = 'pending'
	`, id)
	return err
}

func (r *AIActionRepo) MarkExecuted(ctx context.Context, id string, metaResponse []byte) error {
	if metaResponse == nil {
		metaResponse = []byte("{}")
	}
	_, err := r.db.Exec(ctx, `
		UPDATE ai_actions_log SET status='executed', executed_at = now(), meta_response = $2
		WHERE id = $1
	`, id, metaResponse)
	return err
}

func (r *AIActionRepo) MarkFailed(ctx context.Context, id string, metaResponse []byte) error {
	if metaResponse == nil {
		metaResponse = []byte("{}")
	}
	_, err := r.db.Exec(ctx, `
		UPDATE ai_actions_log SET status='failed', executed_at = now(), meta_response = $2
		WHERE id = $1
	`, id, metaResponse)
	return err
}

func (r *AIActionRepo) MarkReverted(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE ai_actions_log SET status='reverted', decided_at = now()
		WHERE id = $1
	`, id)
	return err
}

// CountPausesLast24h counts how many ads were paused in the last 24h for the given account.
// Used by the per-account pause cap.
func (r *AIActionRepo) CountPausesLast24h(ctx context.Context, userID, accountMetaID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM ai_actions_log
		WHERE user_id = $1
		  AND account_meta_id = $2
		  AND action_type = 'pause_ad'
		  AND status = 'executed'
		  AND executed_at >= now() - interval '24 hours'
	`, userID, accountMetaID).Scan(&n)
	return n, err
}

func scanAIAction(s scanner) (*domain.AIAction, error) {
	var a domain.AIAction
	err := s.Scan(
		&a.ID, &a.UserID, &a.AccountMetaID, &a.ActionType, &a.TargetMetaID, &a.TargetKind,
		&a.Reason, &a.MetricSnapshot, &a.ProposedChange, &a.Source, &a.Mode, &a.Status,
		&a.MetaResponse, &a.CreatedAt, &a.DecidedAt, &a.ExecutedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%w: ai_action", domain.ErrNotFound)
		}
		return nil, err
	}
	return &a, nil
}

// ─── Safety rules repo ────────────────────────────────────────────────────────

type AISafetyRuleRepo struct {
	db *pgxpool.Pool
}

func NewAISafetyRuleRepo(db *pgxpool.Pool) *AISafetyRuleRepo {
	return &AISafetyRuleRepo{db: db}
}

// Upsert inserts or updates a rule. accountMetaID==nil means a global override.
func (r *AISafetyRuleRepo) Upsert(ctx context.Context, rule *domain.AISafetyRule) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO ai_safety_rules (user_id, account_meta_id, rule_key, rule_value, enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, now())
		ON CONFLICT (user_id, account_meta_id, rule_key) DO UPDATE SET
		  rule_value = EXCLUDED.rule_value,
		  enabled    = EXCLUDED.enabled,
		  updated_at = now()
		RETURNING id, created_at, updated_at
	`, rule.UserID, rule.AccountMetaID, rule.RuleKey, rule.RuleValue, rule.Enabled,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

// ListByUser returns every override row for the user.
func (r *AISafetyRuleRepo) ListByUser(ctx context.Context, userID string) ([]*domain.AISafetyRule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, account_meta_id, rule_key, rule_value, enabled, created_at, updated_at
		FROM ai_safety_rules WHERE user_id = $1 AND enabled = true
		ORDER BY rule_key
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.AISafetyRule
	for rows.Next() {
		var rule domain.AISafetyRule
		if err := rows.Scan(&rule.ID, &rule.UserID, &rule.AccountMetaID,
			&rule.RuleKey, &rule.RuleValue, &rule.Enabled,
			&rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &rule)
	}
	return result, rows.Err()
}
