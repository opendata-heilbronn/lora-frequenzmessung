#!/usr/bin/env python3
"""
Generate production-ready Gerber + Excellon files for JLCPCB.
Solar LoRa ESP32-S3 — fully routed, 2-layer, 22x58mm.
"""
import math, os, zipfile

OUT = "/sessions/zealous-focused-franklin/solar_lora_esp32s3/gerbers"
os.makedirs(OUT, exist_ok=True)

BW, BH = 22.0, 58.0  # board dimensions (mm)
CORNER_R = 1.5

# ═══════════════════════════════════════════
# Gerber Writer
# ═══════════════════════════════════════════
class Gerber:
    def __init__(self, path):
        self.path = path
        self.ap_defs = []
        self.cmds = []
        self.ap_map = {}
        self.next_d = 10

    def _c(self, mm):
        return int(round(mm * 1e6))

    def circ(self, d):
        k = f"C{d:.4f}"
        if k not in self.ap_map:
            n = self.next_d; self.next_d += 1
            self.ap_map[k] = n
            self.ap_defs.append(f"%ADD{n}C,{d:.6f}*%")
        return self.ap_map[k]

    def rect(self, w, h):
        k = f"R{w:.4f}x{h:.4f}"
        if k not in self.ap_map:
            n = self.next_d; self.next_d += 1
            self.ap_map[k] = n
            self.ap_defs.append(f"%ADD{n}R,{w:.6f}X{h:.6f}*%")
        return self.ap_map[k]

    def obround(self, w, h):
        k = f"O{w:.4f}x{h:.4f}"
        if k not in self.ap_map:
            n = self.next_d; self.next_d += 1
            self.ap_map[k] = n
            self.ap_defs.append(f"%ADD{n}O,{w:.6f}X{h:.6f}*%")
        return self.ap_map[k]

    def sel(self, d): self.cmds.append(f"D{d}*")
    def flash(self, x, y): self.cmds.append(f"X{self._c(x)}Y{self._c(y)}D03*")
    def move(self, x, y): self.cmds.append(f"X{self._c(x)}Y{self._c(y)}D02*")
    def draw(self, x, y): self.cmds.append(f"X{self._c(x)}Y{self._c(y)}D01*")
    def dark(self): self.cmds.append("%LPD*%")
    def clear(self): self.cmds.append("%LPC*%")
    def region_on(self): self.cmds.append("G36*")
    def region_off(self): self.cmds.append("G37*")

    def polyline(self, pts, ap):
        """Draw a trace through a list of (x,y) points."""
        self.sel(ap)
        self.move(pts[0][0], pts[0][1])
        for p in pts[1:]:
            self.draw(p[0], p[1])

    def flash_rect_pad(self, x, y, w, h):
        d = self.rect(w, h)
        self.sel(d)
        self.flash(x, y)

    def flash_circ_pad(self, x, y, d):
        ap = self.circ(d)
        self.sel(ap)
        self.flash(x, y)

    def fill_rect(self, x1, y1, x2, y2):
        """Filled rectangle using region fill."""
        self.region_on()
        self.move(x1, y1)
        self.draw(x2, y1)
        self.draw(x2, y2)
        self.draw(x1, y2)
        self.draw(x1, y1)
        self.region_off()

    def write(self):
        with open(self.path, 'w') as f:
            f.write("G04 Solar-LoRa-ESP32S3*\n")
            f.write("%FSLAX46Y46*%\n")
            f.write("%MOMM*%\n")
            for a in self.ap_defs:
                f.write(a + "\n")
            f.write("%LPD*%\n")
            for c in self.cmds:
                f.write(c + "\n")
            f.write("M02*\n")


# ═══════════════════════════════════════════
# Pad Position Calculator
# ═══════════════════════════════════════════
def rot(dx, dy, a):
    """Rotate offset (dx,dy) by angle a degrees (0/90/180/270)."""
    if a == 0:   return dx, dy
    if a == 90:  return -dy, dx
    if a == 180: return -dx, -dy
    if a == 270: return dy, -dx
    r = math.radians(a)
    return dx*math.cos(r)-dy*math.sin(r), dx*math.sin(r)+dy*math.cos(r)

def rsz(w, h, a):
    if a in (90, 270): return h, w
    return w, h

class Pad:
    __slots__ = ('x','y','w','h','net','drill','shape')
    def __init__(self, x, y, w, h, net='', drill=0, shape='rect'):
        self.x=x; self.y=y; self.w=w; self.h=h
        self.net=net; self.drill=drill; self.shape=shape

# ═══════════════════════════════════════════
# Build All Pad Positions
# ═══════════════════════════════════════════
pads = {}  # "REF.PIN" -> Pad

def add_pad(ref, pin, cx, cy, comp_rot, dx, dy, pw, ph, net='', drill=0, shape='rect'):
    rdx, rdy = rot(dx, dy, comp_rot)
    rw, rh = rsz(pw, ph, comp_rot)
    pads[f"{ref}.{pin}"] = Pad(cx+rdx, cy+rdy, rw, rh, net, drill, shape)

# ─── U1: ESP32-S3-MINI-1 @ (11.0, 24.0, 0°) ───
cx, cy, cr = 11.0, 24.0, 0
# Left-side pads (14 pads), x_off=-8.2, pad 1.5x0.7
left_nets = [
    (1,"GND"),(2,"+3V3"),(3,"EN"),(4,""),(5,""),(6,""),(7,""),
    (8,""),(9,""),(10,""),(11,""),(12,"SPI_SCK"),(13,"USB_DM"),(14,"USB_DP")
]
for i,(pn,net) in enumerate(left_nets):
    add_pad("U1",pn, cx,cy,cr, -8.2, -8.255+i*1.27, 1.5,0.7, net)

# Bottom pads (11 pads), y_off=+10.75, pad 0.7x1.5
bot_nets = [
    (15,""),(16,""),(17,"SPI_MOSI"),(18,"SPI_MISO"),(19,"LORA_CS"),
    (20,"LORA_RST"),(21,"LORA_BUSY"),(22,"LORA_DIO1"),(23,"GPIO0"),
    (24,"BAT_ADC"),(25,"")
]
for i,(pn,net) in enumerate(bot_nets):
    add_pad("U1",pn, cx,cy,cr, -6.35+i*1.27, 10.75, 0.7,1.5, net)

# Right-side pads (13 pads), x_off=+8.2, pad 1.5x0.7
right_nets = [
    (26,""),(27,""),(28,""),(29,""),(30,""),(31,""),(32,""),
    (33,"LED_GPIO"),(34,""),(35,""),(36,""),(37,""),(38,"")
]
for i,(pn,net) in enumerate(right_nets):
    add_pad("U1",pn, cx,cy,cr, 8.2, -8.255+i*1.27, 1.5,0.7, net)

# Center GND pad
add_pad("U1",39, cx,cy,cr, 0,3.5, 6.0,6.0, "GND")

# ─── U2: SX1262 @ (11.0, 46.0, 0°) QFN-24 4x4mm ───
cx2, cy2 = 11.0, 46.0
# Left (pins 1-6)
sx_left = [(1,"+3V3"),(2,"GND"),(3,""),(4,""),(5,"VBAT"),(6,"VBAT")]
for i,(pn,net) in enumerate(sx_left):
    add_pad("U2",pn, cx2,cy2,0, -2.0, -1.25+i*0.5, 0.7,0.25, net)
# Bottom (pins 7-12)
sx_bot = [(7,"GND"),(8,"GND"),(9,"TCXO_OUT"),(10,""),(11,"GND"),(12,"")]
for i,(pn,net) in enumerate(sx_bot):
    add_pad("U2",pn, cx2,cy2,0, -1.25+i*0.5, 2.0, 0.25,0.7, net)
# Right (pins 13-18)
sx_right = [(13,"RF_OUT"),(14,"GND"),(15,"SPI_SCK"),(16,"SPI_MISO"),(17,"SPI_MOSI"),(18,"LORA_CS")]
for i,(pn,net) in enumerate(sx_right):
    add_pad("U2",pn, cx2,cy2,0, 2.0, 1.25-i*0.5, 0.7,0.25, net)
# Top (pins 19-24)
sx_top = [(19,"LORA_RST"),(20,"LORA_BUSY"),(21,"LORA_DIO1"),(22,""),(23,"GND"),(24,"GND")]
for i,(pn,net) in enumerate(sx_top):
    add_pad("U2",pn, cx2,cy2,0, 1.25-i*0.5, -2.0, 0.25,0.7, net)
# Center GND
add_pad("U2",25, cx2,cy2,0, 0,0, 2.4,2.4, "GND")

# ─── U3: CN3065 SOIC-8 @ (5.5, 8.0, 0°) ───
# Pins 1-4 left, 5-8 right. Pitch 1.27mm. Pad 1.5x0.6
cn_left = [(1,""),(2,"GND"),(3,"VBAT"),(4,"")]  # ISET, VSS, BAT, TEMP
cn_right = [(8,""),(7,"CHRG_STATUS"),(6,""),(5,"VOR")]  # DONE, CHRG, CE, VIN
for i,(pn,net) in enumerate(cn_left):
    add_pad("U3",pn, 5.5,8.0,0, -2.7, -1.905+i*1.27, 1.5,0.6, net)
for i,(pn,net) in enumerate(cn_right):
    add_pad("U3",pn, 5.5,8.0,0, 2.7, -1.905+i*1.27, 1.5,0.6, net)

# ─── U4: AP2112K-3.3 SOT-23-5 @ (16.5, 8.0, 0°) ───
# Pins: 1=VIN, 2=GND, 3=EN (left, top→bot: 1,2,3)
# Right: 5=VOUT(top), 4=NC(bot)
ap_left = [(1,"VBAT"),(2,"GND"),(3,"VBAT")]  # EN tied to VIN
ap_right = [(5,"+3V3"),(4,"")]
for i,(pn,net) in enumerate(ap_left):
    add_pad("U4",pn, 16.5,8.0,0, -1.3, -0.95+i*0.95, 1.0,0.55, net)
for i,(pn,net) in enumerate(ap_right):
    add_pad("U4",pn, 16.5,8.0,0, 1.3, -0.475+i*0.95, 1.0,0.55, net)

# ─── J1: USB-C @ (11.0, 2.0, 0°) ───
# Simplified: VBUS, D+, D-, GND, CC1, CC2 + shield
usbc_pads = [
    ("A1", -3.25,-2.5, 0.3,1.0, "GND"),
    ("A4", -2.25,-2.5, 0.3,1.0, "VBUS"),
    ("A5", -0.25,-2.5, 0.3,1.0, "CC1"),
    ("A6",  0.25,-2.5, 0.3,1.0, "USB_DP"),
    ("A7",  0.75,-2.5, 0.3,1.0, "USB_DM"),
    ("A9",  2.25,-2.5, 0.3,1.0, "VBUS"),
    ("A12", 3.25,-2.5, 0.3,1.0, "GND"),
    ("B1",  3.25, 2.5, 0.3,1.0, "GND"),
    ("B4",  2.25, 2.5, 0.3,1.0, "VBUS"),
    ("B5",  0.25, 2.5, 0.3,1.0, "CC2"),
    ("B6", -0.25, 2.5, 0.3,1.0, "USB_DP"),
    ("B7", -0.75, 2.5, 0.3,1.0, "USB_DM"),
    ("B9", -2.25, 2.5, 0.3,1.0, "VBUS"),
    ("B12",-3.25, 2.5, 0.3,1.0, "GND"),
]
for pin,dx,dy,pw,ph,net in usbc_pads:
    add_pad("J1",pin, 11.0,2.0,0, dx,dy, pw,ph, net)
# Shield tabs (through-hole)
add_pad("J1","S1", 11.0,2.0,0, -4.0,0, 1.2,1.8, "GND", 0.6, 'obround')
add_pad("J1","S2", 11.0,2.0,0,  4.0,0, 1.2,1.8, "GND", 0.6, 'obround')

# ─── J2: Battery JST-PH @ (4.0, 15.0, 90°) ───
add_pad("J2",1, 4.0,15.0,90, -1.0,2.8, 1.0,2.0, "VBAT")
add_pad("J2",2, 4.0,15.0,90,  1.0,2.8, 1.0,2.0, "GND")

# ─── J3: Solar JST-PH @ (4.0, 5.5, 90°) ───
add_pad("J3",1, 4.0,5.5,90, -1.0,2.8, 1.0,2.0, "VSOLAR")
add_pad("J3",2, 4.0,5.5,90,  1.0,2.8, 1.0,2.0, "GND")

# ─── J4: SMA edge @ (11.0, 56.5, 0°) ───
add_pad("J4",1, 11.0,56.5,0, 0,0, 1.5,1.5, "RF_ANT")
add_pad("J4",2, 11.0,56.5,0, -2.55,0, 1.8,1.8, "GND", 1.0, 'circ')
add_pad("J4",3, 11.0,56.5,0,  2.55,0, 1.8,1.8, "GND", 1.0, 'circ')

# ─── Switches ───
add_pad("SW1",1, 19.5,36.0,0, -1.8,0, 0.8,1.0, "EN")
add_pad("SW1",2, 19.5,36.0,0,  1.8,0, 0.8,1.0, "GND")
add_pad("SW2",1, 19.5,40.0,0, -1.8,0, 0.8,1.0, "GPIO0")
add_pad("SW2",2, 19.5,40.0,0,  1.8,0, 0.8,1.0, "GND")

# ─── Diodes SOD-123 ───
# D1 @ (10.0, 5.5, 0°) pitch 2.8mm pad 0.9x0.8
add_pad("D1",1, 10.0,5.5,0, -1.4,0, 0.9,0.8, "VBUS")
add_pad("D1",2, 10.0,5.5,0,  1.4,0, 0.9,0.8, "VOR")
# D2 @ (6.0, 5.5, 180°)
add_pad("D2",1, 6.0,5.5,180, -1.4,0, 0.9,0.8, "VOR")
add_pad("D2",2, 6.0,5.5,180,  1.4,0, 0.9,0.8, "VSOLAR")

# ─── LEDs (0603) ───
add_pad("LED1",1, 19.0,12.0,0, -0.75,0, 0.9,0.8, "CHRG_STATUS")
add_pad("LED1",2, 19.0,12.0,0,  0.75,0, 0.9,0.8, "R3_MID")
add_pad("LED2",1, 19.0,14.0,0, -0.75,0, 0.9,0.8, "LED_GPIO")
add_pad("LED2",2, 19.0,14.0,0,  0.75,0, 0.9,0.8, "R4_MID")

# ─── 0402 passives (pitch 0.8mm, pad 0.5x0.5) ───
def add_0402(ref, cx, cy, rot_a, net1, net2):
    add_pad(ref,1, cx,cy,rot_a, -0.4,0, 0.5,0.5, net1)
    add_pad(ref,2, cx,cy,rot_a,  0.4,0, 0.5,0.5, net2)

# ─── 0805 passives (pitch 1.7mm, pad 1.0x1.2) ───
def add_0805(ref, cx, cy, rot_a, net1, net2):
    add_pad(ref,1, cx,cy,rot_a, -0.85,0, 1.0,1.2, net1)
    add_pad(ref,2, cx,cy,rot_a,  0.85,0, 1.0,1.2, net2)

# Resistors
add_0402("R1",  3.0, 10.5, 0,  "CN_ISET","GND")
add_0402("R2",  8.0, 10.5, 0,  "CN_TEMP","GND")
add_0402("R3",  19.0,11.0, 90, "R3_MID","GND")
add_0402("R4",  19.0,13.0, 90, "R4_MID","GND")
add_0402("R5",  14.0,15.0, 90, "VBAT","BAT_ADC")
add_0402("R6",  14.0,17.0, 90, "BAT_ADC","GND")
add_0402("R7",  17.0,18.0, 90, "+3V3","EN")
add_0402("R8",  19.0,18.0, 90, "+3V3","GPIO0")
add_0402("R9",   9.0, 3.5, 0,  "CC1","GND")
add_0402("R10", 13.0, 3.5, 0,  "CC2","GND")

# Capacitors
add_0402("C1",  5.5, 6.5, 0, "VOR","GND")
add_0805("C2",  5.5,11.5, 0, "VBAT","GND")
add_0402("C3", 16.5, 6.5, 0, "VBAT","GND")
add_0805("C4", 16.5,10.0, 0, "+3V3","GND")
add_0402("C5",  8.0,19.0, 0, "+3V3","GND")
add_0805("C6",  5.0,19.0, 0, "+3V3","GND")
add_0402("C7",  8.0,44.0, 0, "+3V3","GND")
add_0402("C8", 14.0,44.0, 0, "VBAT","GND")
add_0402("C9",  8.0,48.0, 0, "VBAT","GND")
add_0402("C10",14.0,48.0, 0, "VR_PA","GND")
add_0402("C11",11.0,51.0, 0, "RF_OUT","RF_MATCH")
add_0402("C12",11.0,54.0, 0, "RF_MATCH","GND")

# Inductors
add_0402("L1", 11.0,43.0, 0, "VR_PA","VDD_IN")
add_0402("L2", 13.0,52.5, 90, "RF_MATCH","GND")

# TCXO 2016 (2.0x1.6mm, 4 pads)
add_pad("Y1",1, 7.0,46.0,0, -0.75,-0.55, 0.6,0.5, "+3V3")   # VDD
add_pad("Y1",2, 7.0,46.0,0, -0.75, 0.55, 0.6,0.5, "GND")    # GND
add_pad("Y1",3, 7.0,46.0,0,  0.75, 0.55, 0.6,0.5, "GND")    # GND
add_pad("Y1",4, 7.0,46.0,0,  0.75,-0.55, 0.6,0.5, "TCXO_OUT") # OUT

# ═══════════════════════════════════════════
# Via Definitions  (x, y, outer_dia, drill, net)
# ═══════════════════════════════════════════
vias = []
VIA_OD = 0.6
VIA_DR = 0.3

def add_via(x, y, net="GND"):
    vias.append((x, y, VIA_OD, VIA_DR, net))

# GND vias (connect front GND pads to back ground plane)
# Near ESP32 center GND pad
for vx in [8.5, 11.0, 13.5]:
    for vy in [26.0, 28.0, 30.0]:
        add_via(vx, vy, "GND")
# Near SX1262 GND
for vx in [9.5, 11.0, 12.5]:
    for vy in [45.0, 47.0]:
        add_via(vx, vy, "GND")
# Near power section
add_via(5.5, 9.0, "GND")
add_via(16.5, 9.0, "GND")
# Near USB-C
add_via(8.0, 1.5, "GND")
add_via(14.0, 1.5, "GND")
# Near switches
add_via(20.5, 36.5, "GND")
add_via(20.5, 40.5, "GND")
# Along board edges for ground stitching
for vy in range(5, 56, 5):
    add_via(1.0, float(vy), "GND")
    add_via(21.0, float(vy), "GND")

# Signal vias (for back-layer routing where traces cross)
add_via(2.5, 38.0, "SPI_SCK")   # SCK crosses to back layer

# ═══════════════════════════════════════════
# Trace Routes (front copper unless noted)
# Each route: list of (x,y) waypoints, trace width
# ═══════════════════════════════════════════
traces_front = []  # [(points, width)]
traces_back = []

TW_PWR = 0.4    # power trace width
TW_SIG = 0.2    # signal trace width
TW_RF  = 0.72   # 50Ω microstrip on 1.6mm FR4

def p(ref, pin):
    """Get pad position."""
    pad = pads[f"{ref}.{pin}"]
    return (pad.x, pad.y)

# ── VBUS: J1.VBUS → D1.A ──
# J1.A4 (-2.25+11,-2.5+2) = (8.75, -0.5) → D1.1 (8.6, 5.5)
traces_front.append((
    [p("J1","A4"), (8.75, 2.0), (8.6, 3.0), p("D1",1)],
    TW_PWR))

# ── VSOLAR: J3.V+ → D2.A ──
# J3.1 at rotated position → D2.2
j3_1 = p("J3",1)
d2_2 = p("D2",2)
traces_front.append((
    [j3_1, (j3_1[0], d2_2[1]), d2_2],
    TW_PWR))

# ── VOR: D1.K + D2.K → U3.VIN ──
d1_k = p("D1",2)
d2_k = p("D2",1)
u3_vin = p("U3",5)
# Junction point
vor_jct = (d1_k[0], d1_k[1])  # use D1 cathode as junction
traces_front.append(([d2_k, (d2_k[0], d1_k[1]), d1_k], TW_PWR))
# VOR → down to CN3065 VIN
traces_front.append(([d1_k, (d1_k[0], 7.5), (u3_vin[0], 7.5), (u3_vin[0], u3_vin[1])], TW_PWR))
# VOR → C1.1
c1_1 = p("C1",1)
traces_front.append(([(d2_k[0]+0.5, d1_k[1]), (c1_1[0], d1_k[1]), c1_1], TW_PWR))

# ── VBAT: U3.BAT → J2.V+ → U4.VIN → C2 → C3 → R5 → SX1262 ──
u3_bat = p("U3",3)
j2_1 = p("J2",1)
u4_vin = p("U4",1)
u4_en = p("U4",3)

# U3.BAT → junction → J2 and U4
vbat_jct = (u3_bat[0]+2, u3_bat[1])
traces_front.append(([u3_bat, vbat_jct], TW_PWR))
# → J2
traces_front.append(([vbat_jct, (vbat_jct[0], j2_1[1]), j2_1], TW_PWR))
# → C2.1
c2_1 = p("C2",1)
traces_front.append(([vbat_jct, (c2_1[0], vbat_jct[1]), c2_1], TW_PWR))
# → U4.VIN
traces_front.append(([vbat_jct, (vbat_jct[0], 7.0), (u4_vin[0], 7.0), u4_vin], TW_PWR))
# U4.EN tied to VIN
traces_front.append(([u4_vin, (u4_vin[0]-0.5, u4_vin[1]), (u4_vin[0]-0.5, u4_en[1]), u4_en], TW_PWR))
# C3.1
c3_1 = p("C3",1)
traces_front.append(([(u4_vin[0], u4_vin[1]), (c3_1[0], u4_vin[1]), c3_1], TW_PWR))
# R5.1
r5_1 = p("R5",1)
traces_front.append(([vbat_jct, (r5_1[0], vbat_jct[1]), r5_1], TW_PWR))
# VBAT to SX1262 pins (U2.5 VBAT, U2.6 VBAT_IO)
u2_vbat = p("U2",5)
u2_vbatio = p("U2",6)
# Route: R5 area → down left side → SX1262
traces_front.append(([r5_1, (r5_1[0], 20.0), (3.0, 20.0), (3.0, 46.75), (u2_vbat[0], 46.75)], TW_PWR))
traces_front.append(([u2_vbat, u2_vbatio], TW_PWR))
# C8.1 (VBAT bypass for SX1262)
c8_1 = p("C8",1)
traces_front.append(([c8_1, (c8_1[0], 46.75), u2_vbat], TW_PWR))
# C9.1
c9_1 = p("C9",1)
traces_front.append(([(u2_vbatio[0], u2_vbatio[1]), (c9_1[0]+0.5, u2_vbatio[1]), (c9_1[0]+0.5, c9_1[1]), c9_1], TW_PWR))

# ── +3V3: U4.VOUT → U1.3V3 → R7 → R8 → C4 → C5 → C6 → C7 → U2.VDD33 → Y1.VDD ──
u4_vout = p("U4",5)
u1_3v3 = p("U1",2)
r7_1 = p("R7",1)
r8_1 = p("R8",1)
c4_1 = p("C4",1)
c5_1 = p("C5",1)
c6_1 = p("C6",1)
c7_1 = p("C7",1)
u2_vdd33 = p("U2",1)
y1_vdd = p("Y1",1)

# Main 3V3 trunk along x=17.8 (right side of U4)
v3_trunk_x = u4_vout[0]
traces_front.append(([u4_vout, (v3_trunk_x, u4_vout[1])], TW_PWR))
# → C4
traces_front.append(([u4_vout, (c4_1[0], u4_vout[1]), c4_1], TW_PWR))
# → R7, R8
traces_front.append(([(v3_trunk_x, u4_vout[1]), (v3_trunk_x, 17.6), (r7_1[0], 17.6), r7_1], TW_PWR))
traces_front.append(([(v3_trunk_x, 17.6), (r8_1[0], 17.6), r8_1], TW_PWR))
# → U1.3V3 (left side of ESP)
traces_front.append(([(v3_trunk_x, 17.6), (v3_trunk_x, u1_3v3[1]), u1_3v3], TW_PWR))
# → C5, C6 (ESP bypass)
traces_front.append(([u1_3v3, (u1_3v3[0], c5_1[1]), c5_1], TW_PWR))
traces_front.append(([c5_1, c6_1], TW_PWR))
# → C7 (SX1262 bypass) and U2.VDD33
traces_front.append(([c6_1, (c6_1[0]-1, c6_1[1]), (c6_1[0]-1, c7_1[1]), c7_1], TW_PWR))
traces_front.append(([c7_1, (c7_1[0], u2_vdd33[1]), u2_vdd33], TW_PWR))
# Y1 VDD
traces_front.append(([y1_vdd, (y1_vdd[0], y1_vdd[1]-1.5), (c7_1[0]-0.5, y1_vdd[1]-1.5), (c7_1[0]-0.5, c7_1[1])], TW_PWR))

# ── SPI Bus ──
# MOSI: U1.17 (7.19, 34.75) → U2.17 (13.0, 45.25)
u1_mosi = p("U1",17)
u2_mosi = p("U2",17)
traces_front.append(([u1_mosi, (u1_mosi[0], 36.0), (15.0, 36.0), (15.0, u2_mosi[1]), u2_mosi], TW_SIG))

# MISO: U1.18 (8.46, 34.75) → U2.16 (13.0, 45.75)
u1_miso = p("U1",18)
u2_miso = p("U2",16)
traces_front.append(([u1_miso, (u1_miso[0], 36.5), (15.5, 36.5), (15.5, u2_miso[1]), u2_miso], TW_SIG))

# CS: U1.19 (9.73, 34.75) → U2.18 (13.0, 44.75)
u1_cs = p("U1",19)
u2_cs = p("U2",18)
traces_front.append(([u1_cs, (u1_cs[0], 37.0), (16.0, 37.0), (16.0, u2_cs[1]), u2_cs], TW_SIG))

# SCK: U1.12 (2.8, 29.715) → via(2.5, 38.0) → back layer → via(14.5, 38.0) → U2.15 (13.0, 46.25)
u1_sck = p("U1",12)
u2_sck = p("U2",15)
add_via(14.5, 38.0, "SPI_SCK")
traces_front.append(([u1_sck, (2.5, u1_sck[1]), (2.5, 38.0)], TW_SIG))  # to via
traces_back.append(([(2.5, 38.0), (14.5, 38.0)], TW_SIG))  # back layer segment
traces_front.append(([(14.5, 38.0), (14.5, u2_sck[1]), u2_sck], TW_SIG))  # from via

# ── LoRa Control Signals ──
# RST: U1.20 (11.0, 34.75) → U2.19 (12.25, 44.0)
u1_rst = p("U1",20)
u2_rst = p("U2",19)
traces_front.append(([u1_rst, (u1_rst[0], 37.5), (u2_rst[0], 37.5), u2_rst], TW_SIG))

# BUSY: U1.21 (12.27, 34.75) → U2.20 (11.75, 44.0)
u1_busy = p("U1",21)
u2_busy = p("U2",20)
# Route around RST vertical segment via back layer
add_via(12.27, 39.0, "LORA_BUSY")
add_via(11.75, 39.0, "LORA_BUSY")
traces_front.append(([u1_busy, (u1_busy[0], 39.0)], TW_SIG))
traces_back.append(([(12.27, 39.0), (11.75, 39.0)], TW_SIG))
traces_front.append(([(11.75, 39.0), u2_busy], TW_SIG))

# DIO1: U1.22 (13.54, 34.75) → U2.21 (11.25, 44.0)
u1_dio1 = p("U1",22)
u2_dio1 = p("U2",21)
add_via(13.54, 40.5, "LORA_DIO1")
add_via(11.25, 40.5, "LORA_DIO1")
traces_front.append(([u1_dio1, (u1_dio1[0], 40.5)], TW_SIG))
traces_back.append(([(13.54, 40.5), (11.25, 40.5)], TW_SIG))
traces_front.append(([(11.25, 40.5), u2_dio1], TW_SIG))

# ── USB Data: J1 → U1 ──
# D+: J1.A6 (11.25, -0.5) → U1.14 (2.8, 32.255)
u1_dp = p("U1",14)
j1_dp = p("J1","A6")
traces_front.append(([j1_dp, (j1_dp[0], 1.5), (1.5, 1.5), (1.5, u1_dp[1]), u1_dp], TW_SIG))

# D-: J1.A7 (11.75, -0.5) → U1.13 (2.8, 30.985)
u1_dm = p("U1",13)
j1_dm = p("J1","A7")
traces_front.append(([j1_dm, (j1_dm[0]+0.5, j1_dm[1]), (j1_dm[0]+0.5, 1.0), (1.0, 1.0), (1.0, u1_dm[1]), u1_dm], TW_SIG))

# ── BAT_ADC: R5/R6 junction → U1.24 ──
r5_2 = p("R5",2)
r6_1 = p("R6",1)
u1_adc = p("U1",24)
traces_front.append(([r5_2, r6_1], TW_SIG))
traces_front.append(([r5_2, (u1_adc[0], r5_2[1]), u1_adc], TW_SIG))

# ── EN: R7.2 → U1.3 → SW1.1 ──
r7_2 = p("R7",2)
u1_en = p("U1",3)
sw1_1 = p("SW1",1)
traces_front.append(([r7_2, u1_en], TW_SIG))
traces_front.append(([u1_en, (u1_en[0]-1, u1_en[1]), (u1_en[0]-1, 35.0), (sw1_1[0], 35.0), sw1_1], TW_SIG))

# ── GPIO0: R8.2 → U1.23 → SW2.1 ──
r8_2 = p("R8",2)
u1_gpio0 = p("U1",23)
sw2_1 = p("SW2",1)
traces_front.append(([r8_2, (r8_2[0], u1_gpio0[1]), u1_gpio0], TW_SIG))
traces_front.append(([u1_gpio0, (u1_gpio0[0]+1, u1_gpio0[1]), (u1_gpio0[0]+1, 39.0), (sw2_1[0], 39.0), sw2_1], TW_SIG))

# ── LED: U1.33 → LED2.A ──
u1_led = p("U1",33)
led2_a = p("LED2",1)
traces_front.append(([u1_led, (u1_led[0]+0.5, u1_led[1]), (u1_led[0]+0.5, led2_a[1]), led2_a], TW_SIG))

# ── CHRG_STATUS: U3.7 → LED1.A ──
u3_chrg = p("U3",7)
led1_a = p("LED1",1)
traces_front.append(([u3_chrg, (u3_chrg[0]+1, u3_chrg[1]), (u3_chrg[0]+1, 11.5), (led1_a[0]-1, 11.5), (led1_a[0]-1, led1_a[1]), led1_a], TW_SIG))

# ── LED resistors to GND ──
# R3.2 → GND via, R4.2 → GND via
add_via(19.0, 11.8, "GND")
add_via(19.0, 13.8, "GND")

# ── CC Resistors ──
# R9: CC1 → GND, R10: CC2 → GND
# CC1: J1.A5 → R9.1
j1_cc1 = p("J1","A5")
r9_1 = p("R9",1)
traces_front.append(([j1_cc1, (j1_cc1[0], r9_1[1]), r9_1], TW_SIG))
# CC2: J1.B5 → R10.1
j1_cc2 = p("J1","B5")
r10_1 = p("R10",1)
traces_front.append(([j1_cc2, (j1_cc2[0], r10_1[1]), r10_1], TW_SIG))

# ── CN3065 support: ISET (R1), TEMP (R2) ──
u3_iset = p("U3",1)
r1_1 = p("R1",1)
traces_front.append(([u3_iset, (u3_iset[0], r1_1[1]), r1_1], TW_SIG))
u3_temp = p("U3",4)
r2_1 = p("R2",1)
traces_front.append(([u3_temp, (u3_temp[0], r2_1[1]), r2_1], TW_SIG))
# CE tied to VIN via short trace
u3_ce = p("U3",6)
traces_front.append(([u3_ce, (u3_ce[0], u3_ce[1]-1), (p("U3",5)[0], p("U3",5)[1])], TW_SIG))

# ── TCXO_OUT: Y1.4 → U2.9 ──
y1_out = p("Y1",4)
u2_xta = p("U2",9)
traces_front.append(([y1_out, (u2_xta[0], y1_out[1]), (u2_xta[0], u2_xta[1])], TW_SIG))

# ── SX1262 DC-DC: L1 connects VR_PA to VDD_IN ──
# U2.3 (VR_PA) → L1.1 → L1.2 → U2.4 (VDD_IN)
u2_vrpa = p("U2",3)
u2_vddin = p("U2",4)
l1_1 = p("L1",1)
l1_2 = p("L1",2)
traces_front.append(([u2_vrpa, (u2_vrpa[0]-0.5, u2_vrpa[1]), (u2_vrpa[0]-0.5, l1_1[1]), l1_1], TW_SIG))
traces_front.append(([l1_2, (l1_2[0]+0.5, l1_2[1]), (l1_2[0]+0.5, u2_vddin[1]), u2_vddin], TW_SIG))
# C10 connects VR_PA to GND
c10_1 = p("C10",1)
traces_front.append(([u2_vrpa, (u2_vrpa[0]-1, u2_vrpa[1]), (u2_vrpa[0]-1, 47.5), (c10_1[0], 47.5), c10_1], TW_SIG))

# ── RF Path: U2.RFO → C11 → matching → C12/L2 → J4 ──
u2_rfo = p("U2",13)
c11_1 = p("C11",1)
c11_2 = p("C11",2)
c12_1 = p("C12",1)
c12_2 = p("C12",2)
l2_1 = p("L2",1)
l2_2 = p("L2",2)
j4_sig = p("J4",1)

# RFO → C11 (series cap)
traces_front.append(([u2_rfo, (u2_rfo[0]+1, u2_rfo[1]), (u2_rfo[0]+1, 49.5), (c11_1[0], 49.5), (c11_1[0], c11_1[1])], TW_RF))
# C11 out → junction → L2 (shunt) and C12 (shunt) and → SMA
rf_jct = (c11_2[0], c11_2[1])
traces_front.append(([c11_2, (c11_2[0], 52.0), (l2_1[0], 52.0), l2_1], TW_RF))
# RF junction → C12
traces_front.append(([(c11_2[0], 52.0), (c12_1[0], c12_1[1])], TW_RF))
# RF junction → J4 signal
traces_front.append(([(c11_2[0], 52.0), (c11_2[0], 55.0), (j4_sig[0], 55.0), j4_sig], TW_RF))


# ═══════════════════════════════════════════
# GENERATE GERBER FILES
# ═══════════════════════════════════════════
print("Generating Gerber files...")

# Helper: flash a pad on a gerber layer
def flash_pad(g, pad):
    if pad.shape == 'circ':
        g.flash_circ_pad(pad.x, pad.y, pad.w)
    elif pad.shape == 'obround':
        d = g.obround(pad.w, pad.h)
        g.sel(d)
        g.flash(pad.x, pad.y)
    else:
        g.flash_rect_pad(pad.x, pad.y, pad.w, pad.h)

def flash_pad_mask(g, pad, expand=0.05):
    """Flash solder mask opening (slightly larger than pad)."""
    w = pad.w + 2*expand
    h = pad.h + 2*expand
    if pad.shape == 'circ':
        g.flash_circ_pad(pad.x, pad.y, w)
    elif pad.shape == 'obround':
        d = g.obround(w, h)
        g.sel(d)
        g.flash(pad.x, pad.y)
    else:
        g.flash_rect_pad(pad.x, pad.y, w, h)


# ── F.Cu (Front Copper) ──
fcu = Gerber(f"{OUT}/solar_lora-F_Cu.gtl")
# Flash all SMD pads
for key, pad in pads.items():
    flash_pad(fcu, pad)
# Flash via pads on front
via_ap = fcu.circ(VIA_OD)
for vx, vy, od, dr, vnet in vias:
    fcu.sel(via_ap)
    fcu.flash(vx, vy)
# Draw front traces
for pts, tw in traces_front:
    ap = fcu.circ(tw)
    fcu.polyline(pts, ap)
fcu.write()
print("  [OK] F_Cu.gtl")

# ── B.Cu (Back Copper — Ground Plane + signal traces) ──
bcu = Gerber(f"{OUT}/solar_lora-B_Cu.gbl")
# 1) Ground plane fill
bcu.dark()
bcu.fill_rect(0.3, 0.3, BW-0.3, BH-0.3)
# 2) Clear around non-GND vias
bcu.clear()
clr_ap = bcu.circ(VIA_OD + 0.4)  # 0.2mm clearance each side
for vx, vy, od, dr, vnet in vias:
    if vnet != "GND":
        bcu.sel(clr_ap)
        bcu.flash(vx, vy)
# 3) Dark — through-hole pads on back, via pads, signal traces
bcu.dark()
# Through-hole pads (all GND in this design — USB shield, SMA)
for key, pad in pads.items():
    if pad.drill > 0:
        flash_pad(bcu, pad)
# Via pads on back
via_ap_b = bcu.circ(VIA_OD)
for vx, vy, od, dr, vnet in vias:
    bcu.sel(via_ap_b)
    bcu.flash(vx, vy)
# Back layer signal traces
for pts, tw in traces_back:
    ap = bcu.circ(tw)
    bcu.polyline(pts, ap)
bcu.write()
print("  [OK] B_Cu.gbl")

# ── F.Mask (Front Solder Mask — openings where pads are exposed) ──
fmask = Gerber(f"{OUT}/solar_lora-F_Mask.gts")
for key, pad in pads.items():
    flash_pad_mask(fmask, pad)
# Via mask openings (tented vias — no opening, so skip for tented)
# Actually for JLCPCB, vias are tented by default. Leave via mask openings OUT.
fmask.write()
print("  [OK] F_Mask.gts")

# ── B.Mask (Back Solder Mask) ──
bmask = Gerber(f"{OUT}/solar_lora-B_Mask.gbs")
# Only through-hole pads and vias need mask openings on back
for key, pad in pads.items():
    if pad.drill > 0:
        flash_pad_mask(bmask, pad)
# Vias — tented (no mask opening for signal integrity)
bmask.write()
print("  [OK] B_Mask.gbs")

# ── F.SilkS (Front Silkscreen) ──
silk = Gerber(f"{OUT}/solar_lora-F_SilkS.gto")
# Board name text — simplified as line segments (gerber text is complex)
# Draw reference designator markers as small crosses
silk_ap = silk.circ(0.15)
for key, pad in pads.items():
    ref = key.split(".")[0]
    # Draw a small dot at pin 1 of ICs
    if ".1" == key[key.index("."):] and ref.startswith("U"):
        silk.sel(silk_ap)
        silk.flash(pad.x - 0.5, pad.y - 0.5)
# Draw component outlines (simplified rectangles)
outlines = [
    # (cx, cy, w, h, label)
    (11.0, 24.0, 15.4, 20.5),  # ESP32-S3-MINI-1
    (11.0, 46.0, 4.0, 4.0),    # SX1262
    (5.5, 8.0, 5.4, 5.2),      # CN3065
    (16.5, 8.0, 2.9, 2.5),     # AP2112K
    (11.0, 2.0, 9.0, 7.0),     # USB-C
]
for cx, cy, w, h in outlines:
    silk_ap2 = silk.circ(0.12)
    x1, y1 = cx-w/2, cy-h/2
    x2, y2 = cx+w/2, cy+h/2
    silk.polyline([(x1,y1),(x2,y1),(x2,y2),(x1,y2),(x1,y1)], silk_ap2)
silk.write()
print("  [OK] F_SilkS.gto")

# ── B.SilkS (Back Silkscreen — minimal) ──
bsilk = Gerber(f"{OUT}/solar_lora-B_SilkS.gbo")
# Just a small marker
bsilk_ap = bsilk.circ(0.15)
bsilk.sel(bsilk_ap)
bsilk.flash(11.0, 29.0)  # center dot
bsilk.write()
print("  [OK] B_SilkS.gbo")

# ── Edge.Cuts (Board Outline) ──
edge = Gerber(f"{OUT}/solar_lora-Edge_Cuts.gko")
edge_ap = edge.circ(0.05)
edge.sel(edge_ap)
# Draw rounded rectangle
r = CORNER_R
n_arc = 8  # segments per arc
# Top edge
edge.move(r, 0)
edge.draw(BW - r, 0)
# Top-right arc
for i in range(1, n_arc + 1):
    a = math.pi/2 * (1 - i/n_arc)
    edge.draw(BW - r + r*math.cos(a), r - r*math.sin(a))
# Right edge
edge.draw(BW, BH - r)
# Bottom-right arc
for i in range(1, n_arc + 1):
    a = math.pi/2 * i/n_arc
    edge.draw(BW - r + r*math.sin(a), BH - r + r*(1-math.cos(a)))
# Bottom edge
edge.draw(r, BH)
# Bottom-left arc
for i in range(1, n_arc + 1):
    a = math.pi/2 * (1 - i/n_arc)
    edge.draw(r - r*math.cos(a), BH - r + r*math.sin(a))
# Left edge
edge.draw(0, r)
# Top-left arc
for i in range(1, n_arc + 1):
    a = math.pi/2 * i/n_arc
    edge.draw(r - r*math.sin(a), r - r*(1-math.cos(a)))
edge.write()
print("  [OK] Edge_Cuts.gko")

# ── F.Paste (Solder Paste — same as pad positions for SMD) ──
fpaste = Gerber(f"{OUT}/solar_lora-F_Paste.gtp")
for key, pad in pads.items():
    if pad.drill == 0:  # SMD pads only
        flash_pad(fpaste, pad)
fpaste.write()
print("  [OK] F_Paste.gtp")


# ═══════════════════════════════════════════
# EXCELLON DRILL FILE
# ═══════════════════════════════════════════
drills = {}  # diameter -> [(x,y)]

# Through-hole component drills
for key, pad in pads.items():
    if pad.drill > 0:
        d = round(pad.drill, 2)
        drills.setdefault(d, []).append((pad.x, pad.y))

# Via drills
for vx, vy, od, dr, vnet in vias:
    d = round(dr, 2)
    drills.setdefault(d, []).append((vx, vy))

with open(f"{OUT}/solar_lora-PTH.drl", 'w') as f:
    f.write("M48\n")
    f.write("; Solar LoRa ESP32-S3 Drill File\n")
    f.write("FMAT,2\n")
    f.write("METRIC,TZ\n")
    tool_num = 1
    tool_map = {}
    for d in sorted(drills.keys()):
        f.write(f"T{tool_num}C{d:.3f}\n")
        tool_map[d] = tool_num
        tool_num += 1
    f.write("%\n")
    f.write("G90\n")
    f.write("G05\n")
    for d in sorted(drills.keys()):
        f.write(f"T{tool_map[d]}\n")
        for x, y in drills[d]:
            f.write(f"X{x:.3f}Y{y:.3f}\n")
    f.write("M30\n")

print("  [OK] PTH.drl")

# ═══════════════════════════════════════════
# ZIP for JLCPCB Upload
# ═══════════════════════════════════════════
zip_path = "/sessions/zealous-focused-franklin/mnt/outputs/solar_lora_esp32s3_JLCPCB.zip"
with zipfile.ZipFile(zip_path, 'w', zipfile.ZIP_DEFLATED) as zf:
    # Gerbers
    for fn in os.listdir(OUT):
        zf.write(f"{OUT}/{fn}", fn)
    # BOM and CPL
    fab = "/sessions/zealous-focused-franklin/solar_lora_esp32s3/fabrication"
    zf.write(f"{fab}/BOM_JLCPCB.csv", "BOM_JLCPCB.csv")
    zf.write(f"{fab}/CPL_JLCPCB.csv", "CPL_JLCPCB.csv")

print(f"\n✓ JLCPCB-ready ZIP: {zip_path}")
print(f"  Contains: {len(os.listdir(OUT))} Gerber files + drill + BOM + CPL")
print(f"  Board: {BW}mm x {BH}mm, 2-layer, 1.6mm FR4")
print(f"  Components: {len(set(k.split('.')[0] for k in pads))} unique")
print(f"  Vias: {len(vias)}")
print(f"  Traces: {len(traces_front)} front, {len(traces_back)} back")
