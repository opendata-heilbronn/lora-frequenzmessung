CREATE TABLE IF NOT EXISTS sensors (
  id SERIAL PRIMARY KEY,
  uuid VARCHAR NOT NULL UNIQUE,
  name VARCHAR NOT NULL,
  longitude FLOAT8 NOT NULL,
  latitude FLOAT8 NOT NULL,
  type VARCHAR NOT NULL DEFAULT 'density',
  dev_eui VARCHAR,
  app_key VARCHAR,
  ttn_device_id VARCHAR,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
