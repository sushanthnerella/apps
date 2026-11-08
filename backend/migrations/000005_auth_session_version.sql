-- Migration 000005: Add session_version for auth token revocation
-- Ensures sessions can be revoked on password change or security events

ALTER TABLE users 
    ADD COLUMN IF NOT EXISTS session_version INT NOT NULL DEFAULT 1;

ALTER TABLE priests 
    ADD COLUMN IF NOT EXISTS session_version INT NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_users_session_version ON users(id, session_version);
CREATE INDEX IF NOT EXISTS idx_priests_session_version ON priests(id, session_version);
