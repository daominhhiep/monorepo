CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE IF NOT EXISTS users (
    id            uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         citext UNIQUE NOT NULL,
    name          text NOT NULL,
    password_hash text NOT NULL,
    roles         text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS outbox_events (
    id            uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id  text NOT NULL,
    event_type    text NOT NULL,
    subject       text NOT NULL,
    msg_id        text NOT NULL UNIQUE,
    payload       bytea NOT NULL,
    headers       jsonb,
    published_at  timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
    ON outbox_events (created_at) WHERE published_at IS NULL;
