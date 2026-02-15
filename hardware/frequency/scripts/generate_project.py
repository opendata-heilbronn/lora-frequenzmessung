#!/usr/bin/env python3
"""
Solar LoRa ESP32-S3 — KiCad Project Generator
Custom slim board based on Heltec LoRa V3 with solar charging
868 MHz, no OLED, CN3065 solar charger, ESP32-S3-MINI-1 + SX1262
"""

import os, json, uuid, csv, math, textwrap
from datetime import datetime

# ════════════════════════════════════════════════════════════
# Configuration
# ════════════════════════════════════════════════════════════
PROJECT_NAME = "solar_lora_esp32s3"
BASE = "/sessions/zealous-focused-franklin/solar_lora_esp32s3"
FAB  = f"{BASE}/fabrication"
DOCS = f"{BASE}/docs"

BOARD_W = 22.0   # mm width  (slim!)
BOARD_H = 58.0   # mm height
CORNER_R = 1.5   # mm corner radius
BOARD_THICKNESS = 1.6  # mm

os.makedirs(FAB, exist_ok=True)
os.makedirs(DOCS, exist_ok=True)

def uid():
    return str(uuid.uuid4())

# ════════════════════════════════════════════════════════════
# Component / BOM Database  (LCSC part numbers for JLCPCB)
# ════════════════════════════════════════════════════════════
COMPONENTS = [
    # ref, value, package, LCSC, description, x, y, rotation
    ("U1", "ESP32-S3-MINI-1-N8", "ESP32-S3-MINI-1", "C2913206",
     "WiFi+BLE MCU module 8MB flash", 11.0, 24.0, 0),
    ("U2", "SX1262IMLTRT", "QFN-24-4x4", "C125702",
     "LoRa transceiver 868MHz", 11.0, 46.0, 0),
    ("U3", "CN3065", "SOIC-8", "C82081",
     "Solar LiPo charger IC", 5.5, 8.0, 0),
    ("U4", "AP2112K-3.3TRG1", "SOT-23-5", "C51118",
     "3.3V LDO 600mA low-Iq", 16.5, 8.0, 0),
    ("J1", "USB-C-2.0", "USB_C_Mid", "C168688",
     "USB Type-C 2.0 receptacle mid-mount", 11.0, 1.5, 0),
    ("J2", "JST-PH-2P", "JST_PH_S2B", "C131337",
     "Battery connector 2-pin JST-PH", 3.0, 15.0, 90),
    ("J3", "JST-PH-2P", "JST_PH_S2B", "C131337",
     "Solar panel connector 2-pin JST-PH", 3.0, 4.0, 90),
    ("J4", "SMA-Edge", "SMA_Edge", "C496549",
     "SMA edge-mount antenna connector 868MHz", 11.0, 57.0, 0),
    ("SW1", "RESET", "SW_3x2.5", "C318884",
     "Tactile switch reset", 20.0, 36.0, 0),
    ("SW2", "BOOT", "SW_3x2.5", "C318884",
     "Tactile switch boot/GPIO0", 20.0, 40.0, 0),
    ("D1", "SS14", "SOD-123", "C8598",
     "Schottky diode USB VBUS", 8.0, 5.5, 0),
    ("D2", "SS14", "SOD-123", "C8598",
     "Schottky diode solar input", 5.5, 5.5, 180),
    ("LED1", "RED", "LED_0402", "C2286",
     "Charge status LED", 19.0, 12.0, 0),
    ("LED2", "GREEN", "LED_0402", "C2297",
     "User status LED (GPIO35)", 19.0, 14.0, 0),
    # CN3065 support
    ("R1", "3k", "R_0402", "C25890",
     "CN3065 ISET charge current 400mA", 3.0, 10.0, 0),
    ("R2", "10k", "R_0402", "C25744",
     "CN3065 TEMP pin NTC bias", 8.0, 10.0, 0),
    ("R3", "1k", "R_0402", "C11702",
     "LED1 current limit", 19.0, 10.5, 90),
    ("R4", "1k", "R_0402", "C11702",
     "LED2 current limit", 19.0, 13.0, 90),
    # Battery voltage divider
    ("R5", "100k", "R_0402", "C25741",
     "VBAT divider upper", 14.0, 15.0, 90),
    ("R6", "100k", "R_0402", "C25741",
     "VBAT divider lower", 14.0, 17.0, 90),
    # ESP32-S3 support
    ("R7", "10k", "R_0402", "C25744",
     "EN pull-up", 17.0, 18.0, 90),
    ("R8", "10k", "R_0402", "C25744",
     "GPIO0 pull-up", 19.0, 18.0, 90),
    # USB-C CC resistors
    ("R9", "5.1k", "R_0402", "C25905",
     "USB CC1 pull-down", 9.0, 3.0, 0),
    ("R10", "5.1k", "R_0402", "C25905",
     "USB CC2 pull-down", 13.0, 3.0, 0),
    # SX1262 RF matching network (868 MHz Semtech ref design)
    ("C1", "100nF", "C_0402", "C1525",
     "CN3065 VIN bypass", 5.5, 6.5, 0),
    ("C2", "10uF", "C_0805", "C15850",
     "CN3065 BAT bypass", 5.5, 11.0, 0),
    ("C3", "100nF", "C_0402", "C1525",
     "AP2112K input bypass", 16.5, 6.5, 0),
    ("C4", "10uF", "C_0805", "C15850",
     "AP2112K output bypass", 16.5, 10.0, 0),
    ("C5", "100nF", "C_0402", "C1525",
     "ESP32-S3 VDD bypass", 11.0, 19.0, 0),
    ("C6", "10uF", "C_0805", "C15850",
     "ESP32-S3 VDD bulk bypass", 8.0, 19.0, 0),
    # SX1262 decoupling
    ("C7", "100nF", "C_0402", "C1525",
     "SX1262 VDD33 bypass", 8.0, 44.0, 0),
    ("C8", "100nF", "C_0402", "C1525",
     "SX1262 VBAT bypass", 14.0, 44.0, 0),
    ("C9", "100nF", "C_0402", "C1525",
     "SX1262 VBAT_IO bypass", 8.0, 48.0, 0),
    ("C10", "47pF", "C_0402", "C1554",
     "SX1262 VR_PA bypass", 14.0, 48.0, 0),
    # RF matching (868MHz π-network from Semtech AN1200.39)
    ("C11", "1.0pF", "C_0402", "C52726",
     "RF match series cap", 11.0, 51.0, 0),
    ("C12", "1.5pF", "C_0402", "C52727",
     "RF match shunt cap", 11.0, 53.0, 0),
    ("L1", "15nH", "L_0402", "C76977",
     "SX1262 DC-DC inductor", 11.0, 43.0, 0),
    ("L2", "3.9nH", "L_0402", "C76613",
     "RF match shunt inductor", 11.0, 52.0, 90),
    # TCXO
    ("Y1", "32MHz", "TCXO_2016", "C2838573",
     "32MHz TCXO for SX1262", 7.0, 46.0, 0),
]

# ════════════════════════════════════════════════════════════
# Net Definitions
# ════════════════════════════════════════════════════════════
NETS = [
    (0, ""),
    (1, "GND"),
    (2, "+3V3"),
    (3, "VBUS"),
    (4, "VSOLAR"),
    (5, "VOR"),       # OR'd power rail into charger
    (6, "VBAT"),
    (7, "SPI_MOSI"),
    (8, "SPI_MISO"),
    (9, "SPI_SCK"),
    (10, "LORA_CS"),
    (11, "LORA_RST"),
    (12, "LORA_BUSY"),
    (13, "LORA_DIO1"),
    (14, "USB_DP"),
    (15, "USB_DM"),
    (16, "BAT_ADC"),
    (17, "LED_GPIO"),
    (18, "CHRG_STATUS"),
    (19, "RF_OUT"),
    (20, "TCXO_OUT"),
    (21, "EN"),
    (22, "GPIO0"),
]

# ════════════════════════════════════════════════════════════
# KiCad Project File (.kicad_pro)
# ════════════════════════════════════════════════════════════
def generate_kicad_pro():
    pro = {
        "board": {
            "3dviewports": [],
            "design_settings": {
                "defaults": {"board_outline_line_width": 0.05},
                "rules": {
                    "min_clearance": 0.15,
                    "min_track_width": 0.15,
                    "min_via_diameter": 0.6,
                    "min_via_drill": 0.3,
                }
            },
            "layer_presets": [],
            "layer_selections": {}
        },
        "boards": [],
        "cvpcb": {"equivalence_files": []},
        "libraries": {"pinned_footprint_libs": [], "pinned_symbol_libs": []},
        "meta": {"filename": f"{PROJECT_NAME}.kicad_pro", "version": 1},
        "net_settings": {"classes": [{"name": "Default", "clearance": 0.2, "track_width": 0.25}]},
        "pcbnew": {"last_paths": {"gencad": "", "idf": "", "netlist": "", "specctra_dsn": "", "step": "", "vrml": ""}},
        "schematic": {"legacy_lib_dir": "", "legacy_lib_list": []},
        "sheets": [[uid(), "Root"]],
        "text_variables": {}
    }
    with open(f"{BASE}/{PROJECT_NAME}.kicad_pro", "w") as f:
        json.dump(pro, f, indent=2)
    print("[OK] .kicad_pro")


# ════════════════════════════════════════════════════════════
# KiCad Schematic (.kicad_sch) — Simplified but complete
# ════════════════════════════════════════════════════════════
def generate_kicad_sch():
    """Generate a KiCad 7 schematic with embedded lib_symbols."""
    root_uuid = uid()

    # Helper: generate a basic rectangular symbol with labeled pins
    def make_lib_symbol(name, pins, width=5.08, height=None):
        """pins: list of (pin_num, pin_name, side, pin_type, y_offset)
           side: L=left, R=right, T=top, B=bottom
           pin_type: input, output, passive, power_in, bidirectional
        """
        if height is None:
            max_pins_side = max(
                len([p for p in pins if p[2] in ('L','R')]),
                1
            )
            height = max(max_pins_side * 2.54 + 2.54, 5.08)

        half_w = width / 2
        half_h = height / 2
        lines = []
        lines.append(f'    (symbol "{name}"')
        lines.append(f'      (pin_names (offset 1.016))')
        lines.append(f'      (in_bom yes) (on_board yes)')
        lines.append(f'      (property "Reference" "U" (at 0 {half_h+1.27:.2f} 0) (effects (font (size 1.27 1.27))))')
        lines.append(f'      (property "Value" "{name}" (at 0 {-half_h-1.27:.2f} 0) (effects (font (size 1.27 1.27))))')
        lines.append(f'      (property "Footprint" "" (at 0 0 0) (effects (font (size 1.27 1.27)) hide))')
        lines.append(f'      (property "Datasheet" "" (at 0 0 0) (effects (font (size 1.27 1.27)) hide))')
        lines.append(f'      (symbol "{name}_0_1"')
        lines.append(f'        (rectangle (start {-half_w:.2f} {half_h:.2f}) (end {half_w:.2f} {-half_h:.2f})')
        lines.append(f'          (stroke (width 0.254) (type default))')
        lines.append(f'          (fill (type background)))')
        lines.append(f'      )')
        lines.append(f'      (symbol "{name}_1_1"')
        for pnum, pname, side, ptype, yoff in pins:
            if side == 'L':
                px, py, angle = -half_w - 2.54, yoff, 0
            elif side == 'R':
                px, py, angle = half_w + 2.54, yoff, 180
            elif side == 'T':
                px, py, angle = yoff, half_h + 2.54, 270
            else:  # B
                px, py, angle = yoff, -half_h - 2.54, 90
            lines.append(f'        (pin {ptype} line (at {px:.2f} {py:.2f} {angle}) (length 2.54)')
            lines.append(f'          (name "{pname}" (effects (font (size 1.016 1.016))))')
            lines.append(f'          (number "{pnum}" (effects (font (size 1.016 1.016)))))')
        lines.append(f'      )')
        lines.append(f'    )')
        return "\n".join(lines)

    # ── Define library symbols ──
    lib_symbols = []

    # ESP32-S3-MINI-1 (simplified — key pins only)
    esp_pins = [
        ("1",  "GND",       "L", "power_in",     10.16),
        ("2",  "3V3",       "L", "power_in",      7.62),
        ("3",  "EN",        "L", "input",          5.08),
        ("4",  "IO4",       "L", "bidirectional",  2.54),
        ("5",  "IO5",       "L", "bidirectional",  0),
        ("6",  "IO6",       "L", "bidirectional", -2.54),
        ("7",  "IO7",       "L", "bidirectional", -5.08),
        ("8",  "IO15",      "L", "bidirectional", -7.62),
        ("9",  "IO16",      "L", "bidirectional", -10.16),
        ("10", "IO17",      "L", "bidirectional", -12.7),
        ("21", "IO8/SCK",   "R", "bidirectional",  10.16),
        ("22", "IO9/MOSI",  "R", "bidirectional",  7.62),
        ("23", "IO10/MISO", "R", "bidirectional",  5.08),
        ("24", "IO11/CS",   "R", "bidirectional",  2.54),
        ("25", "IO12",      "R", "bidirectional",  0),
        ("26", "IO13",      "R", "bidirectional", -2.54),
        ("27", "IO14",      "R", "bidirectional", -5.08),
        ("33", "IO19/D-",   "R", "bidirectional", -7.62),
        ("34", "IO20/D+",   "R", "bidirectional", -10.16),
        ("36", "IO35",      "R", "bidirectional", -12.7),
        ("15", "IO0",       "B", "bidirectional", -2.54),
        ("14", "IO1/ADC",   "B", "bidirectional",  2.54),
        ("39", "GND_PAD",   "T", "passive",        0),
    ]
    lib_symbols.append(make_lib_symbol("ESP32-S3-MINI-1", esp_pins, width=12.7, height=30.48))

    # SX1262
    sx_pins = [
        ("1",  "VDD33",   "L", "power_in",     7.62),
        ("5",  "VBAT",    "L", "power_in",     5.08),
        ("6",  "VBAT_IO", "L", "power_in",     2.54),
        ("8",  "GND",     "L", "power_in",     0),
        ("3",  "VR_PA",   "L", "passive",     -2.54),
        ("4",  "VDD_IN",  "L", "passive",     -5.08),
        ("13", "RFO",     "L", "output",      -7.62),
        ("15", "SCK",     "R", "input",        7.62),
        ("16", "MISO",    "R", "output",       5.08),
        ("17", "MOSI",    "R", "input",        2.54),
        ("18", "NSS",     "R", "input",        0),
        ("19", "NRESET",  "R", "input",       -2.54),
        ("20", "BUSY",    "R", "output",      -5.08),
        ("21", "DIO1",    "R", "output",      -7.62),
        ("9",  "XTA",     "B", "input",       -2.54),
        ("10", "XTB",     "B", "output",       2.54),
        ("25", "GND_PAD", "T", "passive",      0),
    ]
    lib_symbols.append(make_lib_symbol("SX1262", sx_pins, width=10.16, height=22.86))

    # CN3065
    cn_pins = [
        ("1", "ISET",  "L", "passive",     2.54),
        ("2", "VSS",   "L", "power_in",    0),
        ("3", "BAT",   "L", "power_out",  -2.54),
        ("5", "VIN",   "R", "power_in",    2.54),
        ("6", "CE",    "R", "input",       0),
        ("7", "CHRG",  "R", "output",     -2.54),
        ("8", "DONE",  "R", "output",     -5.08),
        ("4", "TEMP",  "B", "input",       0),
    ]
    lib_symbols.append(make_lib_symbol("CN3065", cn_pins, width=7.62, height=15.24))

    # AP2112K-3.3
    ap_pins = [
        ("1", "VIN",  "L", "power_in",   2.54),
        ("2", "GND",  "L", "power_in",   0),
        ("3", "EN",   "L", "input",      -2.54),
        ("5", "VOUT", "R", "power_out",   2.54),
        ("4", "NC",   "R", "passive",     0),
    ]
    lib_symbols.append(make_lib_symbol("AP2112K-3.3", ap_pins, width=7.62, height=10.16))

    # Passive symbols (R, C, L, LED, SW, diode, connector, TCXO)
    for name, p1name, p2name in [("R","1","2"),("C","1","2"),("L","1","2"),
                                   ("LED","A","K"),("D","A","K")]:
        pins = [
            (p1name, p1name, "L", "passive", 0),
            (p2name, p2name, "R", "passive", 0),
        ]
        lib_symbols.append(make_lib_symbol(name, pins, width=2.54, height=2.54))

    sw_pins = [("1","1","L","passive",0), ("2","2","R","passive",0)]
    lib_symbols.append(make_lib_symbol("SW_Push", sw_pins, width=3.81, height=2.54))

    conn2_pins = [("1","1","L","passive",1.27), ("2","2","L","passive",-1.27)]
    lib_symbols.append(make_lib_symbol("Conn_01x02", conn2_pins, width=3.81, height=5.08))

    tcxo_pins = [
        ("1","VDD","L","power_in",1.27),
        ("2","GND","L","power_in",-1.27),
        ("3","OUT","R","output",0),
    ]
    lib_symbols.append(make_lib_symbol("TCXO", tcxo_pins, width=5.08, height=5.08))

    usbc_pins = [
        ("A1", "GND",  "L", "power_in",    5.08),
        ("A4", "VBUS", "L", "power_in",    2.54),
        ("A5", "CC1",  "L", "passive",     0),
        ("A6", "D+",   "L", "bidirectional",-2.54),
        ("A7", "D-",   "L", "bidirectional",-5.08),
        ("B1", "GND2", "R", "passive",      5.08),
        ("B4", "VBUS2","R", "passive",      2.54),
        ("B5", "CC2",  "R", "passive",      0),
        ("B6", "D+2",  "R", "passive",     -2.54),
        ("B7", "D-2",  "R", "passive",     -5.08),
        ("S1", "SHLD", "B", "passive",      0),
    ]
    lib_symbols.append(make_lib_symbol("USB_C_Receptacle", usbc_pins, width=10.16, height=15.24))

    sma_pins = [("1","SIG","L","passive",1.27),("2","GND","L","passive",-1.27)]
    lib_symbols.append(make_lib_symbol("SMA_Conn", sma_pins, width=3.81, height=5.08))

    # Power symbols
    for sym, pin_type in [("GND","power_in"),("+3V3","power_in"),("VBAT","power_in")]:
        lines = []
        lines.append(f'    (symbol "{sym}"')
        lines.append(f'      (power) (pin_names (offset 0)) (in_bom yes) (on_board yes)')
        lines.append(f'      (property "Reference" "#PWR" (at 0 0 0) (effects (font (size 1.27 1.27)) hide))')
        lines.append(f'      (property "Value" "{sym}" (at 0 -2.54 0) (effects (font (size 1.27 1.27))))')
        lines.append(f'      (property "Footprint" "" (at 0 0 0) (effects (font (size 1.27 1.27)) hide))')
        lines.append(f'      (property "Datasheet" "" (at 0 0 0) (effects (font (size 1.27 1.27)) hide))')
        lines.append(f'      (symbol "{sym}_0_1"')
        if sym == "GND":
            lines.append(f'        (polyline (pts (xy 0 0) (xy 0 -1.27) (xy -1.27 -1.27) (xy 1.27 -1.27)) (stroke (width 0) (type default)) (fill (type none)))')
        else:
            lines.append(f'        (polyline (pts (xy 0 0) (xy 0 1.27)) (stroke (width 0) (type default)) (fill (type none)))')
        lines.append(f'      )')
        lines.append(f'      (symbol "{sym}_1_1"')
        if sym == "GND":
            lines.append(f'        (pin {pin_type} line (at 0 0 270) (length 0) (name "{sym}" (effects (font (size 1.016 1.016)))) (number "1" (effects (font (size 1.016 1.016)))))')
        else:
            lines.append(f'        (pin {pin_type} line (at 0 0 90) (length 0) (name "{sym}" (effects (font (size 1.016 1.016)))) (number "1" (effects (font (size 1.016 1.016)))))')
        lines.append(f'      )')
        lines.append(f'    )')
        lib_symbols.append("\n".join(lines))

    # ── Assemble schematic file ──
    sch = []
    sch.append(f'(kicad_sch (version 20230121) (generator "solar_lora_gen")')
    sch.append(f'  (uuid "{root_uuid}")')
    sch.append(f'  (paper "A3")')

    # Title block
    sch.append(f'  (title_block')
    sch.append(f'    (title "Solar LoRa ESP32-S3 — Custom Slim Board")')
    sch.append(f'    (date "{datetime.now().strftime("%Y-%m-%d")}")')
    sch.append(f'    (rev "1.0")')
    sch.append(f'    (comment 1 "868MHz LoRa | ESP32-S3-MINI-1 | CN3065 Solar Charger")')
    sch.append(f'    (comment 2 "Based on Heltec LoRa V3 — No OLED, low power, slim PCB")')
    sch.append(f'  )')

    # Embedded library symbols
    sch.append(f'  (lib_symbols')
    for s in lib_symbols:
        sch.append(s)
    sch.append(f'  )')

    # ── Place component instances ──
    # We'll place them in logical groups on the schematic sheet
    # Group 1: Power (left side)
    # Group 2: ESP32-S3 (center)
    # Group 3: SX1262 + RF (right side)

    instances = []

    def place_symbol(lib_id, ref, value, x_mm, y_mm, unit=1, props=None):
        """Place a symbol instance on the schematic."""
        # KiCad schematic uses mils (1 mil = 0.0254mm), but newer versions use mm
        # Actually KiCad 7 schematics use mm internally
        inst_uuid = uid()
        lines = []
        lines.append(f'  (symbol (lib_id "{lib_id}") (at {x_mm:.2f} {y_mm:.2f} 0) (unit {unit})')
        lines.append(f'    (in_bom yes) (on_board yes) (dnp no)')
        lines.append(f'    (uuid "{inst_uuid}")')
        lines.append(f'    (property "Reference" "{ref}" (at {x_mm:.2f} {y_mm-3:.2f} 0) (effects (font (size 1.27 1.27))))')
        lines.append(f'    (property "Value" "{value}" (at {x_mm:.2f} {y_mm+3:.2f} 0) (effects (font (size 1.27 1.27))))')
        if props:
            for k, v in props.items():
                lines.append(f'    (property "{k}" "{v}" (at {x_mm:.2f} {y_mm:.2f} 0) (effects (font (size 1.27 1.27)) hide))')
        lines.append(f'  )')
        instances.append("\n".join(lines))

    # ── Power Section (x=40..120, y=40..120) ──
    place_symbol("USB_C_Receptacle", "J1", "USB-C", 50, 50)
    place_symbol("D", "D1", "SS14", 80, 42)
    place_symbol("D", "D2", "SS14", 80, 58)
    place_symbol("Conn_01x02", "J3", "SOLAR", 50, 70)
    place_symbol("CN3065", "U3", "CN3065", 110, 50)
    place_symbol("Conn_01x02", "J2", "BATTERY", 110, 80)
    place_symbol("R", "R1", "3k", 100, 70, props={"Description": "ISET"})
    place_symbol("R", "R2", "10k", 120, 70, props={"Description": "TEMP"})
    place_symbol("C", "C1", "100nF", 95, 40)
    place_symbol("C", "C2", "10uF", 120, 85)
    place_symbol("LED", "LED1", "RED", 135, 48)
    place_symbol("R", "R3", "1k", 140, 48)

    # LDO
    place_symbol("AP2112K-3.3", "U4", "AP2112K-3.3", 160, 50)
    place_symbol("C", "C3", "100nF", 150, 40)
    place_symbol("C", "C4", "10uF", 175, 40)

    # Battery voltage divider
    place_symbol("R", "R5", "100k", 130, 95)
    place_symbol("R", "R6", "100k", 130, 105)

    # ── MCU Section (x=200..280, y=30..100) ──
    place_symbol("ESP32-S3-MINI-1", "U1", "ESP32-S3-MINI-1", 240, 65)
    place_symbol("R", "R7", "10k", 215, 35, props={"Description": "EN pull-up"})
    place_symbol("R", "R8", "10k", 230, 100, props={"Description": "GPIO0 pull-up"})
    place_symbol("SW_Push", "SW1", "RESET", 210, 40)
    place_symbol("SW_Push", "SW2", "BOOT", 225, 105)
    place_symbol("C", "C5", "100nF", 220, 30)
    place_symbol("C", "C6", "10uF", 260, 30)
    place_symbol("LED", "LED2", "GREEN", 275, 85)
    place_symbol("R", "R4", "1k", 280, 85)

    # USB CC resistors
    place_symbol("R", "R9", "5.1k", 60, 35, props={"Description": "CC1"})
    place_symbol("R", "R10", "5.1k", 60, 65, props={"Description": "CC2"})

    # ── LoRa Section (x=300..400, y=30..120) ──
    place_symbol("SX1262", "U2", "SX1262", 340, 65)
    place_symbol("TCXO", "Y1", "32MHz", 325, 100)
    place_symbol("C", "C7", "100nF", 320, 45)
    place_symbol("C", "C8", "100nF", 355, 45)
    place_symbol("C", "C9", "100nF", 320, 55)
    place_symbol("C", "C10", "47pF", 355, 55)
    place_symbol("L", "L1", "15nH", 325, 75)

    # RF matching
    place_symbol("C", "C11", "1.0pF", 355, 80)
    place_symbol("L", "L2", "3.9nH", 370, 85)
    place_symbol("C", "C12", "1.5pF", 370, 95)
    place_symbol("SMA_Conn", "J4", "SMA 868MHz", 390, 80)

    # ── Add net labels as text annotations ──
    labels = []
    def add_label(name, x, y, angle=0):
        labels.append(
            f'  (label "{name}" (at {x:.2f} {y:.2f} {angle}) (effects (font (size 1.27 1.27)))\n'
            f'    (uuid "{uid()}"))'
        )

    # Key net labels
    add_label("VBUS", 75, 42)
    add_label("VSOLAR", 75, 58)
    add_label("VOR", 95, 50)
    add_label("VBAT", 125, 50)
    add_label("+3V3", 175, 50)
    add_label("GND", 110, 60)
    add_label("SPI_SCK", 270, 55)
    add_label("SPI_MOSI", 270, 58)
    add_label("SPI_MISO", 270, 61)
    add_label("LORA_CS", 270, 64)
    add_label("LORA_RST", 270, 67)
    add_label("LORA_BUSY", 270, 70)
    add_label("LORA_DIO1", 270, 73)
    add_label("USB_DP", 55, 47)
    add_label("USB_DM", 55, 50)
    add_label("BAT_ADC", 135, 100)
    add_label("RF_OUT", 360, 80)
    add_label("EN", 215, 40)
    add_label("GPIO0", 225, 100)

    # ── Write all to schematic ──
    for inst in instances:
        sch.append(inst)
    for lbl in labels:
        sch.append(lbl)

    # Text notes for documentation
    notes = [
        (30, 130, "── POWER SECTION ──\\nUSB-C and Solar panel feed through Schottky diodes (D1, D2)\\ninto CN3065 LiPo charger. Battery connects to AP2112K-3.3 LDO."),
        (200, 130, "── MCU SECTION ──\\nESP32-S3-MINI-1-N8 with native USB.\\nSPI bus connects to SX1262. ADC1 monitors battery."),
        (300, 130, "── LoRa SECTION ──\\nSX1262 with 32MHz TCXO.\\n868MHz π-network matching to SMA connector."),
    ]
    for nx, ny, txt in notes:
        sch.append(f'  (text "{txt}" (at {nx} {ny} 0) (effects (font (size 1.5 1.5))))')

    sch.append(f')')

    with open(f"{BASE}/{PROJECT_NAME}.kicad_sch", "w") as f:
        f.write("\n".join(sch))
    print("[OK] .kicad_sch")


# ════════════════════════════════════════════════════════════
# KiCad PCB Layout (.kicad_pcb)
# ════════════════════════════════════════════════════════════
def generate_kicad_pcb():
    """Generate the PCB layout with embedded footprints, board outline, and copper zones."""

    pcb = []
    pcb.append(f'(kicad_pcb (version 20221018) (generator "solar_lora_gen")')
    pcb.append(f'  (general (thickness {BOARD_THICKNESS}) (legacy_teardrops no))')
    pcb.append(f'  (paper "A4")')

    # ── Layers ──
    pcb.append(f'  (layers')
    pcb.append(f'    (0 "F.Cu" signal)')
    pcb.append(f'    (31 "B.Cu" signal)')
    pcb.append(f'    (32 "B.Adhes" user "B.Adhesive")')
    pcb.append(f'    (33 "F.Adhes" user "F.Adhesive")')
    pcb.append(f'    (34 "B.Paste" user)')
    pcb.append(f'    (35 "F.Paste" user)')
    pcb.append(f'    (36 "B.SilkS" user "B.Silkscreen")')
    pcb.append(f'    (37 "F.SilkS" user "F.Silkscreen")')
    pcb.append(f'    (38 "B.Mask" user "B.Mask")')
    pcb.append(f'    (39 "F.Mask" user "F.Mask")')
    pcb.append(f'    (40 "Dwgs.User" user "User.Drawings")')
    pcb.append(f'    (41 "Cmts.User" user "User.Comments")')
    pcb.append(f'    (42 "Edge.Cuts" user)')
    pcb.append(f'    (43 "Margin" user)')
    pcb.append(f'    (44 "B.CrtYd" user "B.Courtyard")')
    pcb.append(f'    (45 "F.CrtYd" user "F.Courtyard")')
    pcb.append(f'    (46 "B.Fab" user "B.Fabrication")')
    pcb.append(f'    (47 "F.Fab" user "F.Fabrication")')
    pcb.append(f'  )')

    # ── Setup ──
    pcb.append(f'  (setup')
    pcb.append(f'    (pad_to_mask_clearance 0.05)')
    pcb.append(f'    (pcbplotparams (layerselection 0x00010fc_ffffffff) (plot_on_all_layers_selection 0x0000000_00000000))')
    pcb.append(f'  )')

    # ── Nets ──
    for nid, nname in NETS:
        pcb.append(f'  (net {nid} "{nname}")')

    # ────────────────────────────────────────────────────────
    # Footprint Generators
    # ────────────────────────────────────────────────────────
    def fp_header(ref, value, lib, x, y, rot=0, layer="F.Cu"):
        return (
            f'  (footprint "{lib}" (layer "{layer}")\n'
            f'    (tstamp "{uid()}")\n'
            f'    (at {x:.3f} {y:.3f} {rot})\n'
            f'    (property "Reference" "{ref}" (at 0 -2 0) (layer "F.SilkS") (effects (font (size 0.8 0.8) (thickness 0.15))))\n'
            f'    (property "Value" "{value}" (at 0 2 0) (layer "F.Fab") (effects (font (size 0.8 0.8) (thickness 0.15))))'
        )

    def smd_pad(num, shape, x, y, w, h, net_id=0, net_name="", layers='"F.Cu" "F.Paste" "F.Mask"', roundrect_ratio=0):
        rr = f' (roundrect_rratio {roundrect_ratio})' if shape == "roundrect" else ""
        return (
            f'    (pad "{num}" smd {shape} (at {x:.3f} {y:.3f}) (size {w:.3f} {h:.3f})\n'
            f'      (layers {layers}) (net {net_id} "{net_name}"){rr})'
        )

    def th_pad(num, shape, x, y, w, h, drill, net_id=0, net_name=""):
        return (
            f'    (pad "{num}" thru_hole {shape} (at {x:.3f} {y:.3f}) (size {w:.3f} {h:.3f}) (drill {drill:.3f})\n'
            f'      (layers "*.Cu" "*.Mask") (net {net_id} "{net_name}"))'
        )

    def fp_line(x1, y1, x2, y2, layer="F.SilkS", width=0.12):
        return f'    (fp_line (start {x1:.3f} {y1:.3f}) (end {x2:.3f} {y2:.3f}) (stroke (width {width}) (type solid)) (layer "{layer}"))'

    def fp_rect(x1, y1, x2, y2, layer="F.CrtYd", width=0.05):
        return f'    (fp_rect (start {x1:.3f} {y1:.3f}) (end {x2:.3f} {y2:.3f}) (stroke (width {width}) (type solid)) (layer "{layer}") (fill none))'

    # ────────────────────────────────────────────────────────
    # ESP32-S3-MINI-1 Footprint (U1)
    # Datasheet recommended land pattern
    # Module: 15.4 x 20.5mm
    # Origin at center of module
    # ────────────────────────────────────────────────────────
    def place_esp32s3(x, y, rot=0):
        lines = [fp_header("U1", "ESP32-S3-MINI-1-N8", "Custom:ESP32-S3-MINI-1", x, y, rot)]
        # Courtyard
        lines.append(fp_rect(-8.5, -11.0, 8.5, 11.0))
        # Silkscreen outline
        lines.append(fp_rect(-7.7, -10.25, 7.7, 10.25, "F.SilkS", 0.12))

        # Pad definitions from ESP32-S3-MINI-1 datasheet
        # Left side pads (x = -7.7mm from center), pitch 1.27mm
        left_pins = [
            (1,  "GND",    1, "GND"),
            (2,  "3V3",    2, "+3V3"),
            (3,  "EN",     21, "EN"),
            (4,  "IO4",    0, ""),
            (5,  "IO5",    0, ""),
            (6,  "IO6",    0, ""),
            (7,  "IO7",    0, ""),
            (8,  "IO15",   0, ""),
            (9,  "IO16",   0, ""),
            (10, "IO17",   0, ""),
            (11, "IO18",   0, ""),
            (12, "IO8",    9,  "SPI_SCK"),
            (13, "IO19",   15, "USB_DM"),
            (14, "IO20",   14, "USB_DP"),
        ]
        for i, (pnum, pname, nid, nname) in enumerate(left_pins):
            py = -8.255 + i * 1.27
            lines.append(smd_pad(str(pnum), "rect", -8.2, py, 1.5, 0.7, nid, nname))

        # Bottom pads (y = 10.25mm from center), pitch 1.27mm
        bottom_pins = [
            (15, "IO3",    0, ""),
            (16, "IO46",   0, ""),
            (17, "IO9",    7,  "SPI_MOSI"),
            (18, "IO10",   8,  "SPI_MISO"),
            (19, "IO11",   10, "LORA_CS"),
            (20, "IO12",   11, "LORA_RST"),
            (21, "IO13",   12, "LORA_BUSY"),
            (22, "IO14",   13, "LORA_DIO1"),
            (23, "IO0",    22, "GPIO0"),
            (24, "IO1",    16, "BAT_ADC"),
            (25, "IO2",    0, ""),
        ]
        for i, (pnum, pname, nid, nname) in enumerate(bottom_pins):
            px = -6.35 + i * 1.27
            lines.append(smd_pad(str(pnum), "rect", px, 10.75, 0.7, 1.5, nid, nname))

        # Right side pads (x = 7.7mm from center)
        right_pins = [
            (26, "IO42",   0, ""),
            (27, "IO41",   0, ""),
            (28, "IO40",   0, ""),
            (29, "IO39",   0, ""),
            (30, "IO38",   0, ""),
            (31, "IO37",   0, ""),
            (32, "IO36",   0, ""),
            (33, "IO35",   17, "LED_GPIO"),
            (34, "IO34",   0, ""),
            (35, "IO33",   0, ""),
            (36, "IO26",   0, ""),
            (37, "IO47",   0, ""),
            (38, "IO21",   0, ""),
        ]
        for i, (pnum, pname, nid, nname) in enumerate(right_pins):
            py = -8.255 + i * 1.27
            lines.append(smd_pad(str(pnum), "rect", 8.2, py, 1.5, 0.7, nid, nname))

        # Center GND pad
        lines.append(smd_pad("39", "rect", 0, 3.5, 6.0, 6.0, 1, "GND"))

        lines.append(f'  )')
        return "\n".join(lines)

    # ────────────────────────────────────────────────────────
    # SX1262 QFN-24 Footprint (U2)
    # 4x4mm, 0.5mm pitch, exposed pad
    # ────────────────────────────────────────────────────────
    def place_sx1262(x, y, rot=0):
        lines = [fp_header("U2", "SX1262", "Custom:QFN-24-4x4", x, y, rot)]
        lines.append(fp_rect(-2.5, -2.5, 2.5, 2.5))
        lines.append(fp_rect(-2.0, -2.0, 2.0, 2.0, "F.SilkS", 0.12))

        # QFN-24: 6 pads per side, 0.5mm pitch
        # Pad size: 0.3 x 0.7mm for edge pads
        # Pin 1 marker at top-left

        # Left side (pins 1-6), x=-2.1
        left_sx = [
            (1,  "VDD33",   2, "+3V3"),
            (2,  "GND",     1, "GND"),
            (3,  "VR_PA",   0, ""),
            (4,  "VDD_IN",  0, ""),
            (5,  "VBAT",    6, "VBAT"),
            (6,  "VBAT_IO", 6, "VBAT"),
        ]
        for i, (pn, _, nid, nn) in enumerate(left_sx):
            py = -1.25 + i * 0.5
            lines.append(smd_pad(str(pn), "rect", -2.0, py, 0.7, 0.25, nid, nn))

        # Bottom (pins 7-12), y=2.1
        bot_sx = [
            (7,  "GND2",    1, "GND"),
            (8,  "GND3",    1, "GND"),
            (9,  "XTA",     20, "TCXO_OUT"),
            (10, "XTB",     0, ""),
            (11, "GND4",    1, "GND"),
            (12, "DIO2",    0, ""),
        ]
        for i, (pn, _, nid, nn) in enumerate(bot_sx):
            px = -1.25 + i * 0.5
            lines.append(smd_pad(str(pn), "rect", px, 2.0, 0.25, 0.7, nid, nn))

        # Right side (pins 13-18), x=2.1
        right_sx = [
            (13, "RFO",     19, "RF_OUT"),
            (14, "GND5",    1,  "GND"),
            (15, "SCK",     9,  "SPI_SCK"),
            (16, "MISO",    8,  "SPI_MISO"),
            (17, "MOSI",    7,  "SPI_MOSI"),
            (18, "NSS",     10, "LORA_CS"),
        ]
        for i, (pn, _, nid, nn) in enumerate(right_sx):
            py = 1.25 - i * 0.5
            lines.append(smd_pad(str(pn), "rect", 2.0, py, 0.7, 0.25, nid, nn))

        # Top (pins 19-24), y=-2.1
        top_sx = [
            (19, "NRESET",  11, "LORA_RST"),
            (20, "BUSY",    12, "LORA_BUSY"),
            (21, "DIO1",    13, "LORA_DIO1"),
            (22, "DIO3",    0, ""),
            (23, "GND6",    1,  "GND"),
            (24, "GND7",    1,  "GND"),
        ]
        for i, (pn, _, nid, nn) in enumerate(top_sx):
            px = 1.25 - i * 0.5
            lines.append(smd_pad(str(pn), "rect", px, -2.0, 0.25, 0.7, nid, nn))

        # Exposed GND pad
        lines.append(smd_pad("25", "rect", 0, 0, 2.4, 2.4, 1, "GND"))

        lines.append(f'  )')
        return "\n".join(lines)

    # ────────────────────────────────────────────────────────
    # Generic SOIC-8 Footprint (CN3065 — U3)
    # ────────────────────────────────────────────────────────
    def place_soic8(ref, value, x, y, rot, pin_nets):
        """pin_nets: dict of pin_num -> (net_id, net_name)"""
        lines = [fp_header(ref, value, "Package_SO:SOIC-8", x, y, rot)]
        lines.append(fp_rect(-2.5, -2.5, 2.5, 2.5))
        # Pins: 4 on each side, pitch 1.27mm
        for i in range(4):
            pn = i + 1
            py = -1.905 + i * 1.27
            nid, nn = pin_nets.get(pn, (0, ""))
            lines.append(smd_pad(str(pn), "rect", -2.7, py, 1.5, 0.6, nid, nn))
        for i in range(4):
            pn = 8 - i
            py = -1.905 + i * 1.27
            nid, nn = pin_nets.get(pn, (0, ""))
            lines.append(smd_pad(str(pn), "rect", 2.7, py, 1.5, 0.6, nid, nn))
        lines.append(f'  )')
        return "\n".join(lines)

    # ────────────────────────────────────────────────────────
    # Generic SOT-23-5 Footprint (AP2112K — U4)
    # ────────────────────────────────────────────────────────
    def place_sot235(ref, value, x, y, rot, pin_nets):
        lines = [fp_header(ref, value, "Package_TO_SOT_SMD:SOT-23-5", x, y, rot)]
        lines.append(fp_rect(-1.6, -1.1, 1.6, 1.1))
        # Left side: pins 1,2,3 (bottom to top)
        for i, pn in enumerate([1, 2, 3]):
            py = 0.95 - i * 0.95
            nid, nn = pin_nets.get(pn, (0, ""))
            lines.append(smd_pad(str(pn), "rect", -1.3, py, 1.0, 0.55, nid, nn))
        # Right side: pins 4,5 (top to bottom)
        for i, pn in enumerate([5, 4]):
            py = -0.475 + i * 0.95
            nid, nn = pin_nets.get(pn, (0, ""))
            lines.append(smd_pad(str(pn), "rect", 1.3, py, 1.0, 0.55, nid, nn))
        lines.append(f'  )')
        return "\n".join(lines)

    # ────────────────────────────────────────────────────────
    # Generic 2-pad SMD (for R, C, L, LED, Diode)
    # ────────────────────────────────────────────────────────
    def place_2pad(ref, value, pkg_name, x, y, rot, pad_w, pad_h, pitch, net1=(0,""), net2=(0,"")):
        lines = [fp_header(ref, value, pkg_name, x, y, rot)]
        half_p = pitch / 2
        lines.append(smd_pad("1", "rect", -half_p, 0, pad_w, pad_h, net1[0], net1[1]))
        lines.append(smd_pad("2", "rect",  half_p, 0, pad_w, pad_h, net2[0], net2[1]))
        cw = pitch + pad_w + 0.3
        ch = pad_h + 0.3
        lines.append(fp_rect(-cw/2, -ch/2, cw/2, ch/2))
        lines.append(f'  )')
        return "\n".join(lines)

    # 0402 dimensions: pad 0.5x0.5mm, pitch 0.8mm
    def place_0402(ref, value, x, y, rot=0, net1=(0,""), net2=(0,"")):
        return place_2pad(ref, value, "Resistor_SMD:R_0402", x, y, rot, 0.5, 0.5, 0.8, net1, net2)

    # 0805 dimensions: pad 1.0x1.0mm, pitch 1.7mm
    def place_0805(ref, value, x, y, rot=0, net1=(0,""), net2=(0,"")):
        return place_2pad(ref, value, "Capacitor_SMD:C_0805", x, y, rot, 1.0, 1.2, 1.7, net1, net2)

    # SOD-123 diode: pad 0.9x0.8mm, pitch 2.8mm
    def place_sod123(ref, value, x, y, rot=0, net1=(0,""), net2=(0,"")):
        return place_2pad(ref, value, "Diode_SMD:D_SOD-123", x, y, rot, 0.9, 0.8, 2.8, net1, net2)

    # ────────────────────────────────────────────────────────
    # USB-C Mid-Mount Receptacle (J1)
    # ────────────────────────────────────────────────────────
    def place_usbc(x, y, rot=0):
        lines = [fp_header("J1", "USB-C", "Custom:USB_C_Mid", x, y, rot)]
        lines.append(fp_rect(-4.5, -4.5, 4.5, 4.5))

        # Simplified USB 2.0 pinout — only connect power + data + CC
        # Shield tabs
        lines.append(th_pad("S1", "oval", -4.0, 0, 1.2, 1.8, 0.6, 1, "GND"))
        lines.append(th_pad("S2", "oval",  4.0, 0, 1.2, 1.8, 0.6, 1, "GND"))

        # Pin row (0.5mm pitch for USB-C)
        # A-side pins
        lines.append(smd_pad("A1",  "rect", -3.25, -2.5, 0.3, 1.0, 1, "GND"))
        lines.append(smd_pad("A4",  "rect", -2.25, -2.5, 0.3, 1.0, 3, "VBUS"))
        lines.append(smd_pad("A5",  "rect", -0.25, -2.5, 0.3, 1.0, 0, ""))  # CC1
        lines.append(smd_pad("A6",  "rect",  0.25, -2.5, 0.3, 1.0, 14, "USB_DP"))
        lines.append(smd_pad("A7",  "rect",  0.75, -2.5, 0.3, 1.0, 15, "USB_DM"))
        lines.append(smd_pad("A9",  "rect",  2.25, -2.5, 0.3, 1.0, 3, "VBUS"))
        lines.append(smd_pad("A12", "rect",  3.25, -2.5, 0.3, 1.0, 1, "GND"))
        # B-side pins
        lines.append(smd_pad("B1",  "rect",  3.25, 2.5, 0.3, 1.0, 1, "GND"))
        lines.append(smd_pad("B4",  "rect",  2.25, 2.5, 0.3, 1.0, 3, "VBUS"))
        lines.append(smd_pad("B5",  "rect",  0.25, 2.5, 0.3, 1.0, 0, ""))  # CC2
        lines.append(smd_pad("B6",  "rect", -0.25, 2.5, 0.3, 1.0, 14, "USB_DP"))
        lines.append(smd_pad("B7",  "rect", -0.75, 2.5, 0.3, 1.0, 15, "USB_DM"))
        lines.append(smd_pad("B9",  "rect", -2.25, 2.5, 0.3, 1.0, 3, "VBUS"))
        lines.append(smd_pad("B12", "rect", -3.25, 2.5, 0.3, 1.0, 1, "GND"))

        lines.append(f'  )')
        return "\n".join(lines)

    # ────────────────────────────────────────────────────────
    # JST-PH 2-pin SMD (J2 battery, J3 solar)
    # ────────────────────────────────────────────────────────
    def place_jst_ph2(ref, value, x, y, rot=0, net1=(0,""), net2=(0,"")):
        lines = [fp_header(ref, value, "Connector_JST:JST_PH_S2B", x, y, rot)]
        lines.append(fp_rect(-2.5, -1.5, 2.5, 3.5))
        lines.append(smd_pad("1", "rect", -1.0, 2.8, 1.0, 2.0, net1[0], net1[1]))
        lines.append(smd_pad("2", "rect",  1.0, 2.8, 1.0, 2.0, net2[0], net2[1]))
        # Mechanical tabs
        lines.append(smd_pad("MP", "rect", -3.4, 0, 1.6, 1.8, 0, ""))
        lines.append(smd_pad("MP", "rect",  3.4, 0, 1.6, 1.8, 0, ""))
        lines.append(f'  )')
        return "\n".join(lines)

    # ────────────────────────────────────────────────────────
    # SMA Edge-Mount (J4)
    # ────────────────────────────────────────────────────────
    def place_sma_edge(x, y, rot=0):
        lines = [fp_header("J4", "SMA", "Custom:SMA_Edge", x, y, rot)]
        lines.append(fp_rect(-3.5, -3.0, 3.5, 3.0))
        # Center signal pin
        lines.append(smd_pad("1", "rect", 0, 0, 1.5, 1.5, 19, "RF_OUT"))
        # GND tabs
        lines.append(th_pad("2", "oval", -2.55, 0, 1.8, 1.8, 1.0, 1, "GND"))
        lines.append(th_pad("3", "oval",  2.55, 0, 1.8, 1.8, 1.0, 1, "GND"))
        lines.append(f'  )')
        return "\n".join(lines)

    # ────────────────────────────────────────────────────────
    # Tactile Switch (SW1, SW2) — 3x2.5mm
    # ────────────────────────────────────────────────────────
    def place_switch(ref, value, x, y, rot=0, net1=(0,""), net2=(0,"")):
        lines = [fp_header(ref, value, "Button_Switch_SMD:SW_SPST_3x2.5", x, y, rot)]
        lines.append(fp_rect(-2.0, -1.5, 2.0, 1.5))
        lines.append(smd_pad("1", "rect", -1.8, 0, 0.8, 1.0, net1[0], net1[1]))
        lines.append(smd_pad("2", "rect",  1.8, 0, 0.8, 1.0, net2[0], net2[1]))
        lines.append(f'  )')
        return "\n".join(lines)

    # ────────────────────────────────────────────────────────
    # Place All Components
    # ────────────────────────────────────────────────────────
    # Board origin at top-left (0,0), components placed in mm

    # U1 — ESP32-S3-MINI-1 centered at board
    pcb.append(place_esp32s3(BOARD_W/2, 24.0))

    # U2 — SX1262 below ESP, toward antenna
    pcb.append(place_sx1262(BOARD_W/2, 46.0))

    # U3 — CN3065 charger (SOIC-8)
    cn3065_nets = {
        1: (0, ""),       # ISET
        2: (1, "GND"),    # VSS
        3: (6, "VBAT"),   # BAT
        4: (0, ""),       # TEMP
        5: (5, "VOR"),    # VIN
        6: (0, ""),       # CE (tie to VIN)
        7: (18, "CHRG_STATUS"),  # CHRG
        8: (0, ""),       # DONE
    }
    pcb.append(place_soic8("U3", "CN3065", 5.5, 8.0, 0, cn3065_nets))

    # U4 — AP2112K-3.3 (SOT-23-5)
    ap_nets = {
        1: (6, "VBAT"),   # VIN
        2: (1, "GND"),    # GND
        3: (6, "VBAT"),   # EN (tie to VIN)
        4: (0, ""),       # NC
        5: (2, "+3V3"),   # VOUT
    }
    pcb.append(place_sot235("U4", "AP2112K-3.3", 16.5, 8.0, 0, ap_nets))

    # J1 — USB-C at top center
    pcb.append(place_usbc(BOARD_W/2, 2.0))

    # J2 — Battery connector (left side)
    pcb.append(place_jst_ph2("J2", "BATT", 4.0, 15.0, 90,
                              net1=(6, "VBAT"), net2=(1, "GND")))

    # J3 — Solar connector (top-left)
    pcb.append(place_jst_ph2("J3", "SOLAR", 4.0, 5.5, 90,
                              net1=(4, "VSOLAR"), net2=(1, "GND")))

    # J4 — SMA antenna at bottom
    pcb.append(place_sma_edge(BOARD_W/2, 56.0))

    # SW1, SW2 — Reset and Boot buttons (right side)
    pcb.append(place_switch("SW1", "RESET", 19.5, 36.0, 0,
                             net1=(21, "EN"), net2=(1, "GND")))
    pcb.append(place_switch("SW2", "BOOT", 19.5, 40.0, 0,
                             net1=(22, "GPIO0"), net2=(1, "GND")))

    # Schottky diodes
    pcb.append(place_sod123("D1", "SS14", 10.0, 5.5, 0,
                             net1=(3, "VBUS"), net2=(5, "VOR")))
    pcb.append(place_sod123("D2", "SS14", 6.0, 5.5, 180,
                             net1=(5, "VOR"), net2=(4, "VSOLAR")))

    # LEDs
    pcb.append(place_0402("LED1", "RED", 19.0, 12.0, 0,
                           net1=(18, "CHRG_STATUS"), net2=(0, "")))
    pcb.append(place_0402("LED2", "GREEN", 19.0, 14.0, 0,
                           net1=(17, "LED_GPIO"), net2=(0, "")))

    # Resistors
    pcb.append(place_0402("R1", "3k",    3.0, 10.5, 0, net1=(0,""), net2=(1,"GND")))  # ISET
    pcb.append(place_0402("R2", "10k",   8.0, 10.5, 0))  # TEMP
    pcb.append(place_0402("R3", "1k",    19.0, 11.0, 90, net2=(1,"GND")))  # LED1 R
    pcb.append(place_0402("R4", "1k",    19.0, 13.0, 90, net2=(1,"GND")))  # LED2 R
    pcb.append(place_0402("R5", "100k",  14.0, 15.0, 90, net1=(6,"VBAT"), net2=(16,"BAT_ADC")))
    pcb.append(place_0402("R6", "100k",  14.0, 17.0, 90, net1=(16,"BAT_ADC"), net2=(1,"GND")))
    pcb.append(place_0402("R7", "10k",   17.0, 18.0, 90, net1=(2,"+3V3"), net2=(21,"EN")))
    pcb.append(place_0402("R8", "10k",   19.0, 18.0, 90, net1=(2,"+3V3"), net2=(22,"GPIO0")))
    pcb.append(place_0402("R9", "5.1k",  9.0,  3.5, 0, net2=(1,"GND")))  # CC1
    pcb.append(place_0402("R10","5.1k",  13.0, 3.5, 0, net2=(1,"GND")))  # CC2

    # Capacitors — Power section
    pcb.append(place_0402("C1", "100nF", 5.5,  6.5, 0, net1=(5,"VOR"), net2=(1,"GND")))
    pcb.append(place_0805("C2", "10uF",  5.5, 11.5, 0, net1=(6,"VBAT"), net2=(1,"GND")))
    pcb.append(place_0402("C3", "100nF", 16.5, 6.5, 0, net1=(6,"VBAT"), net2=(1,"GND")))
    pcb.append(place_0805("C4", "10uF",  16.5,10.0, 0, net1=(2,"+3V3"), net2=(1,"GND")))

    # Capacitors — ESP32 bypass
    pcb.append(place_0402("C5", "100nF", 8.0, 19.0, 0, net1=(2,"+3V3"), net2=(1,"GND")))
    pcb.append(place_0805("C6", "10uF",  5.0, 19.0, 0, net1=(2,"+3V3"), net2=(1,"GND")))

    # Capacitors — SX1262
    pcb.append(place_0402("C7",  "100nF", 8.0,  44.0, 0, net1=(2,"+3V3"), net2=(1,"GND")))
    pcb.append(place_0402("C8",  "100nF", 14.0, 44.0, 0, net1=(6,"VBAT"), net2=(1,"GND")))
    pcb.append(place_0402("C9",  "100nF", 8.0,  48.0, 0, net1=(6,"VBAT"), net2=(1,"GND")))
    pcb.append(place_0402("C10", "47pF",  14.0, 48.0, 0))  # VR_PA
    pcb.append(place_0402("C11", "1.0pF", 11.0, 51.0, 0, net1=(19,"RF_OUT")))  # RF series
    pcb.append(place_0402("C12", "1.5pF", 11.0, 53.5, 0, net2=(1,"GND")))  # RF shunt

    # Inductors
    pcb.append(place_0402("L1", "15nH",  11.0, 43.0, 0))  # DC-DC
    pcb.append(place_0402("L2", "3.9nH", 11.0, 52.0, 90, net2=(1,"GND")))  # RF shunt

    # TCXO (2.0x1.6mm)
    pcb.append(place_2pad("Y1", "32MHz", "Custom:TCXO_2016", 7.0, 46.0, 0,
                           1.2, 0.9, 1.4,
                           net1=(2,"+3V3"), net2=(20,"TCXO_OUT")))

    # ────────────────────────────────────────────────────────
    # Board Outline (Edge.Cuts) — Rounded rectangle
    # ────────────────────────────────────────────────────────
    r = CORNER_R
    w, h = BOARD_W, BOARD_H
    # Straight edges
    pcb.append(f'  (gr_line (start {r} 0) (end {w-r} 0) (stroke (width 0.05) (type solid)) (layer "Edge.Cuts"))')
    pcb.append(f'  (gr_line (start {w} {r}) (end {w} {h-r}) (stroke (width 0.05) (type solid)) (layer "Edge.Cuts"))')
    pcb.append(f'  (gr_line (start {w-r} {h}) (end {r} {h}) (stroke (width 0.05) (type solid)) (layer "Edge.Cuts"))')
    pcb.append(f'  (gr_line (start 0 {h-r}) (end 0 {r}) (stroke (width 0.05) (type solid)) (layer "Edge.Cuts"))')
    # Corner arcs
    pcb.append(f'  (gr_arc (start {r} {r}) (mid {r*(1-0.707):.3f} {r*(1-0.707):.3f}) (end 0 {r}) (stroke (width 0.05) (type solid)) (layer "Edge.Cuts"))')
    pcb.append(f'  (gr_arc (start {w-r} {r}) (mid {w-r*(1-0.707):.3f} {r*(1-0.707):.3f}) (end {w} {r}) (stroke (width 0.05) (type solid)) (layer "Edge.Cuts"))')
    pcb.append(f'  (gr_arc (start {w-r} {h-r}) (mid {w-r*(1-0.707):.3f} {h-r*(1-0.707):.3f}) (end {w} {h-r}) (stroke (width 0.05) (type solid)) (layer "Edge.Cuts"))')
    pcb.append(f'  (gr_arc (start {r} {h-r}) (mid {r*(1-0.707):.3f} {h-r*(1-0.707):.3f}) (end 0 {h-r}) (stroke (width 0.05) (type solid)) (layer "Edge.Cuts"))')

    # ────────────────────────────────────────────────────────
    # Ground Pour Zones (Front and Back)
    # ────────────────────────────────────────────────────────
    for layer in ["F.Cu", "B.Cu"]:
        pcb.append(f'  (zone (net 1) (net_name "GND") (layer "{layer}") (tstamp "{uid()}")')
        pcb.append(f'    (hatch edge 0.5)')
        pcb.append(f'    (connect_pads (clearance 0.3))')
        pcb.append(f'    (min_thickness 0.2)')
        pcb.append(f'    (fill yes (thermal_gap 0.3) (thermal_bridge_width 0.3))')
        pcb.append(f'    (polygon (pts')
        pcb.append(f'      (xy 0.5 0.5) (xy {w-0.5} 0.5) (xy {w-0.5} {h-0.5}) (xy 0.5 {h-0.5})')
        pcb.append(f'    ))')
        pcb.append(f'  )')

    # ────────────────────────────────────────────────────────
    # Silkscreen labels
    # ────────────────────────────────────────────────────────
    pcb.append(f'  (gr_text "Solar LoRa ESP32-S3" (at {w/2} {h+2}) (layer "F.SilkS")')
    pcb.append(f'    (effects (font (size 1.2 1.2) (thickness 0.2))))')
    pcb.append(f'  (gr_text "868MHz | v1.0" (at {w/2} {h+3.5}) (layer "F.SilkS")')
    pcb.append(f'    (effects (font (size 0.8 0.8) (thickness 0.15))))')

    # Pin 1 markers
    pcb.append(f'  (gr_circle (center {BOARD_W/2 - 9.5} {24.0 - 9.5}) (end {BOARD_W/2 - 9.2} {24.0 - 9.5}) (stroke (width 0.2) (type solid)) (layer "F.SilkS") (fill none))')

    pcb.append(f')')

    with open(f"{BASE}/{PROJECT_NAME}.kicad_pcb", "w") as f:
        f.write("\n".join(pcb))
    print("[OK] .kicad_pcb")


# ════════════════════════════════════════════════════════════
# BOM for JLCPCB Assembly
# ════════════════════════════════════════════════════════════
def generate_bom():
    rows = [["Comment", "Designator", "Footprint", "LCSC Part #"]]
    for ref, value, pkg, lcsc, desc, *_ in COMPONENTS:
        rows.append([value, ref, pkg, lcsc])

    with open(f"{FAB}/BOM_JLCPCB.csv", "w", newline="") as f:
        csv.writer(f).writerows(rows)
    print("[OK] BOM_JLCPCB.csv")


# ════════════════════════════════════════════════════════════
# CPL (Component Placement List) for JLCPCB
# ════════════════════════════════════════════════════════════
def generate_cpl():
    rows = [["Designator", "Mid X", "Mid Y", "Rotation", "Layer"]]
    for ref, value, pkg, lcsc, desc, x, y, rot in COMPONENTS:
        rows.append([ref, f"{x:.4f}mm", f"{y:.4f}mm", str(rot), "top"])

    with open(f"{FAB}/CPL_JLCPCB.csv", "w", newline="") as f:
        csv.writer(f).writerows(rows)
    print("[OK] CPL_JLCPCB.csv")


# ════════════════════════════════════════════════════════════
# Schematic Diagram as SVG
# ════════════════════════════════════════════════════════════
def generate_schematic_svg():
    """Create a clean block-diagram style schematic as SVG."""
    svg_w, svg_h = 1200, 800

    blocks = []
    wires = []
    labels = []
    notes = []

    def box(x, y, w, h, title, pins_l=None, pins_r=None, color="#e8f4e8"):
        pins_l = pins_l or []
        pins_r = pins_r or []
        s = f'<rect x="{x}" y="{y}" width="{w}" height="{h}" rx="4" fill="{color}" stroke="#333" stroke-width="1.5"/>\n'
        s += f'<text x="{x+w/2}" y="{y+15}" text-anchor="middle" font-size="11" font-weight="bold" fill="#222">{title}</text>\n'
        for i, (pname, net) in enumerate(pins_l):
            py = y + 30 + i * 18
            s += f'<text x="{x+6}" y="{py}" font-size="9" fill="#444">{pname}</text>\n'
            s += f'<circle cx="{x}" cy="{py-4}" r="2.5" fill="#666"/>\n'
        for i, (pname, net) in enumerate(pins_r):
            py = y + 30 + i * 18
            s += f'<text x="{x+w-6}" y="{py}" text-anchor="end" font-size="9" fill="#444">{pname}</text>\n'
            s += f'<circle cx="{x+w}" cy="{py-4}" r="2.5" fill="#666"/>\n'
        return s

    def wire(x1, y1, x2, y2, color="#c00", width=1.5):
        return f'<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="{color}" stroke-width="{width}"/>\n'

    def label(x, y, text, size=10, color="#006", anchor="start"):
        return f'<text x="{x}" y="{y}" text-anchor="{anchor}" font-size="{size}" fill="{color}">{text}</text>\n'

    # ── Build SVG ──
    svg = f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {svg_w} {svg_h}" width="{svg_w}" height="{svg_h}">\n'
    svg += '<style>text { font-family: monospace; }</style>\n'
    svg += '<rect width="100%" height="100%" fill="#fafafa"/>\n'

    # Title
    svg += label(svg_w/2, 25, "Solar LoRa ESP32-S3 — Circuit Schematic (868 MHz)", 16, "#111", "middle")
    svg += label(svg_w/2, 42, "Rev 1.0 | CN3065 Solar Charger | ESP32-S3-MINI-1 + SX1262 | No OLED", 10, "#666", "middle")

    # ── POWER SECTION ──
    svg += label(30, 72, "── POWER ──", 12, "#060")

    # USB-C
    svg += box(30, 85, 90, 110, "J1: USB-C",
               [], [("VBUS", "VBUS"), ("D+", "USB_DP"), ("D-", "USB_DM"),
                    ("CC1", ""), ("CC2", ""), ("GND", "GND")], "#e8e8f4")

    # Solar connector
    svg += box(30, 220, 90, 55, "J3: SOLAR",
               [], [("V+", "VSOLAR"), ("GND", "GND")], "#f4f4e0")

    # Schottky diodes
    svg += label(145, 105, "D1: SS14", 9, "#600")
    svg += wire(120, 111, 160, 111, "#c00")  # VBUS → VOR
    svg += f'<polygon points="148,106 158,111 148,116" fill="#c00" stroke="#c00"/>\n'

    svg += label(145, 240, "D2: SS14", 9, "#600")
    svg += wire(120, 246, 160, 246, "#c00")
    svg += f'<polygon points="148,241 158,246 148,251" fill="#c00" stroke="#c00"/>\n'

    # VOR junction
    svg += wire(160, 111, 180, 111)
    svg += wire(160, 246, 180, 246)
    svg += wire(180, 111, 180, 246, "#c00")
    svg += f'<circle cx="180" cy="178" r="3" fill="#c00"/>\n'
    svg += label(183, 175, "VOR", 10, "#c00")

    # CN3065
    svg += box(200, 120, 120, 140, "U3: CN3065",
               [("VIN", "VOR"), ("ISET", ""), ("TEMP", ""), ("VSS", "GND")],
               [("BAT", "VBAT"), ("CE", ""), ("CHRG", ""), ("DONE", "")], "#e8f4e8")

    svg += wire(180, 178, 200, 146)  # VOR → CN3065 VIN

    # R1 (ISET)
    svg += label(170, 168, "R1: 3kΩ", 8, "#444")
    svg += wire(200, 164, 170, 164, "#888")
    svg += wire(170, 164, 170, 195, "#888")
    svg += label(160, 198, "GND", 8, "#666")

    # Battery
    svg += box(350, 140, 90, 55, "J2: BATTERY",
               [("V+", "VBAT"), ("GND", "GND")], [], "#f4e8e0")
    svg += wire(320, 150, 350, 150, "#c00")
    svg += label(325, 147, "VBAT", 10, "#c00")

    # Voltage divider
    svg += label(355, 210, "R5: 100kΩ", 8, "#444")
    svg += label(355, 228, "R6: 100kΩ", 8, "#444")
    svg += wire(375, 195, 375, 205, "#888")
    svg += wire(375, 215, 375, 225, "#888")
    svg += wire(375, 235, 375, 245, "#888")
    svg += label(380, 222, "BAT_ADC →", 8, "#060")
    svg += label(380, 248, "GND", 8, "#666")

    # AP2112K LDO
    svg += box(200, 300, 120, 90, "U4: AP2112K-3.3",
               [("VIN", "VBAT"), ("GND", "GND"), ("EN", "VBAT")],
               [("VOUT", "+3V3")], "#e8f4e8")
    svg += wire(320, 330, 340, 330, "#c00", 2)
    svg += label(342, 327, "+3V3", 12, "#060", "start")

    # Caps
    svg += label(200, 290, "C3: 100nF  C4: 10µF", 8, "#888")
    svg += label(325, 370, "bypasses on both sides", 8, "#888")

    # CC resistors
    svg += label(125, 145, "R9: 5.1kΩ (CC1→GND)", 8, "#888")
    svg += label(125, 157, "R10: 5.1kΩ (CC2→GND)", 8, "#888")

    # ── MCU SECTION ──
    svg += label(470, 72, "── MCU ──", 12, "#006")

    esp_pins_l = [
        ("3V3", "+3V3"), ("EN", "EN"), ("IO0/BOOT", "GPIO0"),
        ("IO1/ADC", "BAT_ADC"), ("IO8/SCK", "SPI_SCK"), ("IO9/MOSI", "SPI_MOSI"),
        ("IO10/MISO", "SPI_MISO"), ("IO11/CS", "LORA_CS"),
    ]
    esp_pins_r = [
        ("IO19/D-", "USB_DM"), ("IO20/D+", "USB_DP"),
        ("IO12/RST_L", "LORA_RST"), ("IO13/BUSY", "LORA_BUSY"),
        ("IO14/DIO1", "LORA_DIO1"), ("IO35/LED", "LED_GPIO"),
        ("GND", "GND"), ("IO4..IO7", "GPIO"),
    ]
    svg += box(470, 85, 200, 300, "U1: ESP32-S3-MINI-1-N8",
               esp_pins_l, esp_pins_r, "#dde8f8")

    # Pull-ups
    svg += label(430, 125, "R7: 10kΩ↑", 8, "#888")
    svg += label(430, 140, "R8: 10kΩ↑", 8, "#888")

    # Buttons
    svg += box(380, 110, 70, 35, "SW1: RESET",
               [("1", "EN")], [("2", "GND")], "#f8f0f0")
    svg += box(380, 150, 70, 35, "SW2: BOOT",
               [("1", "GPIO0")], [("2", "GND")], "#f8f0f0")

    # LED
    svg += label(690, 300, "LED2: GREEN", 8, "#080")
    svg += label(690, 312, "R4: 1kΩ → GND", 8, "#888")

    # USB data wires
    svg += wire(120, 129, 470, 291, "#36c", 1)  # D+ simplified
    svg += wire(120, 147, 470, 309, "#36c", 1)  # D- simplified
    svg += label(250, 270, "USB D+/D-", 8, "#36c")

    # ── LORA SECTION ──
    svg += label(790, 72, "── LoRa 868 MHz ──", 12, "#600")

    sx_pins_l = [
        ("VDD33", "+3V3"), ("VBAT", "VBAT"), ("VBAT_IO", "VBAT"),
        ("VR_PA", ""), ("VDD_IN", ""), ("GND", "GND"),
    ]
    sx_pins_r = [
        ("SCK", "SPI_SCK"), ("MISO", "SPI_MISO"), ("MOSI", "SPI_MOSI"),
        ("NSS", "LORA_CS"), ("NRESET", "LORA_RST"), ("BUSY", "LORA_BUSY"),
        ("DIO1", "LORA_DIO1"), ("RFO", "RF_OUT"),
    ]
    svg += box(790, 85, 180, 300, "U2: SX1262",
               sx_pins_l, sx_pins_r, "#f8e8e8")

    # SPI wires (simplified)
    svg += wire(670, 175, 790, 115, "#080", 1.2)
    svg += wire(670, 193, 790, 133, "#080", 1.2)
    svg += wire(670, 211, 790, 151, "#080", 1.2)
    svg += wire(670, 229, 790, 169, "#080", 1.2)
    svg += wire(670, 247, 790, 205, "#080", 1)
    svg += wire(670, 265, 790, 223, "#080", 1)
    svg += wire(670, 283, 790, 241, "#080", 1)
    svg += label(700, 160, "SPI Bus", 10, "#080")

    # TCXO
    svg += box(720, 410, 80, 50, "Y1: 32MHz",
               [("VDD", "+3V3")], [("OUT", "XTA")], "#f0f0e0")
    svg += wire(800, 435, 830, 435, "#888")
    svg += label(835, 438, "→ SX1262 XTA", 8, "#888")

    # RF matching
    svg += label(990, 100, "RF Match (868MHz)", 10, "#600")
    svg += box(990, 115, 130, 100, "π-Network",
               [("IN", "RFO")],
               [("OUT", "SMA")], "#ffe8e8")
    svg += label(1000, 145, "C11: 1.0pF series", 8, "#888")
    svg += label(1000, 158, "L2: 3.9nH shunt", 8, "#888")
    svg += label(1000, 171, "C12: 1.5pF shunt", 8, "#888")

    # RF wire
    svg += wire(970, 313, 990, 145, "#c00", 2)

    # SMA
    svg += box(1140, 115, 50, 60, "J4",
               [("SIG", "RF")], [], "#e0e0e0")
    svg += label(1145, 185, "SMA", 10, "#333")
    svg += wire(1120, 145, 1140, 145, "#c00", 2)

    # DC-DC inductor
    svg += label(720, 475, "L1: 15nH (VR_PA↔VDD_IN)", 8, "#888")
    svg += label(720, 490, "C10: 47pF (VR_PA bypass)", 8, "#888")

    # Decoupling note
    svg += label(790, 510, "C7: 100nF (VDD33)  C8: 100nF (VBAT)", 8, "#888")
    svg += label(790, 525, "C9: 100nF (VBAT_IO)", 8, "#888")

    # ── Net Legend ──
    svg += f'<rect x="30" y="{svg_h-120}" width="500" height="110" rx="6" fill="#f8f8ff" stroke="#ccc"/>\n'
    svg += label(40, svg_h-102, "KEY NETS:", 10, "#333")
    legend = [
        ("VBUS (5V from USB)", "#c00"), ("VSOLAR (4.5-6V panel)", "#c80"),
        ("VOR (diode-OR'd input)", "#a00"), ("VBAT (3.7-4.2V LiPo)", "#c00"),
        ("+3V3 (regulated)", "#060"), ("SPI_* (LoRa bus)", "#080"),
        ("USB_D+/D- (native USB)", "#36c"), ("RF_OUT → π-match → SMA", "#600"),
    ]
    for i, (txt, col) in enumerate(legend):
        row, col_idx = divmod(i, 2)
        lx = 50 + col_idx * 240
        ly = svg_h - 85 + row * 16
        svg += label(lx, ly, txt, 8, col)

    svg += '</svg>'

    with open(f"{DOCS}/schematic.svg", "w") as f:
        f.write(svg)
    print("[OK] schematic.svg")


# ════════════════════════════════════════════════════════════
# Design Documentation (README)
# ════════════════════════════════════════════════════════════
def generate_readme():
    readme = textwrap.dedent("""\
    # Solar LoRa ESP32-S3 — Custom Slim Board

    **Version:** 1.0
    **Date:** {date}
    **Band:** 868 MHz (EU)
    **Based on:** Heltec WiFi LoRa 32 V3 (redesigned, no OLED)

    ---

    ## Overview

    A slim, solar-powered LoRa development board built around the ESP32-S3-MINI-1
    and Semtech SX1262. Designed for low-power outdoor sensor nodes.

    **Board dimensions:** {bw}mm x {bh}mm (2-layer, 1.6mm thick, rounded corners)

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
    - Open `{proj}.kicad_pro`
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

    ---
    Generated by Solar LoRa PCB Generator | {date}
    """).format(
        date=datetime.now().strftime("%Y-%m-%d"),
        bw=BOARD_W, bh=BOARD_H,
        proj=PROJECT_NAME,
    )

    with open(f"{DOCS}/README.md", "w") as f:
        f.write(readme)
    print("[OK] README.md")


# ════════════════════════════════════════════════════════════
# Netlist Connectivity Table (for reference)
# ════════════════════════════════════════════════════════════
def generate_netlist_doc():
    """Plain-text netlist showing all connections."""
    nets = {
        "GND": [
            "J1.GND", "J2.2", "J3.2", "J4.GND",
            "U1.GND (pin1,39)", "U2.GND (pin2,7,8,11,14,23,24,25)",
            "U3.VSS (pin2)", "U4.GND (pin2)",
            "R1.2", "R3.2", "R4.2", "R6.2", "R9.2", "R10.2",
            "C1.2", "C2.2", "C3.2", "C4.2", "C5.2", "C6.2",
            "C7.2", "C8.2", "C9.2", "C12.2", "L2.2",
            "SW1.2", "SW2.2",
        ],
        "+3V3": [
            "U4.VOUT (pin5)", "U1.3V3 (pin2)",
            "U2.VDD33 (pin1)",
            "R7.1", "R8.1",
            "C4.1", "C5.1", "C6.1", "C7.1",
            "Y1.VDD",
        ],
        "VBUS": ["J1.VBUS", "D1.A (anode)"],
        "VSOLAR": ["J3.1", "D2.A (anode)"],
        "VOR": ["D1.K (cathode)", "D2.K (cathode)", "U3.VIN (pin5)", "C1.1"],
        "VBAT": [
            "U3.BAT (pin3)", "J2.1",
            "U4.VIN (pin1)", "U4.EN (pin3)",
            "R5.1", "C2.1", "C3.1",
            "U2.VBAT (pin5)", "U2.VBAT_IO (pin6)", "C8.1", "C9.1",
        ],
        "SPI_SCK":  ["U1.IO8 (pin12)", "U2.SCK (pin15)"],
        "SPI_MOSI": ["U1.IO9 (pin17)", "U2.MOSI (pin17)"],
        "SPI_MISO": ["U1.IO10 (pin18)", "U2.MISO (pin16)"],
        "LORA_CS":  ["U1.IO11 (pin19)", "U2.NSS (pin18)"],
        "LORA_RST": ["U1.IO12 (pin20)", "U2.NRESET (pin19)"],
        "LORA_BUSY":["U1.IO13 (pin21)", "U2.BUSY (pin20)"],
        "LORA_DIO1":["U1.IO14 (pin22)", "U2.DIO1 (pin21)"],
        "USB_DP":   ["J1.D+", "U1.IO20 (pin14)"],
        "USB_DM":   ["J1.D-", "U1.IO19 (pin13)"],
        "BAT_ADC":  ["R5.2", "R6.1", "U1.IO1 (pin24)"],
        "LED_GPIO": ["U1.IO35 (pin33)", "LED2.A"],
        "CHRG_STATUS": ["U3.CHRG (pin7)", "LED1.A"],
        "RF_OUT":   ["U2.RFO (pin13)", "C11.1"],
        "TCXO_OUT": ["Y1.OUT", "U2.XTA (pin9)"],
        "EN":       ["R7.2", "U1.EN (pin3)", "SW1.1"],
        "GPIO0":    ["R8.2", "U1.IO0 (pin23)", "SW2.1"],
    }

    lines = ["NETLIST — Solar LoRa ESP32-S3\n" + "="*50 + "\n"]
    for net_name, connections in nets.items():
        lines.append(f"\n{net_name}:")
        for conn in connections:
            lines.append(f"  └─ {conn}")

    with open(f"{DOCS}/netlist.txt", "w") as f:
        f.write("\n".join(lines))
    print("[OK] netlist.txt")


# ════════════════════════════════════════════════════════════
# Main
# ════════════════════════════════════════════════════════════
if __name__ == "__main__":
    print(f"Generating {PROJECT_NAME} KiCad project...")
    print(f"Board: {BOARD_W}mm x {BOARD_H}mm")
    print()
    generate_kicad_pro()
    generate_kicad_sch()
    generate_kicad_pcb()
    generate_bom()
    generate_cpl()
    generate_schematic_svg()
    generate_readme()
    generate_netlist_doc()
    print("\n✓ All files generated successfully!")
    print(f"  Project: {BASE}/")
    print(f"  Fabrication: {FAB}/")
    print(f"  Docs: {DOCS}/")
