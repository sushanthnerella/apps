-- Migration: Services, Icons, and Rapido Nearby Fields

ALTER TABLE priests ADD COLUMN IF NOT EXISTS services_offered TEXT[] DEFAULT ARRAY['Gruhapravesam', 'Sri Satyanarayana Swamy Vratam', 'Maha Ganapathi Homa / Yagam'];
ALTER TABLE priests ADD COLUMN IF NOT EXISTS eta_minutes INT DEFAULT 10;
ALTER TABLE priests ADD COLUMN IF NOT EXISTS vehicle_type VARCHAR(30) DEFAULT 'Two Wheeler';
ALTER TABLE pooja_catalog ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';
ALTER TABLE pooja_catalog ADD COLUMN IF NOT EXISTS icon VARCHAR(10) DEFAULT '🪔';

UPDATE pooja_catalog SET icon = '🏡', description = 'Traditional house warming ceremony with Navagraha & Vastu homam.' WHERE title LIKE 'Gruhapravesam%';
UPDATE pooja_catalog SET icon = '✨', description = 'Auspicious vratam for prosperity, abundance, and family harmony.' WHERE title LIKE 'Sri Satyanarayana%';
UPDATE pooja_catalog SET icon = '🕉️', description = 'Powerful Vedic yagam to overcome obstacles and grant success.' WHERE title LIKE 'Maha Ganapathi%';

INSERT INTO pooja_catalog (title, category, base_price, samagri_kit_price, duration_minutes, is_online_eligible, icon, description)
SELECT 'Namakaranam & Ayushya Homa', 'Sanskaras', 2116.0, 1100.0, 90, true, '👶', 'Sacred baby naming ceremony with divine blessings for longevity.'
WHERE NOT EXISTS (SELECT 1 FROM pooja_catalog WHERE title LIKE 'Namakaranam%');

INSERT INTO pooja_catalog (title, category, base_price, samagri_kit_price, duration_minutes, is_online_eligible, icon, description)
SELECT 'Mrityunjaya Homa / Shanti', 'Health & Protection', 4116.0, 2100.0, 120, true, '🔱', 'Vedic protection homam dedicated to Lord Shiva for health and healing.'
WHERE NOT EXISTS (SELECT 1 FROM pooja_catalog WHERE title LIKE 'Mrityunjaya%');

INSERT INTO pooja_catalog (title, category, base_price, samagri_kit_price, duration_minutes, is_online_eligible, icon, description)
SELECT 'Navagraha Shanti Homa', 'Astrological Remedies', 3516.0, 1600.0, 100, true, '☀️', 'Pacifies malefic planetary influences and restores dosha balance.'
WHERE NOT EXISTS (SELECT 1 FROM pooja_catalog WHERE title LIKE 'Navagraha%');

-- Update priests with varied locations, services, and ETAs (gated: only executed when purohit.seed_demo is enabled)
DO $$
BEGIN
    IF current_setting('purohit.seed_demo', true) = 'true' THEN
        UPDATE priests SET eta_minutes = 6, latitude = 17.3850, longitude = 78.4867, services_offered = ARRAY['Gruhapravesam (House Warming)', 'Sri Satyanarayana Swamy Vratam', 'Maha Ganapathi Homa / Yagam', 'Navagraha Shanti Homa'] WHERE phone = '+91 98765 43210';
        UPDATE priests SET eta_minutes = 12, latitude = 17.3910, longitude = 78.4910, services_offered = ARRAY['Sri Satyanarayana Swamy Vratam', 'Namakaranam & Ayushya Homa', 'Maha Ganapathi Homa / Yagam'] WHERE phone = '+91 98765 43211';
        UPDATE priests SET eta_minutes = 15, latitude = 17.3780, longitude = 78.4750, services_offered = ARRAY['Gruhapravesam (House Warming)', 'Mrityunjaya Homa / Shanti', 'Navagraha Shanti Homa'] WHERE phone = '+91 98765 43212';
    END IF;
END $$;
