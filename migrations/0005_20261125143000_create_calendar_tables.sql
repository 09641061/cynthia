SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '10s';

CREATE EXTENSION IF NOT EXISTS btree_gist WITH SCHEMA public;

CREATE TABLE IF NOT EXISTS calendar_people (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL CHECK (char_length(btrim(name)) > 0),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS calendar_appointments (
    id UUID PRIMARY KEY,
    person_id UUID NOT NULL REFERENCES calendar_people(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES crm_customers(id) ON DELETE RESTRICT,
    service_id UUID NOT NULL REFERENCES catalog_services(id) ON DELETE RESTRICT,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'confirmed', 'completed', 'cancelled', 'no_show')),
    channel TEXT NOT NULL CHECK (channel IN ('manual', 'whatsapp')),
    notes TEXT NOT NULL DEFAULT '',
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (ends_at > starts_at),
    CONSTRAINT ex_calendar_appointments_no_overlap EXCLUDE USING gist (
        person_id WITH =,
        tstzrange(starts_at, ends_at, '[)') WITH &&
    ) WHERE (status IN ('pending', 'confirmed'))
);

CREATE INDEX IF NOT EXISTS idx_calendar_appointments_range
    ON calendar_appointments (starts_at, ends_at);

CREATE INDEX IF NOT EXISTS idx_calendar_appointments_person_range
    ON calendar_appointments (person_id, starts_at, ends_at);

CREATE INDEX IF NOT EXISTS idx_calendar_appointments_customer
    ON calendar_appointments (customer_id, starts_at DESC);

CREATE INDEX IF NOT EXISTS idx_calendar_people_active_name
    ON calendar_people (active, name)
    WHERE active = true;
