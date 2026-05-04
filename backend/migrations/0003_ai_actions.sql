-- Migration 0003: Autonomous AI optimization agent (HYBRID — auto + propose)
-- Phase F2/F5/F6 of PLANO_GESTOR_IA.md.

-- ─────────────────────────────────────────────
-- AI ACTIONS LOG
-- Every deterministic safety rule trigger and every DeepSeek-proposed action
-- lands here. `mode='auto'` is executed immediately, `mode='propose'` waits
-- for human approval via /ai/actions/:id/approve.
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS ai_actions_log (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  account_meta_id TEXT NOT NULL,
  action_type     TEXT NOT NULL,
  target_meta_id  TEXT NOT NULL,
  target_kind     TEXT NOT NULL,
  reason          TEXT NOT NULL,
  metric_snapshot JSONB,
  proposed_change JSONB,
  source          TEXT NOT NULL,
  mode            TEXT NOT NULL,
  status          TEXT NOT NULL DEFAULT 'pending',
  meta_response   JSONB,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  decided_at      TIMESTAMPTZ,
  executed_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ai_actions_user_status ON ai_actions_log(user_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_actions_target ON ai_actions_log(target_meta_id);

-- ─────────────────────────────────────────────
-- AI SAFETY RULES
-- Per-user override of the safety thresholds. NULL account_meta_id = global.
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS ai_safety_rules (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  account_meta_id TEXT,
  rule_key        TEXT NOT NULL,
  rule_value      NUMERIC NOT NULL,
  enabled         BOOLEAN NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, account_meta_id, rule_key)
);

CREATE INDEX IF NOT EXISTS idx_ai_safety_rules_user ON ai_safety_rules(user_id);
