-- Migration 000007: Verified Email Support
-- Adds verified email columns to users and priests, and supports email recovery tokens.

-- Devotee email support
ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (LOWER(email)) WHERE email IS NOT NULL AND email != '';

-- Priest email support
ALTER TABLE priests ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE priests ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE;
CREATE UNIQUE INDEX IF NOT EXISTS idx_priests_email ON priests (LOWER(email)) WHERE email IS NOT NULL AND email != '';

-- Recovery challenges email binding and flexible token storage
ALTER TABLE password_recovery_challenges ADD COLUMN IF NOT EXISTS email VARCHAR(255);
ALTER TABLE password_recovery_challenges ALTER COLUMN phone DROP NOT NULL;
CREATE INDEX IF NOT EXISTS idx_recovery_email ON password_recovery_challenges (account_role, LOWER(email), status);
