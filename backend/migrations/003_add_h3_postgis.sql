-- Migration 003: Add H3 and PostGIS columns to location_history
-- This migration adds geospatial indexing support

-- ============================================
-- Add H3 hex column for fast geofencing queries
-- ============================================
ALTER TABLE location_history ADD COLUMN IF NOT EXISTS hex_res9 TEXT;

-- ============================================
-- Add PostGIS geometry column for spatial queries
-- ============================================
-- Check if geom column already exists before adding
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'location_history' AND column_name = 'geom'
    ) THEN
        PERFORM AddGeometryColumn('location_history', 'geom', 4326, 'POINT', 2);
    END IF;
END $$;

-- ============================================
-- Create indexes for fast queries
-- ============================================

-- B-tree index on H3 hex for exact match queries (O(1) lookup)
CREATE INDEX IF NOT EXISTS idx_location_history_hex_res9 
ON location_history (hex_res9);

-- GIST index for PostGIS spatial queries
CREATE INDEX IF NOT EXISTS idx_location_history_geom 
ON location_history USING GIST(geom);

-- Composite index for time-bounded hex queries
CREATE INDEX IF NOT EXISTS idx_location_history_hex_time 
ON location_history (hex_res9, time DESC);

-- Composite index for device location history
CREATE INDEX IF NOT EXISTS idx_location_history_device_time 
ON location_history (device_id, time DESC);

-- ============================================
-- Add geometry column to stops table if missing
-- ============================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'stops' AND column_name = 'geom'
    ) THEN
        PERFORM AddGeometryColumn('stops', 'geom', 4326, 'POINT', 2);
    END IF;
END $$;

-- Populate stops geom from lat/lng
UPDATE stops 
SET geom = ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
WHERE geom IS NULL AND latitude IS NOT NULL AND longitude IS NOT NULL;

-- GIST index for stops spatial queries
CREATE INDEX IF NOT EXISTS idx_stops_geom 
ON stops USING GIST(geom);

-- ============================================
-- TimescaleDB compression policy
-- ============================================
-- Enable compression on location_history hypertable
-- This significantly reduces storage for historical data

-- First check if already compressed
DO $$
BEGIN
    -- Try to enable compression
    BEGIN
        ALTER TABLE location_history SET (
            timescaledb.compress,
            timescaledb.compress_segmentby = 'device_id'
        );
    EXCEPTION WHEN duplicate_object THEN
        -- Already has compression settings, ignore
        NULL;
    END;
    
    -- Add compression policy (compress chunks older than 7 days)
    BEGIN
        PERFORM add_compression_policy('location_history', INTERVAL '7 days');
    EXCEPTION WHEN duplicate_object THEN
        -- Policy already exists, ignore
        NULL;
    END;
END $$;

-- ============================================
-- Create function to auto-populate geom on insert
-- ============================================
CREATE OR REPLACE FUNCTION update_location_geom()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL THEN
        NEW.geom := ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trg_location_geom'
    ) THEN
        CREATE TRIGGER trg_location_geom
        BEFORE INSERT ON location_history
        FOR EACH ROW
        EXECUTE FUNCTION update_location_geom();
    END IF;
END $$;
