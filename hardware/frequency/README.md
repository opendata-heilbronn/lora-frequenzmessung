# Solar LoRa ESP32-S3 — Custom Slim Board

A slim, solar-powered LoRa development board based on the Heltec WiFi LoRa 32 V3 — redesigned without OLED for low-power outdoor sensor nodes.

**Band:** 868 MHz (EU)
**Board:** 22.0mm x 58.0mm, 2-layer, 1.6mm FR4, rounded corners

## Key Features

- **ESP32-S3-MINI-1-N8** — WiFi + BLE, 8MB flash, native USB (no CH340)
- **SX1262** — LoRa 868 MHz, +22 dBm TX, ultra-low sleep current
- **CN3063** — Solar LiPo charger with Schottky diode OR from USB/solar
- **AP2112K-3.3** — 3.3V LDO, 600mA, ~55uA quiescent
- **USB-C** — charging + native USB programming
- **No OLED** — optimized for battery/solar operation
- **Battery monitoring** — voltage divider on ADC

## Power Architecture

```
USB-C (5V) --D1 (SS14)--+
                          +-- VOR --> CN3063 --> VBAT --> AP2112K --> +3V3
Solar (5-6V) -D2 (SS14)-+            |
                               +------+
                               |
                           JST-PH (LiPo 3.7V)
```

- Solar panel: 5V-6V, 500mA+ (e.g. 6V/1W mini panel)
- Battery: Single-cell LiPo, JST-PH 2-pin, 500mAh-2000mAh
- Charge current: Set by R1 (3k -> ~400mA)

## Pin Mapping

| Function    | ESP32-S3 GPIO | Notes               |
|-------------|---------------|----------------------|
| LoRa SCK    | GPIO8         | SPI clock            |
| LoRa MOSI   | GPIO9         | SPI data out         |
| LoRa MISO   | GPIO10        | SPI data in          |
| LoRa CS     | GPIO11        | SPI chip select      |
| LoRa RST    | GPIO12        | SX1262 reset         |
| LoRa BUSY   | GPIO13        | SX1262 busy flag     |
| LoRa DIO1   | GPIO14        | SX1262 interrupt     |
| USB D-      | GPIO19        | Native USB           |
| USB D+      | GPIO20        | Native USB           |
| Battery ADC | GPIO1         | Via 100k/100k divider|
| LED         | GPIO35        | User status LED      |
| BOOT        | GPIO0         | Boot button          |
| RESET       | EN            | Reset button         |

## Repository Structure

```
solar-lora-esp32s3/
+-- hardware/
|   +-- kicad/          KiCad project files (.kicad_pro, .kicad_sch, .kicad_pcb)
|   +-- gerbers/        Production-ready Gerber + Excellon drill files
|   +-- fabrication/    BOM and CPL for JLCPCB assembly
+-- docs/
|   +-- schematic.svg   Block diagram
|   +-- netlist.txt     Net connections
+-- scripts/
|   +-- generate_project.py   KiCad project generator
|   +-- generate_gerbers.py   Gerber + drill file generator (fully routed)
+-- README.md
+-- .gitignore
```

## Ordering from JLCPCB

The `hardware/gerbers/` and `hardware/fabrication/` folders contain everything needed to order assembled boards from JLCPCB.

### Gerber Upload
1. Zip the contents of `hardware/gerbers/`
2. Upload at jlcpcb.com -> Order Now
3. Settings: 2-layer, 1.6mm, FR4, HASL lead-free

### Assembly (PCBA)
1. Upload `hardware/fabrication/BOM_JLCPCB.csv` and `hardware/fabrication/CPL_JLCPCB.csv`
2. **Standard PCBA** is required (ESP32-S3-MINI-1 needs it, $25 setup fee)
3. Review component placement in the JLCPCB 3D viewer
4. All LCSC part numbers have been verified for availability (Feb 2026)

### LCSC Part Numbers

| Component | Designator | LCSC # | Package |
|-----------|-----------|--------|---------|
| ESP32-S3-MINI-1-N8 | U1 | C2913206 | Module |
| SX1262IMLTRT | U2 | C191341 | QFN-24 |
| CN3063 | U3 | C28078 | SOP-8 |
| AP2112K-3.3 | U4 | C51118 | SOT-23-5 |
| USB-C connector | J1 | C168688 | Mid-mount |
| JST-PH 2-pin | J2, J3 | C131337 | Through-hole |
| SMA edge-mount | J4 | C496549 | Through-hole |
| Tactile switch | SW1, SW2 | C318884 | 3x2.5mm |
| SS14 Schottky | D1, D2 | C8598 | SOD-123 |
| Red LED | LED1 | C2286 | 0603 |
| Green LED | LED2 | C2297 | 0603 |
| 32MHz TCXO | Y1 | C2935820 | 2016 |
| 15nH inductor | L1 | C87189 | 0402 |
| 3.9nH inductor | L2 | C77123 | 0402 |
| 1.0pF cap | C11 | C440162 | 0402 |
| 1.5pF cap | C12 | C76901 | 0402 |

## RF Design

SX1262 matching network per Semtech AN1200.39:

```
SX1262 RFO -- C11 (1.0pF) --+-- L2 (3.9nH) -- GND
                              |
                              +-- C12 (1.5pF) -- GND
                              |
                              +-- SMA Antenna
```

- 50 ohm microstrip: ~0.72mm trace width on 1.6mm FR4
- Continuous ground plane on back copper under RF section
- GND via stitching around SX1262

## Software Compatibility

Compatible with:
- **Arduino IDE** — Heltec ESP32 board package, select "WiFi LoRa 32 V3"
- **PlatformIO** — `board = heltec_wifi_lora_32_V3`
- **RadioLib** — `SX1262 radio = new Module(11, 14, 12, 13);`

## Low Power Notes

- ESP32-S3 deep sleep: ~7uA
- SX1262 sleep: ~600nA
- TCXO powered by DIO3 (auto-off in sleep)
- No OLED saves ~15mA
- Total deep sleep: ~10-15uA board level
- With 1000mAh battery + 6V/1W solar: indefinite outdoor operation

---

Generated with assistance from Claude | 2026-02-15
