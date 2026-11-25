ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS ai_enabled BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE crm_conversations
    ADD COLUMN IF NOT EXISTS ai_paused BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS handoff_reason TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS assistant_jobs (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES crm_conversations(id) ON DELETE CASCADE,
    trigger_message_id UUID NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    run_after TIMESTAMPTZ NOT NULL,
    locked_until TIMESTAMPTZ,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    reply_text TEXT NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_assistant_jobs_pending_conv
    ON assistant_jobs (conversation_id)
    WHERE status = 'pending';

CREATE UNIQUE INDEX IF NOT EXISTS uq_assistant_jobs_processing_conv
    ON assistant_jobs (conversation_id)
    WHERE status = 'processing';

CREATE INDEX IF NOT EXISTS idx_assistant_jobs_pending_run_after
    ON assistant_jobs (run_after)
    WHERE status = 'pending';
