#!/usr/bin/env bash
# provision.sh — Flash NVS config to a SensorPAX device over serial.
# Usage: ./provision.sh [PORT] [BAUD]
#   PORT  defaults to auto-detected /dev/cu.SLAB_USBtoUART or /dev/cu.usbserial-*
#   BAUD  defaults to 115200
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"
BAUD="${2:-115200}"

# ── Load .env ───────────────────────────────────────────────────────────────
if [[ ! -f "$ENV_FILE" ]]; then
    echo "ERROR: .env not found at $ENV_FILE" >&2
    echo "       Copy .env.example to .env and fill in your credentials." >&2
    exit 1
fi
# shellcheck source=.env
source "$ENV_FILE"

# ── Validate required keys ──────────────────────────────────────────────────
for var in SENSOR_ID FACTOR SLEEP_SEC JOINEUI DEVEUI APPKEY; do
    if [[ -z "${!var:-}" ]]; then
        echo "ERROR: $var is not set in .env" >&2
        exit 1
    fi
done

if [[ ${#JOINEUI} -ne 16 ]]; then
    echo "ERROR: JOINEUI must be 16 hex chars (got ${#JOINEUI})" >&2; exit 1
fi
if [[ ${#DEVEUI} -ne 16 ]]; then
    echo "ERROR: DEVEUI must be 16 hex chars (got ${#DEVEUI})" >&2; exit 1
fi
if [[ ${#APPKEY} -ne 32 ]]; then
    echo "ERROR: APPKEY must be 32 hex chars (got ${#APPKEY})" >&2; exit 1
fi

# ── Auto-detect serial port ─────────────────────────────────────────────────
if [[ -n "${1:-}" ]]; then
    PORT="$1"
else
    PORT=""
    for candidate in /dev/cu.SLAB_USBtoUART /dev/cu.usbserial-* /dev/cu.wchusbserial*; do
        if [[ -e "$candidate" ]]; then
            PORT="$candidate"
            break
        fi
    done
    if [[ -z "$PORT" ]]; then
        echo "ERROR: No serial port found." >&2
        echo "       Specify one explicitly: ./provision.sh /dev/cu.XXXX" >&2
        exit 1
    fi
fi

echo "Using port: $PORT at $BAUD baud"

# ── Ensure pyserial is available (venv inside .pio/ which is already gitignored) ──
VENV_DIR="${SCRIPT_DIR}/.pio/provision-venv"
PYTHON="${VENV_DIR}/bin/python3"

if [[ ! -x "$PYTHON" ]]; then
    echo "Creating provisioning venv at $VENV_DIR..."
    python3 -m venv "$VENV_DIR"
fi

if ! "$PYTHON" -c "import serial" 2>/dev/null; then
    echo "Installing pyserial into venv..."
    "$VENV_DIR/bin/pip" install --quiet pyserial
fi

# ── Embedded Python provisioner ─────────────────────────────────────────────
"$PYTHON" - "$PORT" "$BAUD" "$SENSOR_ID" "$FACTOR" "$SLEEP_SEC" "$JOINEUI" "$DEVEUI" "$APPKEY" <<'PYEOF'
import sys, time, serial

port, baud, sensor_id, factor, sleep_sec, joineui, deveui, appkey = sys.argv[1:]

kvs = [
    ("sensor_id", sensor_id),
    ("factor",    factor),
    ("sleep_sec", sleep_sec),
    ("joineui",   joineui),
    ("deveui",    deveui),
    ("appkey",    appkey),
]

def read_line(ser, timeout=10):
    deadline = time.time() + timeout
    buf = b""
    while time.time() < deadline:
        if ser.in_waiting:
            c = ser.read(1)
            if c == b'\n':
                return buf.decode(errors='replace').strip()
            if c != b'\r':
                buf += c
        else:
            time.sleep(0.01)
    raise TimeoutError(f"Timeout waiting for response (partial: {buf!r})")

with serial.Serial(port, int(baud), timeout=1) as ser:
    # Toggle DTR to trigger ESP32 reset
    ser.dtr = False
    time.sleep(0.1)
    ser.dtr = True
    time.sleep(2)  # Wait for boot + Serial.begin()

    # Drain any noise from boot
    ser.reset_input_buffer()

    # Wait for PROV_READY
    print("Waiting for device ready...")
    deadline = time.time() + 15
    ready = False
    while time.time() < deadline:
        if ser.in_waiting:
            line = read_line(ser)
            print(f"< {line}")
            if "PROV_READY" in line:
                ready = True
                break
        time.sleep(0.05)

    if not ready:
        print("ERROR: Device did not send PROV_READY within 15s", file=sys.stderr)
        sys.exit(1)

    # Send KEY=VALUE pairs
    for key, val in kvs:
        cmd = f"{key}={val}\n"
        print(f"> {cmd.strip()}")
        ser.write(cmd.encode())
        ser.flush()
        resp = read_line(ser)
        print(f"< {resp}")
        if not resp.startswith("OK"):
            print(f"ERROR: unexpected response for {key}: {resp}", file=sys.stderr)
            sys.exit(1)

    # Commit
    print("> COMMIT")
    ser.write(b"COMMIT\n")
    ser.flush()
    resp = read_line(ser)
    print(f"< {resp}")
    if not resp.startswith("OK"):
        print(f"ERROR: COMMIT failed: {resp}", file=sys.stderr)
        sys.exit(1)

    print("Provisioning complete — device is rebooting...")
    time.sleep(3)

    # Show first boot messages
    deadline = time.time() + 10
    while time.time() < deadline:
        if ser.in_waiting:
            line = read_line(ser)
            print(f"[boot] {line}")
        else:
            time.sleep(0.05)

print("Done.")
PYEOF
