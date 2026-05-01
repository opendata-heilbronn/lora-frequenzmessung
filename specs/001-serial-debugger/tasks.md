---
description: "Task list for Sensor Serial Debugger"
---

# Tasks: Sensor Serial Debugger

**Input**: Design documents from `specs/001-serial-debugger/`
**Prerequisites**: plan.md ✅, research.md ✅, data-model.md ✅, contracts/ui-components.md ✅

**User Stories derived from user request + plan.md (no spec.md)**:
- **US1** (P1): Serial connection + raw log display — connect to port, stream lines with timestamps
- **US2** (P2): Live state diagram — parse log lines, animate current firmware state
- **US3** (P3): Error visualization — highlight error lines red in log + ERROR node in diagram

Tests are **required** per CLAUDE.md and constitution Principle II.

---

## Phase 1: Setup

**Purpose**: Add route, nav entry, and type definitions shared by all components.

- [x] T001 Add `/debug` route to `frontend/src/router.ts` importing `SerialDebugger.vue`
- [x] T002 Add `<OnyxNavItem label="Debug" link="/debug" />` to `frontend/src/App.vue` nav bar
- [x] T003 [P] Create `SensorState` type, `LogLine` interface, `ConnectionState` type in `frontend/src/types/serial-debugger.ts` (from data-model.md)
- [x] T004 [P] Create `STATE_PATTERNS` regex array (10 patterns from research.md) in `frontend/src/utils/serial-parser.ts` with exported `parseLine(text): { state: SensorState | null; isError: boolean }` function

---

## Phase 2: Foundational

**Purpose**: Serial port plumbing used by all user stories.

- [x] T005 Create `frontend/src/composables/useSerialPort.ts` — exposes `connectionState`, `connect()`, `disconnect()`, `onLine(cb)` using Web Serial API (pattern from `ProvisionStep.vue`: `navigator.serial.requestPort()`, `port.open({ baudRate: 115200 })`, `ReadableStream` line buffering)
- [x] T006 Add `webSerialSupported` check in `useSerialPort.ts` — returns `false` when `!('serial' in navigator)`

**Checkpoint**: Port composable ready — all user story components can import it.

---

## Phase 3: User Story 1 — Serial Connection + Raw Log (P1) 🎯 MVP

**Goal**: User connects to sensor USB port and sees live timestamped raw log lines.

**Independent Test**: Navigate to `/debug`, mock serial, click Connect, verify lines appear with `[HH:MM:SS.mmm]` timestamps. Disconnect resets view.

### Playwright Tests for US1 ⚠ Write first — must FAIL before implementation

- [x] T007 [P] [US1] Write Playwright test: `/debug` shows Web Serial unsupported warning when `navigator.serial` removed — in `frontend/e2e/debug-serial.spec.ts` (replaces placeholder)
- [x] T008 [P] [US1] Write Playwright test: Connect flow — mock `navigator.serial`, click Connect, inject 3 log lines, verify all 3 appear in log pane with timestamp prefix — in `frontend/e2e/debug-serial.spec.ts`
- [x] T009 [P] [US1] Write Playwright test: Disconnect button closes port and resets log — in `frontend/e2e/debug-serial.spec.ts`

### Implementation for US1

- [x] T010 [US1] Create `frontend/src/components/SerialLogPane.vue` — props: `lines: LogLine[]`, `autoScroll: boolean`; renders `<pre>` with `[HH:MM:SS.mmm] <text>` per line; auto-scrolls to bottom unless user scrolled up; uses `sit-onyx` tokens for colours
- [x] T011 [US1] Create `frontend/src/views/SerialDebugger.vue` — imports `useSerialPort`, renders toolbar (Connect/Disconnect `OnyxButton`, `OnyxBadge` connection status), `OnyxInfoCard` warning for unsupported browsers, `SerialLogPane` in left pane; calls `parseLine` on each incoming line to build `LogLine` array
- [x] T012 [US1] Wire `onLine` callback in `SerialDebugger.vue` to push `LogLine` (id, text, timestamp, state, isError) from `serial-parser.ts` result

**Checkpoint**: US1 independently testable — `npx playwright test e2e/debug-serial.spec.ts`

---

## Phase 4: User Story 2 — Live State Diagram (P2)

**Goal**: Right pane shows state machine (INIT→SCAN→SEND→RECEIVE→SLEEP) with current state highlighted and animated.

**Independent Test**: Inject log lines `Battery:`, `PAX count:`, `Payload:`, `TX ok`, `Sleeping for X` — verify diagram node highlights advance in order.

### Playwright Tests for US2 ⚠ Write first — must FAIL before implementation

- [x] T013 [P] [US2] Write Playwright test: inject `Battery: 3.80V (72.5%)` → INIT node active — in `frontend/e2e/debug-serial.spec.ts`
- [x] T014 [P] [US2] Write Playwright test: inject full cycle lines → diagram advances INIT→SCAN→SEND→RECEIVE→SLEEP in order — in `frontend/e2e/debug-serial.spec.ts`

### Implementation for US2

- [x] T015 [US2] Create `frontend/src/components/StateDiagram.vue` — prop `currentState: SensorState`; renders vertical node list (IDLE, INIT, SCAN, SEND, RECEIVE, SLEEP, ERROR); active node gets filled background + CSS pulse animation; completed nodes show checkmark; each node shows trigger keyword subtitle (from `STATE_PATTERNS` labels)
- [x] T016 [US2] Add `currentState` reactive ref to `SerialDebugger.vue`, updated by `parseLine` result; pass to `StateDiagram.vue` in right pane column

**Checkpoint**: US2 independently testable — state diagram advances with injected log sequence.

---

## Phase 5: User Story 3 — Error Visualization (P3)

**Goal**: Error log lines highlighted red in log pane; ERROR node in diagram lights up red with shake animation when an error state is reached.

**Independent Test**: Inject `TX error -112` → error line red in log, ERROR node active (red, shake), diagram does not advance further.

### Playwright Tests for US3 ⚠ Write first — must FAIL before implementation

- [x] T017 [P] [US3] Write Playwright test: inject `TX error -112` → log line has error styling (check CSS class or computed color) — in `frontend/e2e/debug-serial.spec.ts`
- [x] T018 [P] [US3] Write Playwright test: inject `Radio init failed` → ERROR node is active in diagram — in `frontend/e2e/debug-serial.spec.ts`

### Implementation for US3

- [x] T019 [US3] Style error lines in `SerialLogPane.vue` — `isError: true` lines rendered with `color: var(--onyx-color-danger-500)` and `font-weight: bold`
- [x] T020 [US3] Style ERROR node in `StateDiagram.vue` when `currentState === 'ERROR'` — red fill background, CSS `@keyframes shake` animation; pass `errorLines: LogLine[]` prop and show count badge on ERROR node
- [x] T021 [US3] Add `errorLines` computed ref in `SerialDebugger.vue` filtering `lines` where `isError === true`; pass to `StateDiagram.vue`

**Checkpoint**: All 3 user stories functional. Run full suite: `npx playwright test e2e/debug-serial.spec.ts`

---

## Phase 6: Polish & Cross-Cutting

**Purpose**: Responsive layout, error UX, quickstart validation.

- [x] T022 Add two-column CSS grid layout to `SerialDebugger.vue` (`display: grid; grid-template-columns: 1fr 1fr`) with `@media (max-width: 768px)` stacking to single column
- [x] T023 Add `OnyxInfoCard color="danger"` error display in `SerialDebugger.vue` for port picker denial and unexpected disconnect (error message from `useSerialPort`)
- [x] T024 Run `npm run build` from `frontend/` — fix any TypeScript errors
- [x] T025 Run `npx playwright test e2e/debug-serial.spec.ts` — all tests passing
- [x] T026 Run full suite `npm run test:e2e` — verify no regressions in other tests

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately; T003 and T004 are parallel
- **Foundational (Phase 2)**: Depends on Phase 1 complete (needs types from T003)
- **US1 (Phase 3)**: Depends on Foundational complete. Tests T007–T009 written first (must FAIL)
- **US2 (Phase 4)**: Depends on US1 complete (StateDiagram added to SerialDebugger.vue which US1 created)
- **US3 (Phase 5)**: Depends on US2 complete (extends StateDiagram)
- **Polish (Phase 6)**: Depends on all stories complete

### Within Each User Story

- Tests MUST be written first and FAIL before implementation
- `serial-parser.ts` (T004) before `SerialLogPane.vue` (T010) and `SerialDebugger.vue` (T011)
- `useSerialPort.ts` (T005) before `SerialDebugger.vue` (T011)
- `SerialDebugger.vue` (T011) before `StateDiagram.vue` integration (T016)

### Parallel Opportunities

- T003 and T004 (Phase 1) — different files
- T007, T008, T009 (US1 tests) — all go in same file but can be drafted in parallel
- T013, T014 (US2 tests) — parallel
- T017, T018 (US3 tests) — parallel
- T019, T020 (US3 impl) — different components

---

## Implementation Strategy

### MVP First (US1 only — ~6 tasks after setup)

1. Phase 1: T001–T004
2. Phase 2: T005–T006
3. Phase 3 tests: T007–T009 (write, verify they FAIL)
4. Phase 3 impl: T010–T012
5. **STOP and VALIDATE**: `npx playwright test e2e/debug-serial.spec.ts`

### Incremental Delivery

1. US1 complete → working serial monitor in browser (MVP!)
2. US2 complete → state diagram animates with firmware
3. US3 complete → error highlighting + visual error node
4. Polish complete → responsive, production-ready

---

## Notes

- [P] = different files, no blocking dependency
- [US1/US2/US3] = story label for traceability
- `debug-serial.spec.ts` already exists as placeholder — replace, don't create new file
- All serial tests use `page.addInitScript()` to mock `navigator.serial` (unit project, no real device needed)
- Baud rate 115200 fixed — no config UI in v1
- `ENABLE_LOGGING=1` must be set in firmware build for log output to appear (document in quickstart)
