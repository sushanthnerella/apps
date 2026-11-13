ALTER TABLE bookings ADD COLUMN IF NOT EXISTS completion_otp_attempts INTEGER NOT NULL DEFAULT 0;
-- Plain PostgreSQL remains supported for local development. Production uses PostGIS.
DO $$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'postgis') THEN
  CREATE EXTENSION IF NOT EXISTS postgis;
  EXECUTE 'CREATE INDEX IF NOT EXISTS priests_online_location_gist ON priests USING GIST ((ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography)) WHERE is_online = true AND location_confirmed = true';
 ELSE
  RAISE NOTICE 'PostGIS unavailable: using database-side spherical distance filtering';
 END IF;
END $$;
CREATE INDEX IF NOT EXISTS priests_online_latitude_idx ON priests(latitude)
WHERE is_online = true AND location_confirmed = true;
