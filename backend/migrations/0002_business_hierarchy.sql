-- Migration 0002: Meta Business Manager hierarchy + multi-segment Imovel catalog
-- Phase F1 of PLANO_GESTOR_IA.md (§3, §4, §5).
-- All Meta IDs are stored as TEXT because some exceed int64. Money is NUMERIC.
-- Tokens/secrets are stored already-encrypted by config.Service (AES-256-GCM).

-- ─────────────────────────────────────────────
-- APP CREDENTIALS (one row per Meta App the user owns)
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS app_credentials (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id                TEXT NOT NULL,
    encrypted_app_secret  TEXT NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, app_id)
);

CREATE INDEX IF NOT EXISTS idx_app_credentials_user ON app_credentials(user_id);

-- ─────────────────────────────────────────────
-- META TOKENS (long-lived user / system_user / page tokens)
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS meta_tokens (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id           TEXT NOT NULL,
    meta_user_id     TEXT,
    encrypted_token  TEXT NOT NULL,
    token_type       TEXT NOT NULL CHECK (token_type IN ('user','system_user','page')),
    scopes           TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    expires_at       TIMESTAMPTZ,
    last_refresh     TIMESTAMPTZ NOT NULL DEFAULT now(),
    is_active        BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_meta_tokens_user_active ON meta_tokens(user_id, is_active);

-- ─────────────────────────────────────────────
-- BUSINESS MANAGERS
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS business_managers (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meta_id              TEXT NOT NULL UNIQUE,
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                 TEXT,
    verification_status  TEXT,
    timezone_id          INT,
    vertical             TEXT,
    raw                  JSONB,
    synced_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_business_managers_user ON business_managers(user_id);

-- ─────────────────────────────────────────────
-- META AD ACCOUNTS (separate from legacy user_tokens.ad_account_id)
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS meta_ad_accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meta_id         TEXT NOT NULL UNIQUE,
    bm_id           UUID REFERENCES business_managers(id) ON DELETE SET NULL,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT,
    currency        TEXT,
    timezone_name   TEXT,
    account_status  INT,
    disable_reason  INT,
    spend_cap       NUMERIC(14,2),
    amount_spent    NUMERIC(14,2),
    balance         NUMERIC(14,2),
    access_kind     TEXT CHECK (access_kind IN ('owned','client','personal')),
    raw             JSONB,
    synced_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_meta_ad_accounts_user ON meta_ad_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_meta_ad_accounts_bm   ON meta_ad_accounts(bm_id);

-- ─────────────────────────────────────────────
-- META PAGES
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS meta_pages (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meta_id               TEXT NOT NULL UNIQUE,
    bm_id                 UUID REFERENCES business_managers(id) ON DELETE SET NULL,
    user_id               UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                  TEXT,
    category              TEXT,
    fan_count             BIGINT,
    encrypted_page_token  TEXT,
    ig_user_id            TEXT,
    raw                   JSONB,
    synced_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_meta_pages_bm ON meta_pages(bm_id);

-- ─────────────────────────────────────────────
-- META PIXELS
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS meta_pixels (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meta_id      TEXT NOT NULL UNIQUE,
    bm_id        UUID REFERENCES business_managers(id) ON DELETE SET NULL,
    account_id   UUID REFERENCES meta_ad_accounts(id) ON DELETE SET NULL,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT,
    last_fired   TIMESTAMPTZ,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    raw          JSONB,
    synced_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_meta_pixels_bm ON meta_pixels(bm_id);

-- ─────────────────────────────────────────────
-- META INSTAGRAM ACCOUNTS
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS meta_instagram_accounts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meta_id      TEXT NOT NULL UNIQUE,
    bm_id        UUID REFERENCES business_managers(id) ON DELETE SET NULL,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username     TEXT,
    profile_pic  TEXT,
    raw          JSONB,
    synced_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ─────────────────────────────────────────────
-- IMOVEIS (multi-segment property catalog)
-- ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS imoveis (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nome              TEXT NOT NULL,
    segmento          TEXT NOT NULL CHECK (segmento IN ('mcmv','medio','alto','comercial','terreno','lancamento')),
    cidade            TEXT,
    bairro            TEXT,
    preco_min         NUMERIC(14,2),
    preco_max         NUMERIC(14,2),
    quartos           INT,
    area_m2           NUMERIC(10,2),
    tipologia         TEXT CHECK (tipologia IN ('apartamento','casa','terreno','sala','galpao')),
    diferenciais      TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    fotos             TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    whatsapp_destino  TEXT,
    link_landing      TEXT,
    status            TEXT NOT NULL DEFAULT 'rascunho' CHECK (status IN ('rascunho','ativo','pausado','vendido')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_imoveis_user_status ON imoveis(user_id, status) WHERE deleted_at IS NULL;

-- ─────────────────────────────────────────────
-- Applied changes summary:
--   + app_credentials, meta_tokens
--   + business_managers, meta_ad_accounts, meta_pages, meta_pixels, meta_instagram_accounts
--   + imoveis
-- ─────────────────────────────────────────────
