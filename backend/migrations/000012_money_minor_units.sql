-- Migration 000012: Minor currency units (paise) for exact integer monetary accounting
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS total_amount_minor BIGINT DEFAULT 0;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS dakshina_amount_minor BIGINT DEFAULT 0;
UPDATE bookings SET 
    total_amount_minor = ROUND(total_amount * 100),
    dakshina_amount_minor = ROUND(COALESCE(dakshina_amount, 0) * 100);

ALTER TABLE pooja_catalog ADD COLUMN IF NOT EXISTS base_price_minor BIGINT DEFAULT 0;
ALTER TABLE pooja_catalog ADD COLUMN IF NOT EXISTS samagri_kit_price_minor BIGINT DEFAULT 0;
UPDATE pooja_catalog SET 
    base_price_minor = ROUND(base_price * 100),
    samagri_kit_price_minor = ROUND(COALESCE(samagri_kit_price, 0) * 100);

ALTER TABLE priest_custom_services ADD COLUMN IF NOT EXISTS dakshina_fee_minor BIGINT DEFAULT 0;
ALTER TABLE priest_custom_services ADD COLUMN IF NOT EXISTS samagri_kit_price_minor BIGINT DEFAULT 0;
UPDATE priest_custom_services SET 
    dakshina_fee_minor = ROUND(dakshina_fee * 100),
    samagri_kit_price_minor = ROUND(COALESCE(samagri_kit_price, 0) * 100);

ALTER TABLE pooja_samagri_items ADD COLUMN IF NOT EXISTS estimated_price_minor BIGINT DEFAULT 0;
UPDATE pooja_samagri_items SET 
    estimated_price_minor = ROUND(COALESCE(estimated_price, 0) * 100);

ALTER TABLE vendor_inventory ADD COLUMN IF NOT EXISTS price_minor BIGINT DEFAULT 0;
UPDATE vendor_inventory SET 
    price_minor = ROUND(price * 100);

ALTER TABLE store_orders ADD COLUMN IF NOT EXISTS total_amount_minor BIGINT DEFAULT 0;
UPDATE store_orders SET 
    total_amount_minor = ROUND(total_amount * 100);

ALTER TABLE store_order_items ADD COLUMN IF NOT EXISTS unit_price_minor BIGINT DEFAULT 0;
ALTER TABLE store_order_items ADD COLUMN IF NOT EXISTS subtotal_minor BIGINT DEFAULT 0;
UPDATE store_order_items SET 
    unit_price_minor = ROUND(unit_price * 100),
    subtotal_minor = ROUND(subtotal * 100);

ALTER TABLE temple_vip_pujas ADD COLUMN IF NOT EXISTS price_minor BIGINT DEFAULT 0;
UPDATE temple_vip_pujas SET 
    price_minor = ROUND(price * 100);

ALTER TABLE temple_vip_tickets ADD COLUMN IF NOT EXISTS amount_paid_minor BIGINT DEFAULT 0;
UPDATE temple_vip_tickets SET 
    amount_paid_minor = ROUND(amount_paid * 100);
