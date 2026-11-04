-- Migration 000003: Comprehensive Puja Vaahini Ecosystem
-- Adds: Slots, Custom Services, Samagri Checklists, Puja Store Vendors, Chat, Referrals, Temples & VIP/E-Pooja Ticketing, Profile Pictures

-- 1. Profile Pictures for Users and Priests
ALTER TABLE users ADD COLUMN IF NOT EXISTS profile_picture_url TEXT DEFAULT '';
ALTER TABLE priests ADD COLUMN IF NOT EXISTS profile_picture_url TEXT DEFAULT '';

-- 2. Priest Availability Slots
CREATE TABLE IF NOT EXISTS priest_availability_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    priest_id UUID REFERENCES priests(id) ON DELETE CASCADE,
    slot_date DATE NOT NULL,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_booked BOOLEAN DEFAULT FALSE,
    booked_booking_id UUID REFERENCES bookings(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_priest_slot UNIQUE (priest_id, slot_date, start_time)
);
CREATE INDEX IF NOT EXISTS idx_slots_priest_date ON priest_availability_slots(priest_id, slot_date, is_booked);

-- 3. Priest Custom Services & Dynamic Pricing
CREATE TABLE IF NOT EXISTS priest_custom_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    priest_id UUID REFERENCES priests(id) ON DELETE CASCADE,
    title VARCHAR(150) NOT NULL,
    category VARCHAR(50) DEFAULT 'Special Ritual',
    dakshina_fee NUMERIC(10, 2) NOT NULL,
    samagri_kit_price NUMERIC(10, 2) DEFAULT 0.00,
    duration_minutes INT DEFAULT 90,
    is_online_eligible BOOLEAN DEFAULT TRUE,
    samagri_checklist JSONB DEFAULT '[]'::jsonb,
    description TEXT DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_custom_services_priest ON priest_custom_services(priest_id, is_active);

-- 4. Samagri Checklist per Pooja & Booking Customizations
CREATE TABLE IF NOT EXISTS pooja_samagri_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pooja_id UUID REFERENCES pooja_catalog(id) ON DELETE CASCADE,
    item_name VARCHAR(100) NOT NULL,
    category VARCHAR(50) DEFAULT 'Dry Essentials', -- 'Fresh Items', 'Dry Essentials', 'Havan Samagri', 'Utensils'
    quantity VARCHAR(30) NOT NULL,
    is_mandatory BOOLEAN DEFAULT TRUE,
    estimated_price NUMERIC(10, 2) DEFAULT 0.00
);

CREATE TABLE IF NOT EXISTS booking_materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID REFERENCES bookings(id) ON DELETE CASCADE,
    item_name VARCHAR(100) NOT NULL,
    category VARCHAR(50) DEFAULT 'Dry Essentials',
    quantity VARCHAR(30) NOT NULL,
    is_included_by_priest BOOLEAN DEFAULT TRUE,
    is_fulfilled_by_vendor BOOLEAN DEFAULT FALSE,
    vendor_order_id UUID
);
CREATE INDEX IF NOT EXISTS idx_booking_materials_booking ON booking_materials(booking_id);

-- 5. Local Puja Store Vendors & Inventory
CREATE TABLE IF NOT EXISTS vendors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shop_name VARCHAR(150) NOT NULL,
    owner_name VARCHAR(100) NOT NULL,
    phone VARCHAR(20) UNIQUE NOT NULL,
    profile_picture_url TEXT DEFAULT '',
    address TEXT NOT NULL,
    city VARCHAR(100) DEFAULT 'Hyderabad',
    latitude DOUBLE PRECISION DEFAULT 17.3850,
    longitude DOUBLE PRECISION DEFAULT 78.4867,
    rating_avg NUMERIC(3, 2) DEFAULT 4.90,
    is_open BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_vendors_geo ON vendors(latitude, longitude, is_open);

CREATE TABLE IF NOT EXISTS vendor_inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor_id UUID REFERENCES vendors(id) ON DELETE CASCADE,
    item_name VARCHAR(150) NOT NULL,
    category VARCHAR(50) DEFAULT 'Pooja Samagri',
    price NUMERIC(10, 2) NOT NULL,
    stock_quantity INT DEFAULT 100,
    unit VARCHAR(20) DEFAULT 'packet',
    image_url TEXT DEFAULT '',
    is_available BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_inventory_vendor ON vendor_inventory(vendor_id, is_available);

CREATE TABLE IF NOT EXISTS store_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    vendor_id UUID REFERENCES vendors(id) ON DELETE RESTRICT,
    booking_id UUID REFERENCES bookings(id) ON DELETE SET NULL,
    status VARCHAR(30) DEFAULT 'PLACED', -- 'PLACED', 'PACKED', 'DISPATCHED', 'DELIVERED', 'CANCELLED'
    total_amount NUMERIC(10, 2) NOT NULL,
    delivery_address TEXT NOT NULL,
    delivery_slot VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS store_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES store_orders(id) ON DELETE CASCADE,
    inventory_item_id UUID REFERENCES vendor_inventory(id) ON DELETE RESTRICT,
    item_name VARCHAR(150) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    unit_price NUMERIC(10, 2) NOT NULL,
    subtotal NUMERIC(10, 2) NOT NULL
);

-- 6. Real-time In-App Chat (Devotee <-> Priest)
CREATE TABLE IF NOT EXISTS chat_rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    priest_id UUID REFERENCES priests(id) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID REFERENCES chat_rooms(id) ON DELETE CASCADE,
    sender_id UUID NOT NULL,
    sender_role VARCHAR(20) NOT NULL, -- 'devotee', 'priest', 'system'
    message_type VARCHAR(20) DEFAULT 'text', -- 'text', 'image', 'location', 'system_event'
    content TEXT NOT NULL,
    attachment_url TEXT DEFAULT '',
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_chat_messages_room ON chat_messages(room_id, created_at ASC);

-- 7. Priest Referral & Instant Reassignment
CREATE TABLE IF NOT EXISTS priest_referrals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID REFERENCES bookings(id) ON DELETE CASCADE,
    referring_priest_id UUID REFERENCES priests(id) ON DELETE SET NULL,
    referred_priest_id UUID REFERENCES priests(id) ON DELETE SET NULL,
    reason TEXT DEFAULT '',
    reassigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_referrals_booking ON priest_referrals(booking_id);

-- 8. Temples, VIP Ticketing, Live E-Pooja Video Calls & CMS
CREATE TABLE IF NOT EXISTS temples (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    deity VARCHAR(100) NOT NULL,
    description TEXT DEFAULT '',
    city VARCHAR(100) NOT NULL,
    state VARCHAR(50) DEFAULT 'Telangana',
    latitude DOUBLE PRECISION DEFAULT 17.3850,
    longitude DOUBLE PRECISION DEFAULT 78.4867,
    banner_image_url TEXT DEFAULT '',
    timings VARCHAR(100) DEFAULT '06:00 AM - 08:30 PM',
    contact_phone VARCHAR(20) DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS temple_vip_pujas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    temple_id UUID REFERENCES temples(id) ON DELETE CASCADE,
    title VARCHAR(150) NOT NULL,
    description TEXT DEFAULT '',
    price NUMERIC(10, 2) NOT NULL,
    max_daily_slots INT DEFAULT 20,
    duration_minutes INT DEFAULT 45,
    is_online_video_call BOOLEAN DEFAULT FALSE, -- Online live WebRTC video call vs In-person VIP puja
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS temple_vip_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    temple_id UUID REFERENCES temples(id) ON DELETE RESTRICT,
    vip_puja_id UUID REFERENCES temple_vip_pujas(id) ON DELETE RESTRICT,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    devotee_name VARCHAR(100) NOT NULL,
    devotee_gotram VARCHAR(50),
    devotee_nakshatram VARCHAR(50),
    devotee_phone VARCHAR(20) NOT NULL,
    visit_date DATE NOT NULL,
    slot_time VARCHAR(30) NOT NULL,
    ticket_type VARCHAR(30) DEFAULT 'VIP_IN_PERSON', -- 'VIP_IN_PERSON' or 'ONLINE_E_POOJA_VIDEO'
    qr_token TEXT NOT NULL,
    status VARCHAR(30) DEFAULT 'CONFIRMED', -- 'CONFIRMED', 'CHECKED_IN', 'COMPLETED', 'CANCELLED'
    amount_paid NUMERIC(10, 2) NOT NULL,
    payment_mode VARCHAR(20) DEFAULT 'ONLINE_ESCROW', -- 'ONLINE_ESCROW', 'CASH_AT_COUNTER'
    check_in_timestamp TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_temple_tickets_qr ON temple_vip_tickets(qr_token);
CREATE INDEX IF NOT EXISTS idx_temple_tickets_date ON temple_vip_tickets(temple_id, visit_date, status);

CREATE TABLE IF NOT EXISTS temple_admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    temple_id UUID REFERENCES temples(id) ON DELETE CASCADE,
    email VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    role VARCHAR(30) DEFAULT 'TEMPLE_ADMIN', -- 'TEMPLE_ADMIN', 'TICKET_CLERK', 'CHIEF_ARCHAKA'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed Sample Data: Dhapurmari Temple & VIP / E-Pooja Offerings
INSERT INTO temples (name, slug, deity, description, city, state, latitude, longitude, timings, contact_phone)
VALUES (
    'Dhapurmari Sri Lakshmi Narasimha Swamy Temple',
    'dhapurmari-temple',
    'Lord Sri Lakshmi Narasimha Swamy',
    'Ancient and revered sacred shrine known for powerful blessings, dosha nivarana, and divine peace.',
    'Hyderabad',
    'Telangana',
    17.3850,
    78.4867,
    '05:30 AM - 01:00 PM, 04:30 PM - 08:30 PM',
    '+91 94400 11223'
) ON CONFLICT (slug) DO NOTHING;

INSERT INTO temple_vip_pujas (temple_id, title, description, price, max_daily_slots, duration_minutes, is_online_video_call)
SELECT id, 'Special VIP Abhishekam & Sahasranama Archana', 'Direct sanctum entry for family with special sankalpam and seshavastram.', 1116.0, 15, 45, false
FROM temples WHERE slug = 'dhapurmari-temple'
AND NOT EXISTS (SELECT 1 FROM temple_vip_pujas WHERE title LIKE 'Special VIP Abhishekam%');

INSERT INTO temple_vip_pujas (temple_id, title, description, price, max_daily_slots, duration_minutes, is_online_video_call)
SELECT id, 'Live Temple E-Pooja Video Call (Two-Way Sankalpam)', 'Personalized WebRTC live video darshan and archana conducted by head priest inside sanctum.', 2116.0, 10, 60, true
FROM temples WHERE slug = 'dhapurmari-temple'
AND NOT EXISTS (SELECT 1 FROM temple_vip_pujas WHERE title LIKE 'Live Temple E-Pooja%');

-- Seed Sample Local Pooja Store Vendors
INSERT INTO vendors (shop_name, owner_name, phone, address, city, latitude, longitude, rating_avg)
VALUES (
    'Sri Sai Vedic Pooja Store & Samagri Center',
    'Ramesh Kumar Sharma',
    '+91 98490 55443',
    'Shop #4, Near Temple Arch, Main Road, Hyderabad',
    'Hyderabad',
    17.3860,
    78.4875,
    4.95
) ON CONFLICT (phone) DO NOTHING;

-- Seed Sample Vendor Inventory
INSERT INTO vendor_inventory (vendor_id, item_name, category, price, stock_quantity, unit)
SELECT id, 'Pure Cow Ghee (A2 Vedic Bilona)', 'Havan Essentials', 450.0, 50, '500g Jar'
FROM vendors WHERE phone = '+91 98490 55443'
AND NOT EXISTS (SELECT 1 FROM vendor_inventory WHERE item_name LIKE 'Pure Cow Ghee%');

INSERT INTO vendor_inventory (vendor_id, item_name, category, price, stock_quantity, unit)
SELECT id, 'Deluxe Havan Samagri Bundle (32 Herbs)', 'Havan Essentials', 350.0, 100, '1kg Pack'
FROM vendors WHERE phone = '+91 98490 55443'
AND NOT EXISTS (SELECT 1 FROM vendor_inventory WHERE item_name LIKE 'Deluxe Havan%');

INSERT INTO vendor_inventory (vendor_id, item_name, category, price, stock_quantity, unit)
SELECT id, 'Traditional Brass Panchapatra & Udharani', 'Utensils & Brassware', 550.0, 25, '1 Set'
FROM vendors WHERE phone = '+91 98490 55443'
AND NOT EXISTS (SELECT 1 FROM vendor_inventory WHERE item_name LIKE 'Traditional Brass%');
