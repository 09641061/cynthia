SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '10s';

CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    singleton_key BOOLEAN NOT NULL DEFAULT TRUE UNIQUE CHECK (singleton_key),
    name TEXT NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 120),
    business_type TEXT NOT NULL DEFAULT '' CHECK (char_length(business_type) <= 120),
    contact_email TEXT NOT NULL DEFAULT '' CHECK (char_length(contact_email) <= 254),
    timezone TEXT NOT NULL DEFAULT 'UTC' CHECK (char_length(btrim(timezone)) BETWEEN 1 AND 120),
    business_hours JSONB NOT NULL,
    ai_instructions TEXT NOT NULL DEFAULT '' CHECK (char_length(ai_instructions) <= 4000),
    whatsapp_phone_number TEXT NOT NULL DEFAULT '' CHECK (char_length(whatsapp_phone_number) <= 16),
    whatsapp_phone_number_id TEXT NOT NULL DEFAULT '' CHECK (char_length(whatsapp_phone_number_id) <= 64),
    whatsapp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_organizations_whatsapp_enabled_requires_config CHECK (
        NOT whatsapp_enabled
        OR (whatsapp_phone_number <> '' AND whatsapp_phone_number_id <> '')
    )
);

INSERT INTO organizations (
    id,
    singleton_key,
    name,
    business_type,
    contact_email,
    timezone,
    business_hours,
    ai_instructions
) VALUES (
    gen_random_uuid(),
    TRUE,
    'Takodu',
    'Salon',
    '',
    'UTC',
    '{
        "monday":    {"is_open": true,  "open_time": "09:00", "close_time": "18:00"},
        "tuesday":   {"is_open": true,  "open_time": "09:00", "close_time": "18:00"},
        "wednesday": {"is_open": true,  "open_time": "09:00", "close_time": "18:00"},
        "thursday":  {"is_open": true,  "open_time": "09:00", "close_time": "18:00"},
        "friday":    {"is_open": true,  "open_time": "09:00", "close_time": "18:00"},
        "saturday":  {"is_open": false, "open_time": "09:00", "close_time": "18:00"},
        "sunday":    {"is_open": false, "open_time": "09:00", "close_time": "18:00"}
    }'::jsonb,
    ''
) ON CONFLICT (singleton_key) DO NOTHING;
