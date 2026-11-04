-- Seed Pooja Samagri & Household Items for all 6 catalog poojas

-- Clean any existing items
DELETE FROM pooja_samagri_items;

-- 1. Gruhapravesam (House Warming)
INSERT INTO pooja_samagri_items (pooja_id, item_name, category, quantity, is_mandatory, estimated_price) VALUES
('994c148d-a091-4531-8628-ed78f71d075f', 'Brass / Copper Puja Plates (Thalis)', 'Devotee Household Items', '3 nos', true, 0.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Brass Panchapatra & Udharani', 'Devotee Household Items', '1 set', true, 0.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Brass Deepam / Kuthu Vilakku (Diyas)', 'Devotee Household Items', '2 nos', true, 0.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Pure Cow Ghee / Sesame Oil for Deepam', 'Devotee Household Items', '500g', true, 350.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Matchbox & Cotton Wicks (Vattulu)', 'Devotee Household Items', '2 packs', true, 30.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Puja Bell (Ghanta) & Arati Harati Plate', 'Devotee Household Items', '1 set', true, 0.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Sitting Mats (Asanam) for Devotee & Priest', 'Devotee Household Items', '3 nos', true, 0.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Kalasam Pot (Copper/Brass) & 2 Fresh Coconuts', 'Devotee Household Items', '1 set', true, 100.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Raw Rice (Biyyam)', 'Devotee Household Items', '2 kg', true, 120.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Fresh Fruits (Bananas, Apples, etc.)', 'Devotee Household Items', '5 varieties', true, 250.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Fresh Betel Leaves & Supari Nuts (Tamalapakulu)', 'Devotee Household Items', '30 leaves', true, 60.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Fresh Mango Leaves Toranam (Mavidaakulu)', 'Devotee Household Items', '2 bundles', true, 50.0),
-- Sacred Samagri (Priest Kit)
('994c148d-a091-4531-8628-ed78f71d075f', 'Haldi (Pasupu) & Kumkuma', 'Sacred Ritual Samagri', '150g each', true, 80.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Gandham (Sandalwood Paste) & Akshatulu', 'Sacred Ritual Samagri', '1 box', true, 70.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Dhoop Cones & Fragrant Agarbatti', 'Sacred Ritual Samagri', '2 packs', true, 90.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Bhimseni Karpooram (Pure Camphor)', 'Sacred Ritual Samagri', '100g', true, 140.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Navadhanya Kit (9 Sacred Cosmic Grains)', 'Sacred Ritual Samagri', '1 kit', true, 150.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Homa Kunda Samidhalu (Dried Fig/Peepal twigs)', 'Sacred Ritual Samagri', '2 bundles', true, 200.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Dry Coconut Cups (Kobbari Kudakalu for Purnahuti)', 'Sacred Ritual Samagri', '2 nos', true, 120.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Sacred Janeu (Yagnopaveetam) & Darbha Pavitram', 'Sacred Ritual Samagri', '2 sets', true, 60.0),
('994c148d-a091-4531-8628-ed78f71d075f', 'Ganga Jal / Sacred Teertham', 'Sacred Ritual Samagri', '100ml bottle', true, 50.0);

-- 2. Sri Satyanarayana Swamy Vratam
INSERT INTO pooja_samagri_items (pooja_id, item_name, category, quantity, is_mandatory, estimated_price) VALUES
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Brass / Copper Puja Plates (Thalis)', 'Devotee Household Items', '3 nos', true, 0.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Brass Panchapatra & Udharani', 'Devotee Household Items', '1 set', true, 0.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Brass Deepam / Diyas', 'Devotee Household Items', '2 nos', true, 0.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Cow Ghee for Prasadam & Deepam', 'Devotee Household Items', '500g', true, 350.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Wheat Sooji / Godhuma Rava for Prasadam', 'Devotee Household Items', '500g', true, 50.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Sugar / Jaggery & Milk for Prasadam', 'Devotee Household Items', '500g', true, 60.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Sitting Mats (Asanam)', 'Devotee Household Items', '2 nos', true, 0.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Raw Rice (Biyyam) for Mandapam', 'Devotee Household Items', '2 kg', true, 120.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Fresh Bananas & Variety Fruits', 'Devotee Household Items', '2 dozens', true, 180.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Fresh Betel Leaves & Supari (Tamalapakulu)', 'Devotee Household Items', '25 leaves', true, 50.0),
-- Sacred Samagri (Priest Kit)
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Haldi (Pasupu) & Kumkuma', 'Sacred Ritual Samagri', '100g each', true, 60.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Akshatulu & Gandham Paste', 'Sacred Ritual Samagri', '1 box', true, 50.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Satyanarayana Vrata Katha Book & Navagraha Chart', 'Sacred Ritual Samagri', '1 set', true, 80.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Dhoop Sticks & Camphor (Karpooram)', 'Sacred Ritual Samagri', '1 pack', true, 100.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Navadhanya Pack (9 Grains)', 'Sacred Ritual Samagri', '1 pack', true, 120.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Panchamrutam Ingredients (Honey, Elachi, Ganga Jal)', 'Sacred Ritual Samagri', '1 kit', true, 110.0),
('5dccda01-fcb9-4bc2-9ef9-6015419533bc', 'Sacred Yagnopaveetam & Raksha Kankanam', 'Sacred Ritual Samagri', '2 sets', true, 50.0);

-- 3. Maha Ganapathi Homa / Yagam
INSERT INTO pooja_samagri_items (pooja_id, item_name, category, quantity, is_mandatory, estimated_price) VALUES
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Brass / Copper Puja Plates', 'Devotee Household Items', '3 nos', true, 0.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Brass Panchapatra & Spoon', 'Devotee Household Items', '1 set', true, 0.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Deepam Kundulu / Brass Diyas', 'Devotee Household Items', '2 nos', true, 0.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Pure Cow Ghee for Homa Havis', 'Devotee Household Items', '1 kg', true, 700.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Modak / Laddu Prasadam', 'Devotee Household Items', '21 nos', true, 200.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Raw Rice (Biyyam)', 'Devotee Household Items', '1 kg', true, 60.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Garika Grass (Durva) & Red Flowers', 'Devotee Household Items', '1 bundle', true, 50.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Fresh Coconuts', 'Devotee Household Items', '3 nos', true, 120.0),
-- Sacred Samagri (Priest Kit)
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Modaka Dravyam & Homa Herbs Mix', 'Sacred Ritual Samagri', '500g', true, 250.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Homa Samidhalu (Sacred Wood Sticks)', 'Sacred Ritual Samagri', '3 bundles', true, 300.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Dry Coconut Cups (Kobbari Kudakalu)', 'Sacred Ritual Samagri', '4 nos', true, 160.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Ashta Dravyam Ganapathi Homa Pack', 'Sacred Ritual Samagri', '1 pack', true, 350.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Bhimseni Karpooram & Dhoop', 'Sacred Ritual Samagri', '1 box', true, 120.0),
('91b8ca3e-dfbc-4626-8bca-c2c64123b94e', 'Pasupu, Kumkuma, Akshatulu, Gandham', 'Sacred Ritual Samagri', '1 set', true, 90.0);

-- 4. Namakaranam & Ayushya Homa
INSERT INTO pooja_samagri_items (pooja_id, item_name, category, quantity, is_mandatory, estimated_price) VALUES
('e81975cc-49bd-497c-bc4c-a4d3153cc4e0', 'Gold/Silver Ring or Spoon for Honey', 'Devotee Household Items', '1 no', true, 0.0),
('e81975cc-49bd-497c-bc4c-a4d3153cc4e0', 'Bronze/Brass Plate with Raw Rice (Biyyam Plate for writing name)', 'Devotee Household Items', '1 set', true, 0.0),
('e81975cc-49bd-497c-bc4c-a4d3153cc4e0', 'New Clothes for Baby (Pattu Battalu)', 'Devotee Household Items', '1 set', true, 0.0),
('e81975cc-49bd-497c-bc4c-a4d3153cc4e0', 'Pure Cow Ghee & Honey (Tené)', 'Devotee Household Items', '500g', true, 400.0),
('e81975cc-49bd-497c-bc4c-a4d3153cc4e0', 'Fresh Betel Leaves & Fruits', 'Devotee Household Items', '25 leaves', true, 100.0),
-- Sacred Samagri (Priest Kit)
('e81975cc-49bd-497c-bc4c-a4d3153cc4e0', 'Ayushya Homa Dravyam Herbs Mix', 'Sacred Ritual Samagri', '500g', true, 280.0),
('e81975cc-49bd-497c-bc4c-a4d3153cc4e0', 'Ayushya Sukta Samidhalu', 'Sacred Ritual Samagri', '2 bundles', true, 200.0),
('e81975cc-49bd-497c-bc4c-a4d3153cc4e0', 'Pasupu, Kumkuma, Chandanam', 'Sacred Ritual Samagri', '1 kit', true, 80.0),
('e81975cc-49bd-497c-bc4c-a4d3153cc4e0', 'Sacred Raksha Kankanam & Vibhooti', 'Sacred Ritual Samagri', '1 set', true, 60.0);

-- 5. Mrityunjaya Homa / Shanti
INSERT INTO pooja_samagri_items (pooja_id, item_name, category, quantity, is_mandatory, estimated_price) VALUES
('61b03f23-c57b-443f-9e2c-f256ebdd3720', 'Brass Plates & Panchapatra', 'Devotee Household Items', '3 nos', true, 0.0),
('61b03f23-c57b-443f-9e2c-f256ebdd3720', 'Brass Deepam Diyas & Ghee', 'Devotee Household Items', '1 kg', true, 700.0),
('61b03f23-c57b-443f-9e2c-f256ebdd3720', 'Fresh Bilva Leaves (Maredu Dalalu)', 'Devotee Household Items', '108 nos', true, 100.0),
('61b03f23-c57b-443f-9e2c-f256ebdd3720', 'Sitting Mats & Kalasam', 'Devotee Household Items', '1 set', true, 0.0),
-- Sacred Samagri (Priest Kit)
('61b03f23-c57b-443f-9e2c-f256ebdd3720', 'Maha Mrityunjaya Homa Herbs Dravyam', 'Sacred Ritual Samagri', '1 kg', true, 450.0),
('61b03f23-c57b-443f-9e2c-f256ebdd3720', 'Amruta Valli / Guduchi (Tippa Teega twigs)', 'Sacred Ritual Samagri', '1 bundle', true, 200.0),
('61b03f23-c57b-443f-9e2c-f256ebdd3720', 'Palasha Samidhalu & Sesame Seeds (Til)', 'Sacred Ritual Samagri', '500g', true, 180.0),
('61b03f23-c57b-443f-9e2c-f256ebdd3720', 'Pure Vibhooti / Bhasmam & Chandanam', 'Sacred Ritual Samagri', '1 box', true, 90.0);

-- 6. Navagraha Shanti Homa
INSERT INTO pooja_samagri_items (pooja_id, item_name, category, quantity, is_mandatory, estimated_price) VALUES
('b0440927-e9a4-481b-87b5-351708baebb0', 'Brass Plates & Panchapatra', 'Devotee Household Items', '3 nos', true, 0.0),
('b0440927-e9a4-481b-87b5-351708baebb0', 'Pure Cow Ghee', 'Devotee Household Items', '1 kg', true, 700.0),
('b0440927-e9a4-481b-87b5-351708baebb0', 'Navagraha Vastram (9 colored cloth pieces)', 'Devotee Household Items', '9 pieces', true, 180.0),
('b0440927-e9a4-481b-87b5-351708baebb0', 'Raw Rice & Jaggery', 'Devotee Household Items', '2 kg', true, 140.0),
-- Sacred Samagri (Priest Kit)
('b0440927-e9a4-481b-87b5-351708baebb0', 'Navadhanya Complete Kit (Wheat, Rice, Toor, Moong, Chana, Rajma, Til, Urad, Horsegram)', 'Sacred Ritual Samagri', '9 packets', true, 250.0),
('b0440927-e9a4-481b-87b5-351708baebb0', 'Navagraha 9 Samidha Sticks (Arka, Palasa, Khadira, Apamarga, Pippala, Audumbara, Sami, Durva, Kusa)', 'Sacred Ritual Samagri', '9 bundles', true, 400.0),
('b0440927-e9a4-481b-87b5-351708baebb0', 'Dry Coconut Cups (Kobbari Kudakalu)', 'Sacred Ritual Samagri', '3 nos', true, 150.0),
('b0440927-e9a4-481b-87b5-351708baebb0', 'Bhimseni Karpooram, Dhoop & Akshatulu', 'Sacred Ritual Samagri', '1 box', true, 130.0);
