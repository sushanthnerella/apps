-- Puja Vaahini: Explicit Development & Testing Seeds
-- Run ONLY for local development and test environments.
-- NEVER executed on fresh pilot production environments.

-- 1. Demo Verified Vedic Priests
INSERT INTO priests (phone, full_name, sampradaya, veda_shakha, experience_years, rating_avg, total_completed_poojas, repeat_booking_count, acceptance_rate, is_online, latitude, longitude, consent_agreed, eta_minutes, vehicle_type, services_offered, password_hash)
VALUES 
('+91 98765 43210', 'Brahmasri Venkata Sharma Garu', 'Smartha', 'Rigveda', 18, 4.95, 184, 42, 0.98, true, 17.3850, 78.4867, true, 6, 'Two Wheeler', ARRAY['Gruhapravesam (House Warming)', 'Sri Satyanarayana Swamy Vratam', 'Maha Ganapathi Homa / Yagam', 'Navagraha Shanti Homa'], ''),
('+91 98765 43211', 'Purohit Narayana Acharya', 'Vaishnava', 'Yajurveda', 12, 4.70, 96, 18, 0.85, true, 17.3910, 78.4910, true, 12, 'Two Wheeler', ARRAY['Sri Satyanarayana Swamy Vratam', 'Namakaranam & Ayushya Homa', 'Maha Ganapathi Homa / Yagam'], ''),
('+91 98765 43212', 'Vedam Subrahmanya Sastry', 'Smartha', 'Krishna Yajurveda', 22, 4.85, 210, 55, 0.92, true, 17.3780, 78.4750, true, 15, 'Two Wheeler', ARRAY['Gruhapravesam (House Warming)', 'Mrityunjaya Homa / Shanti', 'Navagraha Shanti Homa'], '')
ON CONFLICT (phone) DO UPDATE SET
    services_offered = EXCLUDED.services_offered,
    eta_minutes = EXCLUDED.eta_minutes;

-- 2. Demo Devotee Accounts
INSERT INTO users (phone, full_name, gotram, nakshatram, rashi, city, consent_agreed, password_hash)
VALUES 
('+91 94400 55667', 'Smt. Lakshmi Prasanna', 'Kashyapa', 'Rohini', 'Vrishabha', 'Hyderabad', true, ''),
('+91 91779 61089', 'Sushanth Sharma', 'Bharadwaja', 'Uttara Phalguni', 'Kanya', 'Hyderabad', true, '')
ON CONFLICT (phone) DO NOTHING;

-- 3. Demo Puja Store Vendors
INSERT INTO vendors (shop_name, owner_name, phone, address, city, latitude, longitude, rating_avg, is_open)
VALUES 
('Sri Sai Vedic Pooja Store & Samagri Center', 'Ramesh Kumar Sharma', '+91 98490 55443', 'Shop #4, Near Temple Arch, Main Road, Hyderabad', 'Hyderabad', 17.3860, 78.4875, 4.95, true)
ON CONFLICT (phone) DO NOTHING;

-- 4. Demo Vendor Inventory
INSERT INTO vendor_inventory (vendor_id, item_name, category, price, stock_quantity, unit, is_available)
SELECT id, 'Pure Cow Ghee (A2 Vedic Bilona)', 'Havan Essentials', 450.0, 50, '500g Jar', true
FROM vendors WHERE phone = '+91 98490 55443'
AND NOT EXISTS (SELECT 1 FROM vendor_inventory WHERE item_name LIKE 'Pure Cow Ghee%');

INSERT INTO vendor_inventory (vendor_id, item_name, category, price, stock_quantity, unit, is_available)
SELECT id, 'Deluxe Havan Samagri Bundle (32 Herbs)', 'Havan Essentials', 350.0, 100, '1kg Pack', true
FROM vendors WHERE phone = '+91 98490 55443'
AND NOT EXISTS (SELECT 1 FROM vendor_inventory WHERE item_name LIKE 'Deluxe Havan%');

INSERT INTO vendor_inventory (vendor_id, item_name, category, price, stock_quantity, unit, is_available)
SELECT id, 'Traditional Brass Panchapatra & Udharani', 'Utensils & Brassware', 550.0, 25, '1 Set', true
FROM vendors WHERE phone = '+91 98490 55443'
AND NOT EXISTS (SELECT 1 FROM vendor_inventory WHERE item_name LIKE 'Traditional Brass%');
