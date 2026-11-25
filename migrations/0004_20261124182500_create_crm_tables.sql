SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '10s';

CREATE TABLE IF NOT EXISTS crm_customers (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL CHECK (char_length(btrim(name)) > 0),
    phone TEXT NOT NULL UNIQUE,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS crm_conversations (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL REFERENCES crm_customers(id) ON DELETE RESTRICT,
    channel TEXT NOT NULL DEFAULT 'whatsapp',
    status TEXT NOT NULL CHECK (status IN ('open', 'closed')),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    last_message_at TIMESTAMPTZ,
    last_message_preview TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_crm_conversations_one_open
    ON crm_conversations (customer_id)
    WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_crm_conversations_status_last_msg
    ON crm_conversations (status, last_message_at DESC);

CREATE TABLE IF NOT EXISTS crm_messages (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES crm_conversations(id) ON DELETE RESTRICT,
    external_id TEXT,
    idempotency_key TEXT,
    direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    content TEXT NOT NULL CHECK (char_length(btrim(content)) > 0),
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (
        (direction = 'inbound' AND status = 'received')
        OR (direction = 'outbound' AND status IN ('pending', 'sent', 'delivered', 'failed', 'uncertain'))
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_crm_messages_external_id
    ON crm_messages (external_id)
    WHERE external_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_crm_messages_idempotency
    ON crm_messages (conversation_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_crm_messages_conv_order
    ON crm_messages (conversation_id, created_at, id);
