# Research: Sensor Serial Debugger

## Web Serial API

**Decision**: Use `navigator.serial` (Web Serial API) directly — same pattern already
working in `frontend/src/components/ProvisionStep.vue`.

**Rationale**: Already proven in codebase. No new dependency. Chrome/Edge desktop only —
matches existing constraint already communicated to users via `OnyxInfoCard`.

**Alternatives considered**: WebUSB (rejected — requires vendor-specific driver knowledge),
backend serial proxy over WebSocket (rejected — unnecessary backend complexity, breaks
constitution Principle I).

---

## Firmware Serial Output Format

Logging gated by `ENABLE_LOGGING` build flag (`-D ENABLE_LOGGING=1` in platformio.ini).
All output via `logMessage(msg)` / `logMessageF(fmt, ...)` → `Serial.println(buffer)`.
Baud rate: **115200** (same as provisioning in `ProvisionStep.vue`).

### Observed log lines from `main.cpp`

| Line (regex pattern) | State |
|---|---|
| `Battery: X.XXV (XX.X%)` | INIT |
| `PAX count: N` | SCAN |
| `Payload: <sensor_id>,0,<density>,1,<battery>[,2,<version>]` | SEND |
| `sendReceive state=N downlinkLen=N` | SEND/RECEIVE |
| `TX ok` | RECEIVE |
| `TX error N` | ERROR |
| `Sleeping for Xs` | SLEEP |
| `Radio init failed` | ERROR |
| `Join failed` | ERROR |
| `OTA triggered, tag=…` | OTA (special) |
| `Downlink: …` | RECEIVE |

### Provisioning-mode lines (only when unconfigured)

`PROV_READY`, `OK`, `ERROR: …` — not part of normal debug cycle.

---

## State Machine Derived from Firmware

Firmware runs a single-pass setup() with no real loop. Deep sleep between cycles.
Inferred states from log sequence in `main.cpp`:

```
INIT → SCAN → SEND → RECEIVE → SLEEP
              ↓               ↓
            ERROR ←──────── ERROR
```

| State | Trigger keyword(s) |
|---|---|
| `INIT` | `Battery:` |
| `SCAN` | `PAX count:` |
| `SEND` | `Payload:`, `sendReceive state=` |
| `RECEIVE` | `TX ok`, `Downlink:` |
| `ERROR` | `TX error`, `Radio init failed`, `Join failed` |
| `SLEEP` | `Sleeping for` |

---

## Layout Decision: Split-pane

**Decision**: Left pane — scrolling serial log. Right pane — state diagram + current state
badge. Two-column CSS grid on desktop; stacked on mobile.

**Rationale**: User wants "live view how far it is" and "visual representation of errors"
simultaneously with raw log. Matches pattern used in debugger/IDE tools.

**Alternatives considered**: Tab-based (rejected — hides log while viewing state, poor UX
for live debugging), overlay panel (rejected — occludes log).

---

## Component Architecture

**Decision**: Single new view `SerialDebugger.vue` + router route `/debug`.
No new backend routes needed. Web Serial is purely browser-side.

**Key sub-components**:
- `SerialDebugger.vue` — root view, holds connection state, port reference
- `SerialLogPane.vue` — auto-scrolling `<pre>` log with line colouring
- `StateDiagram.vue` — SVG or CSS state diagram, highlights active state, shows error
  markers

**Rationale**: Consistent with existing view-per-route pattern. ProvisionStep.vue
already handles Web Serial plumbing — copy pattern, don't abstract (YAGNI).

---

## State Parsing Strategy

**Decision**: Simple regex scan per incoming line, no complex parser.

Rules (evaluated in order, first match wins):

```ts
const STATE_PATTERNS: Array<{ re: RegExp; state: SensorState; isError?: boolean }> = [
  { re: /^Battery:/,          state: 'INIT' },
  { re: /^PAX count:/,        state: 'SCAN' },
  { re: /^Payload:/,          state: 'SEND' },
  { re: /^sendReceive/,       state: 'SEND' },
  { re: /^TX ok/,             state: 'RECEIVE' },
  { re: /^Downlink:/,         state: 'RECEIVE' },
  { re: /^TX error/,          state: 'ERROR', isError: true },
  { re: /^Radio init failed/, state: 'ERROR', isError: true },
  { re: /^Join failed/,       state: 'ERROR', isError: true },
  { re: /^Sleeping for/,      state: 'SLEEP' },
]
```

**Rationale**: Firmware output is line-based and prefix-stable. Regex is readable and
testable without a parsing library. Exact patterns derived from `main.cpp` source.

---

## Playwright Testing Strategy

**Decision**: Mock `navigator.serial` via `page.addInitScript()` — same technique as
existing `debug-serial.spec.ts` placeholder. Inject a fake `SerialPort` object that
returns pre-scripted byte sequences matching real firmware log patterns.

**Tests needed**:
1. View renders + "Connect" button present (no serial support → warning shown)
2. Connect flow → state diagram transitions through INIT→SCAN→SEND→RECEIVE→SLEEP
3. Error line → state diagram shows ERROR, line highlighted red in log
4. Disconnect button closes port, resets state

**Rationale**: Web Serial can't be driven by Playwright without a real device.
Mocking is the established pattern in this repo (unit project).
