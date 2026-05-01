# Implementation Plan: Sensor Serial Debugger

**Branch**: `001-serial-debugger` | **Date**: 2026-05-01 | **Spec**: specs/001-serial-debugger/
**Input**: User request — browser-based serial debug view with live state diagram

## Summary

New Vue 3 view at `/debug` that connects to the sensor's USB serial port via Web Serial API,
streams raw firmware log output in a left pane, and renders a live state diagram on the right
showing the current firmware state (INIT → SCAN → SEND → RECEIVE → SLEEP / ERROR).
State detection is pure regex against known firmware log prefixes. No backend changes needed.

## Technical Context

**Language/Version**: TypeScript / Vue 3.4 + Vite  
**Primary Dependencies**: `sit-onyx ^1.8.0`, Web Serial API (browser-native), Vue Router 4  
**Storage**: N/A — all state ephemeral, in-component  
**Testing**: Playwright (unit project — mocked `navigator.serial`)  
**Target Platform**: Chrome ≥ 89 / Edge ≥ 89 desktop (Web Serial requirement)  
**Project Type**: Frontend view addition to existing web app  
**Performance Goals**: Log lines rendered at ≥ 115200 baud throughput without jank  
**Constraints**: No new backend routes; Chrome/Edge only; must degrade gracefully in Firefox  
**Scale/Scope**: Single developer, single sprint

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Microservice Separation | ✅ PASS | Pure frontend — no new Backend or ManagementAPI routes |
| II. Test-First | ✅ PASS | Playwright tests required for connect flow, state transitions, error display |
| III. Auth Layers Fixed | ✅ PASS | View behind JWT guard (router.ts `beforeEach`) — no auth changes |
| IV. Async Long-Running | ✅ PASS | Serial reading uses async ReadableStream, no blocking |
| V. Simplicity / YAGNI | ✅ PASS | No new abstractions; regex parser; copy ProvisionStep.vue pattern |

*Post-design re-check: all gates still pass. No complexity tracking entry required.*

## Project Structure

### Documentation (this feature)

```text
specs/001-serial-debugger/
├── plan.md              # this file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── ui-components.md # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code

```text
frontend/
├── src/
│   ├── views/
│   │   └── SerialDebugger.vue          # new — root view, connection logic
│   ├── components/
│   │   ├── SerialLogPane.vue           # new — scrolling log pane
│   │   └── StateDiagram.vue            # new — state machine visualisation
│   ├── router.ts                       # modify — add /debug route
│   └── App.vue                         # modify — add Debug nav item
└── e2e/
    └── debug-serial.spec.ts            # modify — replace placeholder with real tests
```

## Complexity Tracking

> No constitution violations — table not required.
