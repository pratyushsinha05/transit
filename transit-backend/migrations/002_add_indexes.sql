-- Spatial index for stops
CREATE INDEX IF NOT EXISTS idx_stops_geom ON stops USING GIST(geom);

-- Time-series index for location history
CREATE INDEX IF NOT EXISTS idx_location_history_device_time ON location_history (device_id, time DESC);
