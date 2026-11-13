-- Migration 000013: Priest location tracking with captured timestamp and accuracy
ALTER TABLE priests ADD COLUMN IF NOT EXISTS location_captured_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE priests ADD COLUMN IF NOT EXISTS location_accuracy_m DOUBLE PRECISION;
