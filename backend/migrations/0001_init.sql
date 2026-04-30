-- Migration 0001: Core tables

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─────────────────────────────────────────────
-- SYSTEM CONFIGURATION
-- All keys/secrets stored here. .env has DATABASE_URL only.
-- ─────────────────────────────────────────────
CREATE TABLE system_config (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key         TEXT NOT NULL UNIQUE,
    value       TEXT NOT NULL,
    is_secret   BOOLEAN NOT NULL DEFAULT false,
    description TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_system_config_key ON system_config(key);

-- ─────────────────────────────────────────────
-- USERS
-- ─────────────────────────────────────────────
CREATE TABLE users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name         TEXT NOT NULL,
    is_admin     BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;

-- ─────────────────────────────────────────────
-- USER TOKENS (Meta Ads OAuth)
-- ─────────────────────────────────────────────
CREATE TABLE user_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ad_account_id   TEXT NOT NULL,
    encrypted_token TEXT NOT NULL,
    token_expiry    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, ad_account_id)
);

CREATE INDEX idx_user_tokens_user_id ON user_tokens(user_id);

-- ─────────────────────────────────────────────
-- CAMPAIGNS
-- ─────────────────────────────────────────────
CREATE TABLE campaigns (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meta_campaign_id  TEXT NOT NULL,
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ad_account_id     TEXT NOT NULL,
    name              TEXT NOT NULL,
    objective         TEXT,
    status            TEXT NOT NULL DEFAULT 'PAUSED',
    daily_budget      NUMERIC(14,2),
    lifetime_budget   NUMERIC(14,2),
    health_status     TEXT NOT NULL DEFAULT 'HEALTHY',
    last_synced_at    TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    UNIQUE(user_id, meta_campaign_id)
);

CREATE INDEX idx_campaigns_user_id         ON campaigns(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_campaigns_meta_id         ON campaigns(meta_campaign_id);
CREATE INDEX idx_campaigns_health_status   ON campaigns(health_status) WHERE deleted_at IS NULL;

-- ─────────────────────────────────────────────
-- AD SETS
-- ─────────────────────────────────────────────
CREATE TABLE ad_sets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meta_ad_set_id  TEXT NOT NULL,
    campaign_id     UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PAUSED',
    daily_budget    NUMERIC(14,2),
    targeting       JSONB,
    optimization_goal TEXT,
    billing_event   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE(campaign_id, meta_ad_set_id)
);

CREATE INDEX idx_ad_sets_campaign_id ON ad_sets(campaign_id) WHERE deleted_at IS NULL;

-- ─────────────────────────────────────────────
-- ADS
-- ─────────────────────────────────────────────
CREATE TABLE ads (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meta_ad_id     TEXT NOT NULL,
    ad_set_id      UUID NOT NULL REFERENCES ad_sets(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'PAUSED',
    creative_title TEXT,
    creative_body  TEXT,
    image_url      TEXT,
    cta_type       TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    UNIQUE(ad_set_id, meta_ad_id)
);

CREATE INDEX idx_ads_ad_set_id ON ads(ad_set_id) WHERE deleted_at IS NULL;

-- ─────────────────────────────────────────────
-- CAMPAIGN INSIGHTS (daily)
-- ─────────────────────────────────────────────
CREATE TABLE campaign_insights (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id  UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    date         DATE NOT NULL,
    spend        NUMERIC(14,2) NOT NULL DEFAULT 0,
    impressions  BIGINT NOT NULL DEFAULT 0,
    clicks       BIGINT NOT NULL DEFAULT 0,
    ctr          NUMERIC(8,4) NOT NULL DEFAULT 0,
    cpc          NUMERIC(10,4) NOT NULL DEFAULT 0,
    cpm          NUMERIC(10,4) NOT NULL DEFAULT 0,
    reach        BIGINT NOT NULL DEFAULT 0,
    frequency    NUMERIC(8,4) NOT NULL DEFAULT 0,
    leads        BIGINT NOT NULL DEFAULT 0,
    purchases    BIGINT NOT NULL DEFAULT 0,
    roas         NUMERIC(10,4) NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(campaign_id, date)
);

CREATE INDEX idx_campaign_insights_campaign_date ON campaign_insights(campaign_id, date DESC);

-- ─────────────────────────────────────────────
-- ANOMALIES
-- ─────────────────────────────────────────────
CREATE TABLE anomalies (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id  UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    type         TEXT NOT NULL,
    severity     TEXT NOT NULL DEFAULT 'LOW',
    description  TEXT NOT NULL,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_anomalies_campaign_id ON anomalies(campaign_id, is_active);
CREATE INDEX idx_anomalies_active      ON anomalies(is_active, detected_at DESC);

-- ─────────────────────────────────────────────
-- RECOMMENDATIONS
-- ─────────────────────────────────────────────
CREATE TABLE recommendations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id      UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    priority         TEXT NOT NULL DEFAULT 'MEDIUM',
    category         TEXT NOT NULL,
    action           TEXT NOT NULL,
    expected_impact  TEXT,
    rationale        TEXT,
    model_used       TEXT,
    is_applied       BOOLEAN NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_recommendations_campaign_id ON recommendations(campaign_id, created_at DESC);

-- ─────────────────────────────────────────────
-- BUDGET SUGGESTIONS
-- ─────────────────────────────────────────────
CREATE TABLE budget_suggestions (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ad_account_id                   TEXT NOT NULL,
    campaign_id                     UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    current_budget                  NUMERIC(14,2),
    suggested_budget                NUMERIC(14,2),
    change_reason                   TEXT,
    should_pause                    BOOLEAN NOT NULL DEFAULT false,
    expected_roas_improvement       TEXT,
    portfolio_summary               TEXT,
    model_used                      TEXT,
    is_applied                      BOOLEAN NOT NULL DEFAULT false,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_budget_suggestions_user_id ON budget_suggestions(user_id, created_at DESC);

-- ─────────────────────────────────────────────
-- LLM USAGE TRACKING
-- ─────────────────────────────────────────────
CREATE TABLE llm_usage (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    task_type     TEXT NOT NULL,
    provider      TEXT NOT NULL,
    model         TEXT NOT NULL,
    input_tokens  INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd      NUMERIC(10,6) NOT NULL DEFAULT 0,
    latency_ms    INTEGER NOT NULL DEFAULT 0,
    success       BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_llm_usage_user_id    ON llm_usage(user_id, created_at DESC);
CREATE INDEX idx_llm_usage_provider   ON llm_usage(provider, created_at DESC);
CREATE INDEX idx_llm_usage_created_at ON llm_usage(created_at DESC);

-- ─────────────────────────────────────────────
-- CREATIVE INSIGHTS
-- ─────────────────────────────────────────────
CREATE TABLE creative_insights (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ad_account_id       TEXT NOT NULL,
    winning_patterns    JSONB,
    losing_patterns     JSONB,
    headline_insights   TEXT,
    cta_insights        TEXT,
    recommendations     JSONB,
    model_used          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_creative_insights_user_id ON creative_insights(user_id, created_at DESC);

-- ─────────────────────────────────────────────
-- LLM A/B TESTS
-- ─────────────────────────────────────────────
CREATE TABLE llm_ab_tests (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_type    TEXT NOT NULL,
    provider_a   TEXT NOT NULL,
    provider_b   TEXT NOT NULL,
    start_date   TIMESTAMPTZ NOT NULL,
    end_date     TIMESTAMPTZ NOT NULL,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_llm_ab_tests_active ON llm_ab_tests(is_active, end_date);

CREATE TABLE llm_ab_results (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_id                 UUID NOT NULL REFERENCES llm_ab_tests(id) ON DELETE CASCADE,
    provider                TEXT NOT NULL,
    response_quality_score  NUMERIC(5,2) NOT NULL DEFAULT 0,
    latency_ms              INTEGER NOT NULL DEFAULT 0,
    cost_usd                NUMERIC(10,6) NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_llm_ab_results_test_id ON llm_ab_results(test_id);
