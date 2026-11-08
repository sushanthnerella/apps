-- Migration 000004: Runtime Schema Completion & Integrity
-- Fixes missing columns used by runtime queries across auth, bookings, catalog, and custom services.
-- Safe and idempotent forward migration. Preserves all existing records, UUIDs, and foreign keys.

-- 1. Authentication: Add password_hash to users and priests (if not present)
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) DEFAULT '';
ALTER TABLE priests ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255) DEFAULT '';

-- 2. Bookings: Add detail, schedule, and cancellation fields required by runtime SQL
-- Fields are strictly nullable without fabricating customer personal or schedule details.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS scheduled_date VARCHAR(50);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS slot_time VARCHAR(100);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS devotee_gotram VARCHAR(50);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS devotee_nakshatram VARCHAR(50);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS devotee_rashi VARCHAR(50);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS sankalpam_purpose TEXT;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cancellation_reason TEXT;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS cancelled_by VARCHAR(50);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS refund_amount NUMERIC(10, 2) DEFAULT 0.00;

-- 3. Preserve referential integrity for pooja_id foreign key on bookings
-- Ensures invalid booking references remain rejected. Custom-offering redesign is deferred to Task 07.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'bookings_pooja_id_fkey'
    ) THEN
        ALTER TABLE bookings 
            ADD CONSTRAINT bookings_pooja_id_fkey 
            FOREIGN KEY (pooja_id) REFERENCES pooja_catalog(id) ON DELETE RESTRICT;
    END IF;
END $$;

-- 4. Pooja Catalog: Add 3-language internationalization columns
ALTER TABLE pooja_catalog 
  ADD COLUMN IF NOT EXISTS title_en VARCHAR(200),
  ADD COLUMN IF NOT EXISTS title_te VARCHAR(200),
  ADD COLUMN IF NOT EXISTS title_hi VARCHAR(200),
  ADD COLUMN IF NOT EXISTS description_en TEXT,
  ADD COLUMN IF NOT EXISTS description_te TEXT,
  ADD COLUMN IF NOT EXISTS description_hi TEXT;

-- Backfill authentic Vedic translations for standard catalog poojas
UPDATE pooja_catalog SET 
  title_en = COALESCE(NULLIF(title_en, ''), 'Gruhapravesam (House Warming Ceremony)'),
  title_te = COALESCE(NULLIF(title_te, ''), 'గృహప్రవేశం (నూతన గృహ ప్రవేశ మహోత్సవం)'),
  title_hi = COALESCE(NULLIF(title_hi, ''), 'गृहप्रवेश पूजा एवं वास्तु शांति हवन'),
  description_en = COALESCE(NULLIF(description_en, ''), 'Traditional Vedic Griha Pravesha with Vastu Shanti, Navagraha and Ganapathi Homam.'),
  description_te = COALESCE(NULLIF(description_te, ''), 'వాస్తు శాంతి, నవగ్రహ మరియు గణపతి హోమములతో కూడిన సంపూర్ణ వేదోక్త గృహప్రవేశ క్రతువు.'),
  description_hi = COALESCE(NULLIF(description_hi, ''), 'वास्तु शांति, नवग्रह एवं गणपति हवन से संपन्न पारंपरिक वैदिक गृहप्रवेश पूजन।')
WHERE title ILIKE '%Gruhapravesam%';

UPDATE pooja_catalog SET 
  title_en = COALESCE(NULLIF(title_en, ''), 'Sri Satyanarayana Swamy Vratam'),
  title_te = COALESCE(NULLIF(title_te, ''), 'శ్రీ సత్యనారాయణ స్వామి వ్రతం'),
  title_hi = COALESCE(NULLIF(title_hi, ''), 'श्री सत्यनारायण स्वामी व्रत कथा'),
  description_en = COALESCE(NULLIF(description_en, ''), 'Auspicious Vishnu puja with five chapters of Satyanarayana katha, Prasadam distribution and blessings.'),
  description_te = COALESCE(NULLIF(description_te, ''), 'పంచాధ్యాయ వ్రత కథ, సత్యదేవుని కలశ స్థాపన మరియు పవిత్ర సపాద ప్రసాద నివేదనతో కూడిన పూజ.'),
  description_hi = COALESCE(NULLIF(description_hi, ''), 'श्री सत्यनारायण पंच-अध्याय कथा, कलश स्थापना एवं सपाद महाप्रसाद भोग सहित पावन व्रत।')
WHERE title ILIKE '%Satyanarayana%';

UPDATE pooja_catalog SET 
  title_en = COALESCE(NULLIF(title_en, ''), 'Maha Ganapathi Homa / Yagam'),
  title_te = COALESCE(NULLIF(title_te, ''), 'మహా గణపతి హోమం / యాగానుష్ఠానం'),
  title_hi = COALESCE(NULLIF(title_hi, ''), 'महा गणपति हवन / विघ्नहर्ता अनुष्ठान'),
  description_en = COALESCE(NULLIF(description_en, ''), 'Sacred fire ritual dedicated to Lord Ganesha for removal of obstacles, success in business and home peace.'),
  description_te = COALESCE(NULLIF(description_te, ''), 'సర్వ విఘ్న నివారణకు, నూతన కార్య విజయానికి మరియు శుభ ఫలితాలకై చేయు పవిత్ర వేద హోమము.'),
  description_hi = COALESCE(NULLIF(description_hi, ''), 'सर्व विघ्न बाधा निवारण, व्यापार वृद्धि एवं नवकार्यों की निर्विघ्न सिद्धि हेतु वैदिक महाहवन।')
WHERE title ILIKE '%Ganapathi%';

UPDATE pooja_catalog SET 
  title_en = COALESCE(NULLIF(title_en, ''), 'Namakaranam & Ayushya Homa'),
  title_te = COALESCE(NULLIF(title_te, ''), 'నామకరణం & ఆయుష్య హోమం'),
  title_hi = COALESCE(NULLIF(title_hi, ''), 'नामकरण संस्कार एवं आयुष्य हवन'),
  description_en = COALESCE(NULLIF(description_en, ''), 'Traditional child naming ceremony combined with Vedic Ayushya Homa for health and longevity.'),
  description_te = COALESCE(NULLIF(description_te, ''), 'శిశువునకు జన్మనక్షత్ర నామకరణం, దీర్ఘాయువు మరియు ఆరోగ్య సిద్ధి కొరకు చేయు ఆయుష్య హోమము.'),
  description_hi = COALESCE(NULLIF(description_hi, ''), 'शिशु का वैदिक नामकरण संस्कार एवं दीर्घायु, उत्तम स्वास्थ्य हेतु आयुष्य होम अनुष्ठान।')
WHERE title ILIKE '%Namakaranam%';

UPDATE pooja_catalog SET 
  title_en = COALESCE(NULLIF(title_en, ''), 'Mrityunjaya Homa / Shanti'),
  title_te = COALESCE(NULLIF(title_te, ''), 'మహా మృత్యుంజయ హోమం / ఆయుర్వృద్ధి శాంతి'),
  title_hi = COALESCE(NULLIF(title_hi, ''), 'महा मृत्युंजय हवन / आरोग्य शांति अनुष्ठान'),
  description_en = COALESCE(NULLIF(description_en, ''), 'Powerful Rudra mantra fire ritual invoking Lord Shiva for health recovery, warding off negative energies and longevity.'),
  description_te = COALESCE(NULLIF(description_te, ''), 'పరమశివుని రుద్ర మంత్రములతో అపమృత్యు దోష నివారణకు, సంపూర్ణ ఆరోగ్యానికి చేయు మహత్తర హోమము.'),
  description_hi = COALESCE(NULLIF(description_hi, ''), 'भगवान शिव के महामृत्युंजय मंत्रों से युक्त अकाल मृत्यु निवारक एवं दीर्घायु प्रदायक महाहवन।')
WHERE title ILIKE '%Mrityunjaya%';

UPDATE pooja_catalog SET 
  title_en = COALESCE(NULLIF(title_en, ''), 'Navagraha Shanti Homa'),
  title_te = COALESCE(NULLIF(title_te, ''), 'నవగ్రహ శాంతి హోమం & దోష నివారణ'),
  title_hi = COALESCE(NULLIF(title_hi, ''), 'नवग्रह शांति हवन एवं ग्रह दोष निवारण'),
  description_en = COALESCE(NULLIF(description_en, ''), 'Vedic pacification of the nine celestial planetary deities to mitigate graha doshas and attract auspicious energies.'),
  description_te = COALESCE(NULLIF(description_te, ''), 'నవధాన్యాలు, సమిధలతో తొమ్మిది గ్రహ దేవతలను తృప్తిపరచి జాతక దోషములను తొలగించు పవిత్ర హోమము.'),
  description_hi = COALESCE(NULLIF(description_hi, ''), 'नवग्रहों की प्रसन्नता, अनुकूलता एवं कुंडली के अनिष्ट ग्रह दोषों की शांति हेतु विशिष्ट वैदिक हवन।')
WHERE title ILIKE '%Navagraha%';

-- 5. Priest Custom Services: Add 3-language internationalization columns
ALTER TABLE priest_custom_services 
  ADD COLUMN IF NOT EXISTS title_en VARCHAR(200),
  ADD COLUMN IF NOT EXISTS title_te VARCHAR(200),
  ADD COLUMN IF NOT EXISTS title_hi VARCHAR(200),
  ADD COLUMN IF NOT EXISTS description_en TEXT,
  ADD COLUMN IF NOT EXISTS description_te TEXT,
  ADD COLUMN IF NOT EXISTS description_hi TEXT;

-- 6. Performance & Query Path Indexes
CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_priest_id ON bookings(priest_id);
CREATE INDEX IF NOT EXISTS idx_slots_booked ON priest_availability_slots(is_booked);
