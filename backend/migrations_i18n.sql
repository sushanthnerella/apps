ALTER TABLE pooja_catalog 
  ADD COLUMN IF NOT EXISTS title_en VARCHAR(200),
  ADD COLUMN IF NOT EXISTS title_te VARCHAR(200),
  ADD COLUMN IF NOT EXISTS title_hi VARCHAR(200),
  ADD COLUMN IF NOT EXISTS description_en TEXT,
  ADD COLUMN IF NOT EXISTS description_te TEXT,
  ADD COLUMN IF NOT EXISTS description_hi TEXT;

ALTER TABLE priest_custom_services 
  ADD COLUMN IF NOT EXISTS title_en VARCHAR(200),
  ADD COLUMN IF NOT EXISTS title_te VARCHAR(200),
  ADD COLUMN IF NOT EXISTS title_hi VARCHAR(200),
  ADD COLUMN IF NOT EXISTS description_en TEXT,
  ADD COLUMN IF NOT EXISTS description_te TEXT,
  ADD COLUMN IF NOT EXISTS description_hi TEXT;

UPDATE pooja_catalog SET 
  title_en = 'Gruhapravesam (House Warming Ceremony)',
  title_te = 'గృహప్రవేశం (నూతన గృహ ప్రవేశ మహోత్సవం)',
  title_hi = 'गृहप्रवेश पूजा एवं वास्तु शांति हवन',
  description_en = 'Traditional Vedic Griha Pravesha with Vastu Shanti, Navagraha and Ganapathi Homam.',
  description_te = 'వాస్తు శాంతి, నవగ్రహ మరియు గణపతి హోమములతో కూడిన సంపూర్ణ వేదోక్త గృహప్రవేశ క్రతువు.',
  description_hi = 'वास्तु शांति, नवग्रह एवं गणपति हवन से संपन्न पारंपरिक वैदिक गृहप्रवेश पूजन।'
WHERE title ILIKE '%Gruhapravesam%';

UPDATE pooja_catalog SET 
  title_en = 'Sri Satyanarayana Swamy Vratam',
  title_te = 'శ్రీ సత్యనారాయణ స్వామి వ్రతం',
  title_hi = 'श्री सत्यनारायण स्वामी व्रत कथा',
  description_en = 'Auspicious Vishnu puja with five chapters of Satyanarayana katha, Prasadam distribution and blessings.',
  description_te = 'పంచాధ్యాయ వ్రత కథ, సత్యదేవుని కలశ స్థాపన మరియు పవిత్ర సపాద ప్రసాద నివేదనతో కూడిన పూజ.',
  description_hi = 'श्री सत्यनारायण पंच-अध्याय कथा, कलश स्थापना एवं सपाद महाप्रसाद भोग सहित पावन व्रत।'
WHERE title ILIKE '%Satyanarayana%';

UPDATE pooja_catalog SET 
  title_en = 'Maha Ganapathi Homa / Yagam',
  title_te = 'మహా గణపతి హోమం / యాగానుష్ఠానం',
  title_hi = 'महा गणपति हवन / विघ्नहर्ता अनुष्ठान',
  description_en = 'Sacred fire ritual dedicated to Lord Ganesha for removal of obstacles, success in business and home peace.',
  description_te = 'సర్వ విఘ్న నివారణకు, నూతన కార్య విజయానికి మరియు శుభ ఫలితాలకై చేయు పవిత్ర వేద హోమము.',
  description_hi = 'सर्व विघ्न बाधा निवारण, व्यापार वृद्धि एवं नवकार्यों की निर्विघ्न सिद्धि हेतु वैदिक महाहवन।'
WHERE title ILIKE '%Ganapathi%';

UPDATE pooja_catalog SET 
  title_en = 'Namakaranam & Ayushya Homa',
  title_te = 'నామకరణం & ఆయుష్య హోమం',
  title_hi = 'नामकरण संस्कार एवं आयुष्य हवन',
  description_en = 'Traditional child naming ceremony combined with Vedic Ayushya Homa for health and longevity.',
  description_te = 'శిశువునకు జన్మనక్షత్ర నామకరణం, దీర్ఘాయువు మరియు ఆరోగ్య సిద్ధి కొరకు చేయు ఆయుష్య హోమము.',
  description_hi = 'शिशु का वैदिक नामकरण संस्कार एवं दीर्घायु, उत्तम स्वास्थ्य हेतु आयुष्य होम अनुष्ठान।'
WHERE title ILIKE '%Namakaranam%';

UPDATE pooja_catalog SET 
  title_en = 'Mrityunjaya Homa / Shanti',
  title_te = 'మహా మృత్యుంజయ హోమం / ఆయుర్వృద్ధి శాంతి',
  title_hi = 'महा मृत्युंजय हवन / आरोग्य शांति अनुष्ठान',
  description_en = 'Powerful Rudra mantra fire ritual invoking Lord Shiva for health recovery, warding off negative energies and longevity.',
  description_te = 'పరమశివుని రుద్ర మంత్రములతో అపమృత్యు దోష నివారణకు, సంపూర్ణ ఆరోగ్యానికి చేయు మహత్తర హోమము.',
  description_hi = 'भगवान शिव के महामृत्युंजय मंत्रों से युक्त अकाल मृत्यु निवारक एवं दीर्घायु प्रदायक महाहवन।'
WHERE title ILIKE '%Mrityunjaya%';

UPDATE pooja_catalog SET 
  title_en = 'Navagraha Shanti Homa',
  title_te = 'నవగ్రహ శాంతి హోమం & దోష నివారణ',
  title_hi = 'नवग्रह शांति हवन एवं ग्रह दोष निवारण',
  description_en = 'Vedic pacification of the nine celestial planetary deities to mitigate graha doshas and attract auspicious energies.',
  description_te = 'నవధాన్యాలు, సమిధలతో తొమ్మిది గ్రహ దేవతలను తృప్తిపరచి జాతక దోషములను తొలగించు పవిత్ర హోమము.',
  description_hi = 'नवग्रहों की प्रसन्नता, अनुकूलता एवं कुंडली के अनिष्ट ग्रह दोषों की शांति हेतु विशिष्ट वैदिक हवन।'
WHERE title ILIKE '%Navagraha%';

UPDATE priest_custom_services SET 
  title_en = 'Navagraha + Dhanvantari Maha Homa Special',
  title_te = 'నవగ్రహ + ధన్వంతరి మహా హోమం (ప్రత్యేక పౌరోహిత్యం)',
  title_hi = 'नवग्रह एवं धन्वंतरि महाहवन विशेष अनुष्ठान',
  description_en = 'Personalized Vedic ritual invoking planetary harmony, longevity, and family peace.',
  description_te = 'గ్రహ శాంతి, వ్యాధి నివారణ మరియు సంపూర్ణ కుటుంబ సౌభాగ్యము కొరకు ఆచార్యుల ప్రత్యేక హోమము.',
  description_hi = 'ग्रह शांति, स्वास्थ्य लाभ एवं परिवार में सुख-शांति हेतु आचार्य जी द्वारा विशेष महाहवन।'
WHERE title_en IS NULL;
