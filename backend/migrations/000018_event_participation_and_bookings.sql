-- Migration 000018: Event priest participation, ritual templates, priest prices, ghat duty, event cards and event bookings

CREATE TABLE IF NOT EXISTS event_rituals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    name_te TEXT NOT NULL DEFAULT '',
    name_hi TEXT NOT NULL DEFAULT '',
    duration_minutes INT NOT NULL CHECK (duration_minutes BETWEEN 5 AND 480),
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_event_rituals_name UNIQUE (event_id, name)
);

CREATE TABLE IF NOT EXISTS event_participations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    priest_id UUID NOT NULL REFERENCES priests(id) ON DELETE CASCADE,
    status VARCHAR(30) NOT NULL DEFAULT 'SUBMITTED'
        CHECK (status IN ('SUBMITTED', 'CHANGES_REQUESTED', 'APPROVED', 'REJECTED', 'SUSPENDED')),
    reviewer_reason TEXT NOT NULL DEFAULT '',
    submitted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    reviewed_by UUID,
    CONSTRAINT uq_event_participation UNIQUE (event_id, priest_id),
    -- Negative outcomes must carry a reason (PRD v2 PRI-003).
    CONSTRAINT chk_participation_reason CHECK (
        status NOT IN ('CHANGES_REQUESTED', 'REJECTED', 'SUSPENDED') OR length(trim(reviewer_reason)) > 0
    )
);

CREATE INDEX IF NOT EXISTS idx_event_participations_status ON event_participations(event_id, status);

CREATE TABLE IF NOT EXISTS event_priest_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    priest_id UUID NOT NULL REFERENCES priests(id) ON DELETE CASCADE,
    ritual_id UUID NOT NULL REFERENCES event_rituals(id) ON DELETE RESTRICT,
    price_minor BIGINT NOT NULL CHECK (price_minor > 0 AND price_minor <= 10000000),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_event_priest_service UNIQUE (priest_id, ritual_id)
);

-- A row exists only while the priest is on duty at a ghat for the event.
CREATE TABLE IF NOT EXISTS event_duty (
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    priest_id UUID NOT NULL REFERENCES priests(id) ON DELETE CASCADE,
    ghat_id UUID NOT NULL REFERENCES ghats(id) ON DELETE RESTRICT,
    duty_status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE' CHECK (duty_status IN ('AVAILABLE', 'PERFORMING', 'BREAK')),
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (event_id, priest_id)
);

CREATE INDEX IF NOT EXISTS idx_event_duty_ghat ON event_duty(event_id, ghat_id, duty_status);

-- Pooja Vahini event cards. card_code is the opaque, unguessable id printed in the QR.
CREATE TABLE IF NOT EXISTS event_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_code VARCHAR(40) NOT NULL UNIQUE,
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    priest_id UUID NOT NULL REFERENCES priests(id) ON DELETE CASCADE,
    issuer VARCHAR(30) NOT NULL DEFAULT 'POOJA_VAHINI' CHECK (issuer = 'POOJA_VAHINI'),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'REVOKED')),
    issued_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_event_credential UNIQUE (event_id, priest_id)
);

ALTER TABLE bookings ADD COLUMN IF NOT EXISTS event_id UUID REFERENCES events(id) ON DELETE RESTRICT;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS ghat_id UUID REFERENCES ghats(id) ON DELETE RESTRICT;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS event_service_id UUID REFERENCES event_priest_services(id) ON DELETE RESTRICT;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS agreement_snapshot JSONB;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS request_expires_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMP WITH TIME ZONE;

CREATE INDEX IF NOT EXISTS idx_bookings_event_expiry ON bookings(request_expires_at) WHERE status = 'REQUESTED' AND event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_bookings_event_priest_active ON bookings(priest_id, event_id)
    WHERE event_id IS NOT NULL AND status IN ('ACCEPTED', 'IN_TRANSIT', 'ARRIVED', 'POOJA_STARTED');
