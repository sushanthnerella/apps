-- Migration 000006: Password Recovery Challenges
-- Opaque, cryptographically hashed challenge records bound to account identity and role.
-- Plaintext verification codes are NEVER stored in this table.

CREATE TABLE IF NOT EXISTS password_recovery_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL,
    account_role VARCHAR(20) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    purpose VARCHAR(50) NOT NULL DEFAULT 'password_reset',
    code_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    max_attempts INT NOT NULL DEFAULT 5,
    attempt_count INT NOT NULL DEFAULT 0,
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    consumed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_recovery_account ON password_recovery_challenges(account_role, account_id, status);
CREATE INDEX IF NOT EXISTS idx_recovery_phone ON password_recovery_challenges(account_role, phone, status);
CREATE INDEX IF NOT EXISTS idx_recovery_expires ON password_recovery_challenges(expires_at);
