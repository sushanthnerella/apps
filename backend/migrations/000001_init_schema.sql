-- Purohit Platform Database Schema (PostgreSQL 16+)

-- Users (Devotees) with Registration & Consent
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(20) UNIQUE NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    gotram VARCHAR(50),
    nakshatram VARCHAR(50),
    rashi VARCHAR(50),
    city VARCHAR(100) DEFAULT 'Hyderabad',
    consent_agreed BOOLEAN DEFAULT TRUE,
    consent_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    preferred_language VARCHAR(30) DEFAULT 'Telugu',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Priests (Ayagaryllu) with Registration & Consent
CREATE TABLE IF NOT EXISTS priests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(20) UNIQUE NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    sampradaya VARCHAR(50) NOT NULL, -- Smartha, Vaishnava, Madhwa, etc.
    veda_shakha VARCHAR(50) DEFAULT 'Rigveda',
    languages TEXT[] DEFAULT '{"Telugu"}',
    experience_years INT DEFAULT 5,
    rating_avg NUMERIC(3, 2) DEFAULT 5.00,
    total_completed_poojas INT DEFAULT 0,
    repeat_booking_count INT DEFAULT 0,
    acceptance_rate NUMERIC(3, 2) DEFAULT 1.00,
    is_online BOOLEAN DEFAULT TRUE,
    latitude DOUBLE PRECISION DEFAULT 17.3850,
    longitude DOUBLE PRECISION DEFAULT 78.4867,
    consent_agreed BOOLEAN DEFAULT TRUE,
    consent_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_priests_online ON priests(is_online);

-- Pooja Catalog
CREATE TABLE IF NOT EXISTS pooja_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(150) NOT NULL,
    category VARCHAR(50) NOT NULL,
    base_price NUMERIC(10, 2) NOT NULL,
    samagri_kit_price NUMERIC(10, 2) NOT NULL,
    duration_minutes INT NOT NULL,
    is_online_eligible BOOLEAN DEFAULT TRUE,
    samagri_checklist JSONB DEFAULT '[]'::jsonb
);

-- Bookings & Escrow State Machine
CREATE TABLE IF NOT EXISTS bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    priest_id UUID REFERENCES priests(id) ON DELETE SET NULL,
    pooja_id UUID REFERENCES pooja_catalog(id) ON DELETE RESTRICT,
    booking_type VARCHAR(30) NOT NULL DEFAULT 'INSTANT_DISPATCH',
    status VARCHAR(30) NOT NULL DEFAULT 'REQUESTED',
    escrow_status VARCHAR(30) NOT NULL DEFAULT 'HELD_IN_ESCROW',
    total_amount NUMERIC(10, 2) NOT NULL,
    dakshina_amount NUMERIC(10, 2) DEFAULT 0.00,
    include_samagri_kit BOOLEAN DEFAULT TRUE,
    completion_otp VARCHAR(6),
    scheduled_muhurtham TIMESTAMP WITH TIME ZONE,
    is_muhurtham_priest_determined BOOLEAN DEFAULT FALSE,
    event_address TEXT NOT NULL,
    latitude DOUBLE PRECISION DEFAULT 17.3850,
    longitude DOUBLE PRECISION DEFAULT 78.4867,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status);

-- WebRTC Video Rooms
CREATE TABLE IF NOT EXISTS webrtc_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID REFERENCES bookings(id) ON DELETE CASCADE,
    stream_mode VARCHAR(30) NOT NULL DEFAULT 'ONE_TO_ONE_PRIVATE',
    is_active BOOLEAN DEFAULT TRUE,
    viewer_count INT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed Pooja Catalog if empty
INSERT INTO pooja_catalog (title, category, base_price, samagri_kit_price, duration_minutes, is_online_eligible, samagri_checklist)
SELECT 'Gruhapravesam (House Warming)', 'Major Occasion', 5116.0, 2500.0, 180, false, '[{"name":"Navadhanya","quantity":"1 pack"},{"name":"Cow Ghee","quantity":"1 kg"}]'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM pooja_catalog WHERE title LIKE 'Gruhapravesam%');

INSERT INTO pooja_catalog (title, category, base_price, samagri_kit_price, duration_minutes, is_online_eligible, samagri_checklist)
SELECT 'Sri Satyanarayana Swamy Vratam', 'Prosperity & Devotion', 2516.0, 1200.0, 120, true, '[{"name":"Prasadam Rava","quantity":"500g"},{"name":"Tulasi Leaves","quantity":"1 bowl"}]'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM pooja_catalog WHERE title LIKE 'Sri Satyanarayana%');

INSERT INTO pooja_catalog (title, category, base_price, samagri_kit_price, duration_minutes, is_online_eligible, samagri_checklist)
SELECT 'Maha Ganapathi Homa / Yagam', 'Obstacle Removal', 3116.0, 1800.0, 90, true, '[{"name":"Havan Samagri","quantity":"2 kg"},{"name":"Dry Coconuts","quantity":"10 units"}]'::jsonb
WHERE NOT EXISTS (SELECT 1 FROM pooja_catalog WHERE title LIKE 'Maha Ganapathi%');

-- Seed Sample Verified Priests with Consent
INSERT INTO priests (phone, full_name, sampradaya, veda_shakha, experience_years, rating_avg, total_completed_poojas, repeat_booking_count, acceptance_rate, is_online, latitude, longitude, consent_agreed)
VALUES 
('+91 98765 43210', 'Brahmasri Venkata Sharma Garu', 'Smartha', 'Rigveda', 18, 4.95, 184, 42, 0.98, true, 17.3850, 78.4867, true),
('+91 98765 43211', 'Purohit Narayana Acharya', 'Vaishnava', 'Yajurveda', 12, 4.70, 96, 18, 0.85, true, 17.3910, 78.4910, true),
('+91 98765 43212', 'Vedam Subrahmanya Sastry', 'Smartha', 'Krishna Yajurveda', 22, 4.85, 210, 55, 0.92, true, 17.3780, 78.4750, true)
ON CONFLICT (phone) DO NOTHING;
