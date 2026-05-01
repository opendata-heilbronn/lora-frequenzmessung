# UI Component Contracts: Serial Debugger

## Route

```
/debug   →   SerialDebugger.vue   (auth required, added to router.ts)
```

Nav entry: `<OnyxNavItem label="Debug" link="/debug" />` in `App.vue`.

---

## SerialDebugger.vue (root view)

Owns: port reference, connection state, log lines array, current sensor state.

### Layout

```
┌─────────────────────────────────────────────────┐
│  [Connect / Disconnect btn]  [baud: 115200]      │  ← toolbar
├──────────────────────────┬──────────────────────-┤
│  SerialLogPane           │  StateDiagram         │
│  (scrolling raw log)     │  (state machine viz)  │
│                          │                       │
│  line 1                  │  [INIT]               │
│  line 2  ← red if error  │  [SCAN]  ← active     │
│  ...                     │  [SEND]               │
│                          │  [RECEIVE]            │
│                          │  [SLEEP]              │
│                          │  [ERROR] ← red badge  │
└──────────────────────────┴───────────────────────┘
```

Responsive: on mobile (`< 768px`) panes stack vertically (log top, diagram bottom).

### Browser compat guard

If `!('serial' in navigator)`:
- Render `<OnyxInfoCard color="warning">` only — identical copy to `ProvisionStep.vue`
- No log pane, no diagram

---

## SerialLogPane.vue

- `<pre>` block, monospace, max-height with overflow-y scroll
- New lines appended. Auto-scroll to bottom unless user has scrolled up.
- Error lines: `color: var(--onyx-color-danger-500)` + bold
- State-change lines: faint left border in accent colour
- Timestamp prefix: `[HH:MM:SS.mmm]`

---

## StateDiagram.vue

- Vertical list of state nodes: INIT → SCAN → SEND → RECEIVE → SLEEP
- ERROR node displayed inline on the right of SEND and RECEIVE (branches)
- Active state: filled background pill + pulse animation
- Completed states: checkmark icon, muted
- ERROR state (when active): red fill, shake animation
- Each node shows the trigger log keyword as subtitle text (e.g. "Battery: …")
- No SVG required — pure CSS/div approach acceptable

---

## Toolbar

- **Connect button**: disabled while `connectionState === 'connecting'`
  - label: "Connect" when disconnected, "Disconnect" when connected
  - uses `OnyxButton color="primary"` / `color="danger"`
- **Status badge**: `OnyxBadge` showing connection state text
- **Baud rate**: fixed 115200, shown as static text (not configurable in v1)

---

## Error Handling UX

| Scenario | UX |
|---|---|
| User denies port picker | `OnyxInfoCard color="danger"` below toolbar |
| Port closed unexpectedly | Same info card + connection state → disconnected |
| No lines match any state for > 60s | No change — log still scrolls, diagram stays at last known state |
