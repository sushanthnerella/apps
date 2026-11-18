-- Migration 000017: Event mode (Pushkaralu) configuration, ghats and append-only audit log

CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(80) NOT NULL UNIQUE,
    name TEXT NOT NULL,
    name_te TEXT NOT NULL DEFAULT '',
    name_hi TEXT NOT NULL DEFAULT '',
    river TEXT NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT'
        CHECK (status IN ('DRAFT', 'SCHEDULED', 'ACTIVE', 'PAUSED', 'ENDED', 'ARCHIVED')),
    booking_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    card_issuance_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    govt_cards_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at TIMESTAMP WITH TIME ZONE NOT NULL,
    config_version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_events_window CHECK (ends_at > starts_at),
    -- Government-issued cards stay off until an authority signs written approval (PRD v2).
    CONSTRAINT chk_events_govt_cards_locked CHECK (govt_cards_enabled = FALSE)
);

-- Only one event may be live (ACTIVE or PAUSED) at a time, so app config is unambiguous.
CREATE UNIQUE INDEX IF NOT EXISTS uq_events_single_live
    ON events ((TRUE)) WHERE status IN ('ACTIVE', 'PAUSED');

CREATE TABLE IF NOT EXISTS ghats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    name_te TEXT NOT NULL DEFAULT '',
    name_hi TEXT NOT NULL DEFAULT '',
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_ghats_event_name UNIQUE (event_id, name),
    CONSTRAINT chk_ghats_coords CHECK (
        (latitude IS NULL AND longitude IS NULL)
        OR (latitude BETWEEN -90 AND 90 AND longitude BETWEEN -180 AND 180)
    )
);

CREATE INDEX IF NOT EXISTS idx_ghats_event ON ghats(event_id, sort_order);

CREATE TABLE IF NOT EXISTS audit_log (
    id BIGSERIAL PRIMARY KEY,
    actor_id UUID,
    actor_role VARCHAR(30) NOT NULL,
    action VARCHAR(80) NOT NULL,
    entity_type VARCHAR(40) NOT NULL,
    entity_id TEXT NOT NULL,
    before_state JSONB,
    after_state JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log(entity_type, entity_id, created_at DESC);

CREATE OR REPLACE FUNCTION audit_log_append_only() RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_audit_log_append_only ON audit_log;
CREATE TRIGGER trg_audit_log_append_only
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_append_only();
