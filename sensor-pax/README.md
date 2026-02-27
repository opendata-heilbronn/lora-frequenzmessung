# SensorPAX

ESP32-based LoRaWAN node that counts BLE devices (people) and reports PAX count + battery voltage via LoRaWAN.

---

## WiFi OTA Firmware Updates

Nodes are deployed in sealed enclosures. Firmware updates are delivered wirelessly: a LoRaWAN downlink (or the PRG button) triggers the node to connect to a known WiFi AP, download a firmware binary over HTTP, and self-update.

### How it works

```
Normal cycle:   wake → BLE scan → LoRaWAN TX → check downlink → deep sleep

OTA triggered:  wake → detect flag/button → connect WiFi → HTTP GET firmware.bin
                     → write OTA partition → reboot
                     (timeout 60 s: if WiFi/download fails → clear flag, resume normal)
```

### Triggering an OTA update

#### Option A — LoRaWAN downlink (remote, no physical access)

1. From TTN (or any LoRaWAN NS), queue a downlink on **FPort 2** with payload `0x01`.
2. On the next scheduled uplink the node receives the downlink, sets a persistent NVS flag, and goes back to sleep.
3. On the **next wake** the node enters OTA mode automatically.

#### Option B — PRG button (local, physical access)

1. Hold the **PRG button (GPIO 0)** while the node boots / wakes from deep sleep.
2. The node enters OTA mode immediately — no LoRaWAN step needed.

---

### Setting up the update server

The node connects to a fixed SSID and downloads from a fixed URL (configured in `src/customs.h`):

| Setting | Default value |
|---------|--------------|
| `OTA_WIFI_SSID` | `SensorPAX-Update` |
| `OTA_WIFI_PASS` | `changeme` |
| `OTA_FIRMWARE_URL` | `http://192.168.4.1/firmware.bin` |
| `OTA_TIMEOUT_SEC` | `60` |

**Steps:**

1. Build the new firmware: `pio run`
   Binary is at `.pio/build/heltec_wifi_lora_32_V3/firmware.bin`

2. Start a WiFi hotspot named `SensorPAX-Update` (phone or laptop).

3. Serve `firmware.bin` over HTTP at the configured URL, e.g. with Python:
   ```bash
   # from the directory containing firmware.bin
   python3 -m http.server 80
   ```
   The file must be accessible at `http://192.168.4.1/firmware.bin`
   (adjust `OTA_FIRMWARE_URL` if your hotspot uses a different gateway IP).

4. Trigger OTA on target nodes (Option A or B above).

5. The node connects, downloads (~30 s), reboots into the new firmware, and resumes normal operation.

---

### Updating multiple nodes at once

1. Host `firmware.bin` on a laptop/phone running a simple HTTP server.
2. Create the WiFi hotspot.
3. Queue downlink `0x01` on all target nodes in TTN.
4. Drive/walk near the nodes — each wakes on its next cycle, connects, updates, and reboots.
5. Nodes that don't see the AP time out after 60 s, clear the flag, and resume normal operation (no retry loop).

---

### Changing credentials

Edit `src/customs.h` and rebuild:

```cpp
const char OTA_WIFI_SSID[]    = "YourHotspotName";
const char OTA_WIFI_PASS[]    = "YourPassword";
const char OTA_FIRMWARE_URL[] = "http://192.168.x.x/firmware.bin";
const int  OTA_TIMEOUT_SEC    = 60;
```

---

## Building & Flashing

```bash
# Build
pio run

# Flash (USB)
pio run --target upload

# Serial monitor
pio device monitor
```

## Project structure

```
src/
  main.cpp      — boot logic, BLE scan, LoRaWAN TX/RX, OTA trigger check
  customs.h     — node credentials, sleep time, OTA settings
  ota.h         — WiFi OTA implementation (connect, download, apply)
  pax.h         — BLE PAX counting wrappers
  logging.h     — serial logging helpers
lib/
  paxlib/       — BLE scanning library
```
