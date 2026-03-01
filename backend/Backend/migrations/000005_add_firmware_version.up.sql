ALTER TABLE sensors
  ADD COLUMN firmware_version VARCHAR,
  ADD COLUMN firmware_version_time TIMESTAMPTZ;
