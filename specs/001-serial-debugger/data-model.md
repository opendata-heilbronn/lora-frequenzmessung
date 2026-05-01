# Data Model: Sensor Serial Debugger

All state is ephemeral (in-component memory only — no persistence, no API).

---

## SensorState (enum)

```ts
type SensorState = 'IDLE' | 'INIT' | 'SCAN' | 'SEND' | 'RECEIVE' | 'SLEEP' | 'ERROR'
```

| Value | Meaning |
|---|---|
| `IDLE` | No firmware cycle detected yet (initial / after connect) |
| `INIT` | Battery reading logged |
| `SCAN` | PAX BLE scan in progress / completed |
| `SEND` | Payload built, LoRa TX in progress |
| `RECEIVE` | TX acknowledged, downlink handled |
| `SLEEP` | Device going to deep sleep |
| `ERROR` | Fatal error logged (TX error, Radio init failed, Join failed) |

---

## LogLine

```ts
interface LogLine {
  id: number           // monotonic counter, used as :key in v-for
  text: string         // raw line from serial (trimmed)
  state: SensorState | null  // parsed state this line triggered, or null
  isError: boolean     // true → highlight red
  timestamp: Date
}
```

---

## ConnectionState

```ts
type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'error'
```

---

## Component Props / Emits

### SerialLogPane.vue

```ts
// props
lines: LogLine[]
autoScroll: boolean   // default true
```

### StateDiagram.vue

```ts
// props
currentState: SensorState
errorLines: LogLine[]   // passed so diagram can annotate which lines caused errors
```

---

## State Transition Rules

Valid forward transitions (firmware always runs linearly per wake cycle):

```
IDLE → INIT
INIT → SCAN
SCAN → SEND
SEND → RECEIVE | ERROR
RECEIVE → SLEEP | ERROR
SLEEP → INIT  (next wake cycle restarts)
* → ERROR     (any state can transition to ERROR on matching log line)
ERROR → INIT  (next wake cycle after error)
```

Backward transitions are valid only at `SLEEP → INIT` (new cycle begins).
