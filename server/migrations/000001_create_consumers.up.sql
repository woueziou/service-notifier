CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE consumers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL UNIQUE,
    email_prefix VARCHAR(255) NOT NULL,
    sender_email VARCHAR(255) NOT NULL,
    api_key_hash VARCHAR(64) NOT NULL,
    active      BOOLEAN NOT NULL DEFAULT true,
    suspended   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_consumers_api_key_hash ON consumers(api_key_hash);
