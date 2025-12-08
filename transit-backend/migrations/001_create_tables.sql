-- Enable extensions
CREATE EXTENSION IF NOT EXISTS timescaledb;
CREATE EXTENSION IF NOT EXISTS postgis;

-- Devices table
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    name TEXT,
    status TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Routes table
CREATE TABLE IF NOT EXISTS routes (
    id TEXT PRIMARY KEY,
    name TEXT,
    description TEXT
);

-- Stops table
CREATE TABLE IF NOT EXISTS stops (
    id TEXT PRIMARY KEY,
    route_id TEXT REFERENCES routes(id),
    name TEXT,
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    sequence_number INT,
    geom GEOMETRY(POINT, 4326)
);

-- Location History table (Hypertable)
CREATE TABLE IF NOT EXISTS location_history (
    time TIMESTAMP NOT NULL,
    device_id TEXT REFERENCES devices(id),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    speed FLOAT,
    accuracy FLOAT,
    metadata JSONB
);

-- Convert to hypertable
SELECT create_hypertable('location_history', 'time', if_not_exists => TRUE);

-- Trips table
CREATE TABLE IF NOT EXISTS trips (
    id TEXT PRIMARY KEY,
    route_id TEXT REFERENCES routes(id),
    device_id TEXT REFERENCES devices(id),
    started_at TIMESTAMP,
    current_stop INT,
    status TEXT
);
