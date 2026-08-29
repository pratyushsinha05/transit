    -- Migration 004: Seed sample data
-- This migration adds sample routes, stops, and devices for testing

-- ============================================
-- Clear existing sample data (optional, for dev only)
-- ============================================
-- TRUNCATE trips, location_history, stops, routes, devices CASCADE;

-- ============================================
-- Insert sample devices (buses)
-- ============================================
INSERT INTO devices (id, name, status, created_at) VALUES
    ('bus-001', 'Bus Alpha', 'ACTIVE', NOW()),
    ('bus-002', 'Bus Beta', 'ACTIVE', NOW()),
    ('bus-003', 'Bus Gamma', 'ACTIVE', NOW()),
    ('bus-004', 'Bus Delta', 'ACTIVE', NOW()),
    ('bus-005', 'Bus Epsilon', 'ACTIVE', NOW())
ON CONFLICT (id) DO UPDATE SET status = 'ACTIVE';

-- ============================================
-- Insert sample routes
-- ============================================
INSERT INTO routes (id, name, description) VALUES
    ('route-1', 'Downtown Express', 'Main street to downtown via highway'),
    ('route-2', 'Airport Shuttle', 'City center to airport'),
    ('route-3', 'University Loop', 'Campus circular route')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

-- ============================================
-- Insert sample stops for Route 1 (Downtown Express)
-- Using New Delhi coordinates as example
-- ============================================
INSERT INTO stops (id, route_id, name, latitude, longitude, sequence_number) VALUES
    ('stop-r1-1', 'route-1', 'Central Station', 28.6139, 77.2090, 1),
    ('stop-r1-2', 'route-1', 'City Mall', 28.6280, 77.2180, 2),
    ('stop-r1-3', 'route-1', 'Tech Park', 28.6350, 77.2250, 3),
    ('stop-r1-4', 'route-1', 'Business District', 28.6420, 77.2320, 4),
    ('stop-r1-5', 'route-1', 'Downtown Terminal', 28.6500, 77.2400, 5)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude;

-- ============================================
-- Insert sample stops for Route 2 (Airport Shuttle)
-- ============================================
INSERT INTO stops (id, route_id, name, latitude, longitude, sequence_number) VALUES
    ('stop-r2-1', 'route-2', 'City Center', 28.6320, 77.2195, 1),
    ('stop-r2-2', 'route-2', 'Metro Station', 28.6450, 77.2050, 2),
    ('stop-r2-3', 'route-2', 'Highway Junction', 28.6700, 77.1800, 3),
    ('stop-r2-4', 'route-2', 'Airport Terminal 1', 28.5565, 77.1000, 4),
    ('stop-r2-5', 'route-2', 'Airport Terminal 3', 28.5552, 77.0880, 5)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude;

-- ============================================
-- Insert sample stops for Route 3 (University Loop)
-- ============================================
INSERT INTO stops (id, route_id, name, latitude, longitude, sequence_number) VALUES
    ('stop-r3-1', 'route-3', 'Main Gate', 28.5449, 77.1926, 1),
    ('stop-r3-2', 'route-3', 'Library', 28.5465, 77.1890, 2),
    ('stop-r3-3', 'route-3', 'Student Center', 28.5480, 77.1850, 3),
    ('stop-r3-4', 'route-3', 'Sports Complex', 28.5495, 77.1810, 4),
    ('stop-r3-5', 'route-3', 'Hostel Area', 28.5510, 77.1770, 5)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude;

-- ============================================
-- Populate geometry columns for stops
-- ============================================
UPDATE stops 
SET geom = ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
WHERE latitude IS NOT NULL AND longitude IS NOT NULL;

-- ============================================
-- Insert sample active trips
-- ============================================
INSERT INTO trips (id, route_id, device_id, started_at, current_stop, status) VALUES
    ('trip-001', 'route-1', 'bus-001', NOW() - INTERVAL '30 minutes', 2, 'IN_PROGRESS'),
    ('trip-002', 'route-1', 'bus-002', NOW() - INTERVAL '15 minutes', 1, 'IN_PROGRESS'),
    ('trip-003', 'route-2', 'bus-003', NOW() - INTERVAL '45 minutes', 3, 'IN_PROGRESS'),
    ('trip-004', 'route-3', 'bus-004', NOW() - INTERVAL '20 minutes', 2, 'IN_PROGRESS')
ON CONFLICT (id) DO UPDATE SET status = 'IN_PROGRESS';

-- ============================================
-- Insert sample location history (simulating bus movement)
-- ============================================
-- Bus 001 moving along Route 1
-- hex_res9 values precomputed from Go h3-go/v4 at resolution 9
-- Guarded so a double-apply cannot duplicate seed pings (DEFECT-7).
-- location_history has no unique constraint, and the NOW()-relative timestamps
-- below differ on every apply, so ON CONFLICT cannot help here. See CLAUDE.md Sec 4.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM location_history) THEN
        INSERT INTO location_history (time, device_id, latitude, longitude, speed, accuracy, hex_res9) VALUES
            (NOW() - INTERVAL '5 minutes', 'bus-001', 28.6280, 77.2180, 25.0, 10.0, '893da11408fffff'),
            (NOW() - INTERVAL '4 minutes', 'bus-001', 28.6300, 77.2200, 28.0, 8.0,  '893da11408fffff'),
            (NOW() - INTERVAL '3 minutes', 'bus-001', 28.6320, 77.2220, 30.0, 5.0,  '893da1140b3ffff'),
            (NOW() - INTERVAL '2 minutes', 'bus-001', 28.6335, 77.2235, 26.0, 6.0,  '893da1140b3ffff'),
            (NOW() - INTERVAL '1 minute', 'bus-001', 28.6350, 77.2250, 22.0, 5.0,  '893da1140b7ffff');

        -- Bus 002 at Central Station
        INSERT INTO location_history (time, device_id, latitude, longitude, speed, accuracy, hex_res9) VALUES
            (NOW() - INTERVAL '2 minutes', 'bus-002', 28.6139, 77.2090, 0.0,  5.0, '893da11462fffff'),
            (NOW() - INTERVAL '1 minute', 'bus-002', 28.6140, 77.2091, 5.0,  5.0, '893da11462fffff'),
            (NOW(),                        'bus-002', 28.6142, 77.2095, 15.0, 5.0, '893da11462fffff');

        -- Bus 003 on Airport Shuttle
        INSERT INTO location_history (time, device_id, latitude, longitude, speed, accuracy, hex_res9) VALUES
            (NOW() - INTERVAL '3 minutes', 'bus-003', 28.6700, 77.1800, 45.0, 10.0, '893da11601bffff'),
            (NOW() - INTERVAL '2 minutes', 'bus-003', 28.6400, 77.1500, 50.0, 8.0,  '893da1175c7ffff'),
            (NOW() - INTERVAL '1 minute', 'bus-003', 28.6100, 77.1200, 55.0, 5.0,  '893da1176c3ffff');
    END IF;
END $$;

-- ============================================
-- geom is populated automatically by the trg_location_geom BEFORE INSERT trigger
-- (added in migration 003). No manual UPDATE needed here.
-- hex_res9 values are pre-computed above (not NULL).
-- ============================================

-- ============================================
-- Verify data was inserted
-- ============================================
DO $$
DECLARE
    device_count INT;
    route_count INT;
    stop_count INT;
    location_count INT;
BEGIN
    SELECT COUNT(*) INTO device_count FROM devices;
    SELECT COUNT(*) INTO route_count FROM routes;
    SELECT COUNT(*) INTO stop_count FROM stops;
    SELECT COUNT(*) INTO location_count FROM location_history;
    
    RAISE NOTICE 'Seed data summary:';
    RAISE NOTICE '  Devices: %', device_count;
    RAISE NOTICE '  Routes: %', route_count;
    RAISE NOTICE '  Stops: %', stop_count;
    RAISE NOTICE '  Location records: %', location_count;
END $$;
