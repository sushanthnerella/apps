-- Migration 000016: Store orders and inventory pure minor units support
ALTER TABLE store_orders ALTER COLUMN total_amount DROP NOT NULL;
ALTER TABLE store_order_items ALTER COLUMN unit_price DROP NOT NULL;
ALTER TABLE store_order_items ALTER COLUMN subtotal DROP NOT NULL;
ALTER TABLE vendor_inventory ALTER COLUMN price DROP NOT NULL;
