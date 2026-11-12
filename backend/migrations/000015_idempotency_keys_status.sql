-- Migration 000015: Reserve-first idempotency tracking with status and lock expiry
ALTER TABLE idempotency_keys ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'IN_PROGRESS';
ALTER TABLE idempotency_keys ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP WITH TIME ZONE;
ALTER TABLE idempotency_keys ALTER COLUMN response_status DROP NOT NULL;
ALTER TABLE idempotency_keys ALTER COLUMN response_body DROP NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_idempotency_status'
    ) THEN
        ALTER TABLE idempotency_keys ADD CONSTRAINT chk_idempotency_status CHECK (status IN ('IN_PROGRESS', 'COMPLETED'));
    END IF;
END $$;
