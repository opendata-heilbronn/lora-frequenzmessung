# LORA Frequency Measurement

A "simple" way to measure crowd density using Bluetooth and LoRa technology.

## Introduction

"LOAR Frequency Measurement" is a project that aims to measure crowd density in a given area. The project uses Bluetooth technology to detect the presence of people and LoRa technology to send the data to a server. The server then processes the data and provides a real-time visualization of the crowd density in the area.


# Install 
ENV vars ? well todo 

# Migration 
install
```
mac
brew install golang-migrate
```
Create migration
```
migrate create -ext sql -dir db/migrations -seq create_users_table
```
Run migration
```
migrate -database "postgres://timescaledb:password@localhost:5432/postgres?sslmode=disable"  -path db/migrations up
```

# Docker 
build for backend 
```
docker build -f docker/go/backend/Dockerfile .
```
build for aggregator 
```
docker build -f docker/go/Aggregator/Dockerfile .
```


---

## Sensor-PAX (BLE crowd counter + LoRaWAN node)

The `sensor-pax` folder contains the firmware for a low‑power sensor node that estimates nearby people density by scanning Bluetooth LE devices and periodically sending the result via LoRaWAN.

- Target board: Heltec WiFi LoRa 32 V3 (ESP32‑S3 + SX1262)
- Framework/tooling: Arduino (via PlatformIO)
- Radio region: EU868 (configurable)

### How it works

- Scanning and counting (libpax)
  - The node uses `libpax` to scan for BLE devices and count unique MACs seen in a short window.
  - BLE is enabled, Wi‑Fi scanning is disabled by default.
  - Default scan window is 30 seconds with an RSSI threshold of −80 dBm to reject very weak/remote devices.
  - These defaults are set in `sensor-pax/src/pax.h`:
    - `configuration.blecounter = 1`, `configuration.blescantime = 30`
    - `configuration.wificounter = 0`
    - `configuration.ble_rssi_threshold = -80`, `configuration.wifi_rssi_threshold = -80`
  - `libpax_counter_start()` updates `count_from_libpax.pax`; the project reads this as `current_count`.

- Privacy guardrail
  - For privacy, readings with fewer than 6 detected devices are not transmitted.
  - See `sensor-pax/src/lora.h` in `DEVICE_STATE_SEND`.

- Density estimation and scaling
  - A simple linear scaling converts raw device count to an estimated people density: `guess = current_count * factor`.
  - Default `factor` = `0.7` (edit in `sensor-pax/src/customs.h`).

- Battery measurement
  - Battery voltage is measured on the Heltec V3 using the on‑board divider and ADC.
  - Code: `sensor-pax/src/main.cpp` (`readBatteryVoltage()` + `calculateBatteryPercentage()`), using a divider ratio of `4.9`, min/max 3.3–4.2 V → 0–100%.

- LoRaWAN uplink cycle
  - Class A, OTAA by default, ADR enabled, unconfirmed uplinks to reduce airtime.
  - A duty cycle timer controls how often data is sent; default every 900 seconds (15 min).
  - State machine in `sensor-pax/src/lora.h` handles join → send → sleep.

### Uplink payload format

The firmware builds a compact CSV string payload that is parsed on the backend:

```
<sensor_id>,0,<density>,1,<battery>
```

- `sensor_id`: 8‑character ID string (set in `customs.h`, default `863f75b0`)
- `0,<density>`: measurement type `0` = frequency/density (scaled people estimate)
- `1,<battery>`: measurement type `1` = battery level in percent (0–100)

Example (pretty‑printed):

```
863f75b0,0,12.6000,1,84.2500
```

Notes:
- Values are formatted with 4 decimal places by the firmware.
- The backend maps the numbered types using `SensorTypFrequency = 0` and `SensorTypBattery = 1` (see `customs.h`).

### Configuration

Edit `sensor-pax/src/customs.h` to configure the node:

- Identity and LoRaWAN credentials
  - `sensor_id`: string ID of the node.
  - OTAA keys: `devEui[]`, `appEui[]`, `appKey[]`.
  - ABP keys/addr also exist but OTAA is used by default (`overTheAirActivation = true`).
- Measurement and reporting
  - `factor`: linear scale from device count → people density (default `0.7`).
  - `sleepTime`: seconds between scheduled transmissions (default `900` = 15 min).
- UI and logging
  - `ENABLE_LOGGING`: `1` to print to Serial (115200), `0` to save power.
  - `ENABLE_DISPLAY`: `1` to enable the OLED display, `0` to save power.

Advanced scan settings (BLE/Wi‑Fi on/off, RSSI thresholds, scan time) are initialized in `sensor-pax/src/pax.h` via `libpax_default_config()` and can be adjusted there if needed.

Region and board are defined in `sensor-pax/platformio.ini` using build flags (default EU868, Heltec V3).

### Build and flash (PlatformIO)

Prerequisites:
- Install PlatformIO CLI or VS Code + PlatformIO extension.

Commands (from the `sensor-pax` directory):

```
# Build
pio run

# Flash (auto-detects the serial port)
pio run -t upload

# Serial monitor at 115200 baud
pio device monitor -b 115200
```

Before flashing:
- Set your LoRaWAN OTAA credentials in `customs.h` (and keep secrets out of VCS).
- Ensure the radio region in `platformio.ini` matches your deployment (EU868 by default).

### File map (key sources)

- `sensor-pax/src/main.cpp` — startup, battery measurement, loop glue.
- `sensor-pax/src/pax.h` — `libpax` configuration and counter callback.
- `sensor-pax/src/lora.h` — LoRaWAN state machine, payload builder, duty cycle.
- `sensor-pax/src/customs.h` — IDs, LoRa keys, scaling factor, sleep interval, feature flags.
- `sensor-pax/src/logging.h` — compile‑time logging toggles and helpers.
- `sensor-pax/platformio.ini` — board/region and library dependencies.

### Power & privacy notes

- Unconfirmed uplinks and the disabled OLED save airtime and power; you can disable Serial logging for further savings.
- The payload is only sent when at least 6 devices are detected to avoid exposing very small counts.
