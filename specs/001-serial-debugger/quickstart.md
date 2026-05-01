# Quickstart: Sensor Serial Debugger

## Validate the implementation

1. Flash a sensor with `ENABLE_LOGGING=1` firmware (or use a dev build).
2. Connect sensor via USB.
3. Open Chrome or Edge (desktop).
4. Navigate to `http://localhost:5173/debug`.
5. Click **Connect** → select the sensor's COM/tty port.
6. Power-cycle or reset the sensor.
7. Observe:
   - Raw log lines appear in the left pane with timestamps.
   - State diagram on the right advances: INIT → SCAN → SEND → RECEIVE → SLEEP.
   - Any `TX error` or `Radio init failed` line turns red in log and ERROR node lights up.

## Playwright smoke-test (mocked)

```bash
cd frontend
npx playwright test e2e/debug-serial.spec.ts
```

Expected: all tests pass, state transitions exercised via mocked `navigator.serial`.

## Browser requirement

Chrome ≥ 89 or Edge ≥ 89 (desktop). Web Serial is not available in Firefox or Safari.
The view shows a warning `OnyxInfoCard` on unsupported browsers.
