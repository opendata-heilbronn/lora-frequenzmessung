CREATE TABLE IF NOT EXISTS sensor_data
(
    time      TIMESTAMPTZ NOT NULL,
    sensor_id varchar,
    name      varchar,
    longitude float8,
    latitude  float8,
    type      varchar,
    value     float8
);