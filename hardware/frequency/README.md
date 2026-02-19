# Solar LoRa ESP32-S3 — Custom Slim Board

**Version:** 1.0
**Date:** 2026-02-15
**Band:** 868 MHz (EU)
**Based on:** Heltec WiFi LoRa 32 V3 (redesigned, no OLED)

---

## Overview

A slim, solar-powered LoRa development board built around the ESP32-S3-MINI-1
and Semtech SX1262. Designed for low-power outdoor sensor nodes.

**Board dimensions:** 22.0mm x 58.0mm (2-layer, 1.6mm thick, rounded corners)

## Key Features

- **ESP32-S3-MINI-1-N8** — WiFi + BLE, 8MB flash, native USB (no CH340 needed)
- **SX1262** — LoRa transceiver, 868 MHz, +22 dBm TX power, ultra-low sleep
- **CN3065** — Solar LiPo charger with Schottky diode OR from USB/solar
- **AP2112K-3.3** — 3.3V LDO, 600mA, ~55µA quiescent current
- **No OLED** — power saved for battery/solar operation
- **Battery monitoring** — voltage divider on ADC1 (GPIO1)
- **USB-C** — charging + native USB programming (no UART bridge chip)

## Pin Mapping (Software-Compatible with Heltec V3)

| Function      | ESP32-S3 GPIO | Notes                    |
|---------------|---------------|--------------------------|
| LoRa SCK      | GPIO8         | SPI clock                |
| LoRa MOSI     | GPIO9         | SPI data out             |
| LoRa MISO     | GPIO10        | SPI data in              |
| LoRa CS       | GPIO11        | SPI chip select          |
| LoRa RST      | GPIO12        | SX1262 reset             |
| LoRa BUSY     | GPIO13        | SX1262 busy flag         |
| LoRa DIO1     | GPIO14        | SX1262 interrupt         |
| USB D-        | GPIO19        | Native USB               |
| USB D+        | GPIO20        | Native USB               |
| Battery ADC   | GPIO1         | Via 100k/100k divider    |
| LED           | GPIO35        | User status LED (green)  |
| BOOT          | GPIO0         | Boot button (pull-up)    |
| RESET         | EN            | Reset button (pull-up)   |

## Power Architecture

```
USB-C (5V) ──D1 (SS14)──┐
                          ├── VOR ──► CN3065 ──► VBAT ──► AP2112K ──► +3V3
Solar (5-6V) ─D2 (SS14)─┘            │
                                ┌─────┘
                                │
                            JST-PH (LiPo 3.7V)
```

- **Solar panel recommendation:** 5V-6V, 500mA+ (e.g., 6V/1W mini panel)
- **Battery:** Single-cell LiPo, JST-PH 2-pin, 500mAh–2000mAh
- **Charge current:** Set by R1 (3kΩ → ~400mA). Adjust per CN3065 datasheet.
- **Charge indicator:** LED1 (red) on CHRG pin

## RF Design Notes (CRITICAL for 868 MHz performance)

The SX1262 RF section uses the Semtech AN1200.39 reference matching network:

```
SX1262 RFO ── C11 (1.0pF) ──┬── L2 (3.9nH) ── GND
                              │
                              ├── C12 (1.5pF) ── GND
                              │
                              └── SMA Antenna Connector
```

**Layout rules:**
1. RF trace from RFO to SMA MUST be 50Ω impedance matched
   - For standard 1.6mm FR4 (εr=4.5), use ~0.7mm trace on top layer with
     continuous ground plane beneath → microstrip ~50Ω
2. Keep RF traces as short as possible (< 15mm ideal)
3. NO vias in the RF signal path
4. Solid ground plane under entire SX1262 and RF section (back copper)
5. Multiple GND vias around SX1262 exposed pad
6. Place matching components (C11, L2, C12) directly adjacent to RFO pin
7. Keep >2mm clearance between RF trace and any other signal traces
8. SX1262 DC-DC inductor (L1, 15nH) must be close to VR_PA/VDD_IN pins

## TCXO

The 32 MHz TCXO (Y1) provides the reference clock for SX1262.
DIO3 of SX1262 can be configured in software to power the TCXO
(saves power in deep sleep). Connect TCXO VDD to SX1262 DIO3 output
if desired, or to +3V3 for always-on operation.

## How to Complete and Order

### Step 1: Open in KiCad 7+
- Open `solar_lora_esp32s3.kicad_pro`
- Review schematic (`.kicad_sch`) — check net connections
- Review PCB layout (`.kicad_pcb`) — components are placed but NOT routed

### Step 2: Route the PCB
- Use KiCad's interactive router (press 'X' to route)
- Route power traces first (+3V3, VBAT, GND) with wider widths (0.3-0.5mm)
- Route SPI bus (0.2mm traces, keep parallel, similar length)
- Route RF trace LAST — use 50Ω microstrip (see RF notes above)
- Fill ground zones (Edit → Fill All Zones, or press 'B')
- Run DRC (Inspect → Design Rules Check)

### Step 3: Generate Gerbers
- File → Fabrication Outputs → Gerbers
- Select layers: F.Cu, B.Cu, F.SilkS, B.SilkS, F.Mask, B.Mask, Edge.Cuts
- Also generate drill files (Excellon format)

### Step 4: Order from JLCPCB
- Go to jlcpcb.com → Order Now
- Upload Gerber ZIP
- Settings: 2-layer, 1.6mm, FR4, HASL lead-free
- For assembly: upload `BOM_JLCPCB.csv` and `CPL_JLCPCB.csv`
- Select "Economic PCBA" for cost savings
- Review component placement in the JLCPCB viewer

### Step 5: Verify LCSC Part Availability
Some parts (ESP32-S3-MINI-1, SX1262) may be "Extended" parts on JLCPCB
(extra $3 fee per unique extended part). Check availability before ordering.

## LCSC Part Alternatives

| Component | Primary LCSC | Alternative LCSC | Notes |
|-----------|-------------|-----------------|-------|
| ESP32-S3-MINI-1-N8 | C2913206 | C3013847 (N4 variant) | N4 = 4MB flash |
| SX1262IMLTRT | C125702 | C2843fault | Check stock |
| CN3065 | C82081 | C264872 | Same IC, different mfg |
| AP2112K-3.3 | C51118 | C166012 | Same specs |

## Low Power Tips

1. Use ESP32-S3 deep sleep (~7µA) between LoRa transmissions
2. Configure SX1262 sleep mode after TX/RX (~600nA)
3. TCXO powered by DIO3 (auto-off in sleep)
4. No OLED = ~15mA saved
5. AP2112K quiescent: ~55µA (consider ME6211 for ~40µA)
6. Total deep sleep: ~10-15µA (board level)
7. With 1000mAh battery + 6V/1W solar: indefinite outdoor operation

## Software

Compatible with:
- **Arduino IDE** — Use Heltec ESP32 board package, select "WiFi LoRa 32 V3"
- **PlatformIO** — board = `heltec_wifi_lora_32_V3`
- **ESP-IDF** — Target ESP32-S3, configure SPI pins manually
- **RadioLib** — SX1262 library, use pin definitions from table above

Example Arduino setup:
```cpp
#include <RadioLib.h>
SX1262 radio = new Module(11, 14, 12, 13); // CS, DIO1, RST, BUSY
// SPI: SCK=8, MOSI=9, MISO=10
```

## Development Setup

### Prerequisites

- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI

### Start Infrastructure

```bash
docker-compose up -d   # TimescaleDB, Mosquitto, Grafana
migrate -database "$DB_DSN" -path db/migrations up
```

### Start the Backend

```bash
cd backend
cp ../.env.local.dist .env.local   # adjust DB_DSN, TTN keys etc.
source .env.local
go run ./Backend
```

The API server starts on **port 3001**.

### Start the Frontend

```bash
cd frontend
npm install
npm run dev
```

The dev server starts on **port 5173** and proxies `/api/*` to the backend.

### Verify Tests

**Backend (Go):**

```bash
cd backend
go test ./...
```

**Frontend (Playwright — unit tests, no backend needed):**

```bash
cd frontend
npm install
npx playwright install --with-deps chromium
npx playwright test --project=unit
```

**Frontend (integration tests — requires running backend):**

```bash
npx playwright test --project=integration --workers=1
```

All 77 unit tests should pass. Integration tests create real sensors and need the backend + database running.

---
Generated by Solar LoRa PCB Generator | 2026-02-15
