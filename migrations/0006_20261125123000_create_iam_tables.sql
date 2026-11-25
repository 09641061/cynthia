SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '10s';

CREATE TABLE IF NOT EXISTS iam_users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    first_name TEXT NOT NULL CHECK (char_length(btrim(first_name)) > 0),
    last_name TEXT NOT NULL CHECK (char_length(btrim(last_name)) > 0),
    password_hash TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'user' CHECK (type IN ('owner', 'user')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS iam_users_single_owner_idx
    ON iam_users ((TRUE))
    WHERE type = 'owner';

CREATE TABLE IF NOT EXISTS iam_roles (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE CHECK (char_length(btrim(name)) > 0),
    position INT NOT NULL CHECK (position > 0),
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_iam_roles_position
    ON iam_roles (position ASC, id ASC);

CREATE TABLE IF NOT EXISTS iam_member_roles (
    user_id UUID NOT NULL REFERENCES iam_users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES iam_roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_iam_member_roles_role
    ON iam_member_roles (role_id);

ALTER TABLE calendar_appointments
    DROP CONSTRAINT IF EXISTS calendar_appointments_person_id_fkey;

ALTER TABLE calendar_appointments
    ADD CONSTRAINT calendar_appointments_person_id_fkey
    FOREIGN KEY (person_id) REFERENCES iam_users(id) ON DELETE RESTRICT;

DROP TABLE IF EXISTS calendar_people;

ALTER TABLE calendar_appointments DROP COLUMN IF EXISTS version;
ALTER TABLE catalog_categories DROP COLUMN IF EXISTS version;
ALTER TABLE catalog_services DROP COLUMN IF EXISTS version;
ALTER TABLE crm_customers DROP COLUMN IF EXISTS version;
ALTER TABLE crm_conversations DROP COLUMN IF EXISTS version;
ALTER TABLE organizations DROP COLUMN IF EXISTS version;
