-- 0005 — store Meta-side campaign timestamps so we know real campaign age,
-- not just when our sync first saw it.
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS meta_created_time TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS meta_start_time   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS meta_stop_time    TIMESTAMPTZ;

ALTER TABLE ad_sets
    ADD COLUMN IF NOT EXISTS meta_created_time TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS meta_start_time   TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS meta_end_time     TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_campaigns_meta_start_time ON campaigns(meta_start_time);
