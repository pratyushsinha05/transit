-- Migration 005: rename stops -> zones (Phase 2 domain rename).
-- Mechanical. No data change, no column type change.
-- Guarded so it is idempotent and safe on a database where 001-004 have
-- already run and on a fresh one where they run in sequence first.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'public' AND table_name = 'stops')
       AND NOT EXISTS (SELECT 1 FROM information_schema.tables
                       WHERE table_schema = 'public' AND table_name = 'zones')
    THEN
        ALTER TABLE stops RENAME TO zones;
    END IF;
END $$;

-- Index names do not follow the table automatically.
ALTER INDEX IF EXISTS idx_stops_geom RENAME TO idx_zones_geom;
