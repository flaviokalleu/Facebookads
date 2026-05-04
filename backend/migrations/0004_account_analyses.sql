-- 0004 — per-account AI analysis cache
CREATE TABLE account_analyses (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_meta_id TEXT NOT NULL,
    summary         TEXT NOT NULL,
    highlights      JSONB,
    model_used      TEXT,
    input_tokens    INTEGER NOT NULL DEFAULT 0,
    output_tokens   INTEGER NOT NULL DEFAULT 0,
    cost_usd        NUMERIC(10,6) NOT NULL DEFAULT 0,
    latency_ms      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_account_analyses_user_account ON account_analyses(user_id, account_meta_id, created_at DESC);
