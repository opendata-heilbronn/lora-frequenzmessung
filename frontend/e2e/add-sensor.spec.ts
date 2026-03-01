import { test, expect, type Page } from '@playwright/test'

const SENSOR_UUID = 'aabbccdd-1234-5678-9abc-def012345678'

const MOCK_SENSOR = {
  id: 1,
  uuid: SENSOR_UUID,
  name: 'test-sensor-01',
  type: 'pax',
  latitude: 49.1438602,
  longitude: 9.2149624,
  dev_eui: '',
  app_key: '',
  ttn_device_id: '',
  created_at: '2026-01-15T10:30:00Z',
}

const MOCK_TTN = {
  dev_eui: '0011223344556677',
  app_key: 'AABBCCDDEEFF00112233445566778899',
  ttn_device_id: 'test-sensor-01',
}

function mockCreateSensor(page: Page) {
  return page.route('**/api/sensors', (route) => {
    if (route.request().method() === 'POST') {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
    } else {
      route.continue()
    }
  })
}

function mockRegisterTTN(page: Page, response?: { status: number; body: string }) {
  return page.route(`**/api/sensors/${SENSOR_UUID}/register-ttn`, (route) => {
    route.fulfill(response ?? { status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_TTN) })
  })
}

function mockBuildFirmware(page: Page) {
  return page.route(`**/api/sensors/${SENSOR_UUID}/build-firmware`, (route) => {
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'building' }) })
  })
}

function mockBuildStatus(page: Page, status: 'building' | 'done' | 'error' = 'done') {
  return page.route(`**/api/sensors/${SENSOR_UUID}/build-status`, (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ status, message: status === 'error' ? 'Compile failed' : '' }),
    })
  })
}

/** Mock GET /api/sensors to return the created sensor, and DELETE to remove it. */
function mockSensorListAndDelete(page: Page, sensor: typeof MOCK_SENSOR) {
  return Promise.all([
    page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([sensor]) })
      } else {
        route.continue()
      }
    }),
    page.route(`**/api/sensors/${sensor.uuid}`, (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
      } else {
        route.continue()
      }
    }),
  ])
}

/** Navigate back to sensor list and delete the created sensor via the Delete button. */
async function deleteSensorViaUI(page: Page, sensorName: string) {
  await page.getByRole('link', { name: /Back to sensor list/ }).click()
  await expect(page.locator('tr', { hasText: sensorName })).toBeVisible()

  const row = page.locator('tr', { hasText: sensorName })
  await row.getByRole('button', { name: 'Delete' }).click()
  await row.getByRole('button', { name: 'Yes' }).click()
  await expect(page.locator('tr', { hasText: sensorName })).not.toBeVisible()
}

async function fillAndSubmitForm(page: Page) {
  await page.getByLabel('Name').fill('test-sensor-01')
  await page.getByLabel('Latitude').fill('49.1438602')
  await page.getByLabel('Longitude').fill('9.2149624')
  await page.getByRole('button', { name: /Create Sensor/ }).click()
}

test.describe('AddSensor wizard', () => {
  test.beforeEach(async ({ page }) => {
    // Block manifest requests so esp-web-tools doesn't make real calls
    await page.route('**/api/sensors/*/manifest.json', (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ name: 'test' }) })
    })
  })

  test('full flash-and-provision flow: erase → write → reboot → provision → done', async ({ page }) => {
    // Provision responses consumed in order: PROV_READY + one OK per key line + RESTART for COMMIT
    await page.addInitScript(() => {
      const responses = [
        'PROV_READY\n',
        'OK\n', // sensor_id
        'OK\n', // factor
        'OK\n', // sleep_sec
        'OK\n', // joineui
        'OK\n', // deveui
        'OK\n', // appkey
        'RESTART\n', // COMMIT
      ]
      let idx = 0
      const mockPort = {
        open: async () => {},
        close: async () => {},
        readable: {
          getReader: () => ({
            read: async () => ({ value: new TextEncoder().encode(responses[idx++] ?? 'OK\n'), done: false }),
            releaseLock: () => {},
          }),
        },
        writable: {
          getWriter: () => ({ write: async () => {}, releaseLock: () => {} }),
        },
      }
      Object.defineProperty(navigator, 'serial', {
        value: { requestPort: async () => mockPort, getPorts: async () => [] },
        configurable: true,
      })
      // Mock the esp-web-tools flash function so no real serial/esptool is needed.
      // Fires erase → write → finished events synchronously, then returns.
      ;(window as any).__espFlash = async (onEvent: (s: any) => void) => {
        onEvent({ state: 'erasing', message: 'Erasing...', details: { done: false } })
        onEvent({ state: 'writing', message: '50%', details: { bytesTotal: 100, bytesWritten: 50, percentage: 50 } })
        onEvent({ state: 'finished', message: 'All done!' })
      }
      // Skip the 4-second reboot wait in tests
      ;(window as any).__rebootWaitMs = 50
    })

    await mockCreateSensor(page)
    await mockRegisterTTN(page)
    await mockBuildFirmware(page)
    await mockBuildStatus(page, 'done')

    await page.goto('/add')
    await fillAndSubmitForm(page)

    // Wait for build to finish and navigate to flash stage
    await expect(page.getByRole('button', { name: /Flash Sensor/ })).toBeVisible({ timeout: 10000 })
    await page.getByRole('button', { name: /Flash Sensor/ }).click()
    await expect(page.getByText('Flash & Provision Sensor')).toBeVisible()

    // Click the custom flash button (not esp-web-install-button)
    await page.getByRole('button', { name: /Connect & Flash/ }).click()

    // Full chain must complete: provisioning runs after flash, then done
    await expect(page.getByText('Device flashed and provisioned successfully!')).toBeVisible({ timeout: 10000 })
  })

  test('after create, TTN registration starts automatically', async ({ page }) => {
    await mockCreateSensor(page)

    const ttnRequests: string[] = []
    await page.route(`**/api/sensors/${SENSOR_UUID}/register-ttn`, (route) => {
      ttnRequests.push(route.request().url())
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_TTN) })
    })
    await mockBuildFirmware(page)
    await mockBuildStatus(page, 'done')

    await page.goto('/add')
    await fillAndSubmitForm(page)

    // TTN should start without any button click — "Setting Up Sensor" card appears
    await expect(page.getByText('Setting Up Sensor')).toBeVisible()
    // TTN result appears automatically (no button click)
    await expect(page.getByText('TTN registration')).toBeVisible()
    expect(ttnRequests.length).toBeGreaterThan(0)
  })

  test('after TTN success, firmware build starts automatically', async ({ page }) => {
    await mockCreateSensor(page)
    await mockRegisterTTN(page)
    await mockBuildFirmware(page)
    await mockBuildStatus(page, 'done')

    await page.goto('/add')
    await fillAndSubmitForm(page)

    // Wait for TTN to complete (credentials shown)
    await expect(page.getByText('0011****6677')).toBeVisible()

    // Build starts automatically — "Building firmware…" appears without any button click
    await expect(page.getByText('Building firmware…')).toBeVisible()

    // Build completes, Flash button appears
    await expect(page.getByRole('button', { name: /Flash Sensor/ })).toBeVisible({ timeout: 10000 })
  })

  test('Flash button appears only after build completes', async ({ page }) => {
    await mockCreateSensor(page)
    await mockRegisterTTN(page)
    await mockBuildFirmware(page)

    let buildDone = false
    await page.route(`**/api/sensors/${SENSOR_UUID}/build-status`, (route) => {
      const status = buildDone ? 'done' : 'building'
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status }) })
    })

    await page.goto('/add')
    await fillAndSubmitForm(page)

    // Wait for build to start
    await expect(page.getByText('Building firmware…')).toBeVisible()

    // Flash button must NOT be present during build
    await expect(page.getByRole('button', { name: /Flash Sensor/ })).not.toBeVisible()

    // Allow build to complete on next poll
    buildDone = true

    // Flash button must appear once done
    await expect(page.getByRole('button', { name: /Flash Sensor/ })).toBeVisible({ timeout: 10000 })
  })

  test('TTN failure shows retry and skip buttons', async ({ page }) => {
    await mockCreateSensor(page)
    await mockRegisterTTN(page, {
      status: 502,
      body: JSON.stringify({ error: 'TTN API unavailable' }),
    })

    await page.goto('/add')
    await fillAndSubmitForm(page)

    // Error shown automatically (no button click)
    await expect(page.getByText('TTN API unavailable')).toBeVisible()

    // Both action buttons are visible
    await expect(page.getByRole('button', { name: /Retry TTN/ })).toBeVisible()
    await expect(page.getByRole('button', { name: /Skip TTN/ })).toBeVisible()
  })

  test('skip TTN button proceeds to build without TTN credentials', async ({ page }) => {
    await mockCreateSensor(page)
    await mockRegisterTTN(page, {
      status: 502,
      body: JSON.stringify({ error: 'TTN API unavailable' }),
    })
    await mockBuildFirmware(page)
    await mockBuildStatus(page, 'done')

    await page.goto('/add')
    await fillAndSubmitForm(page)

    // Wait for TTN error
    await expect(page.getByText('TTN API unavailable')).toBeVisible()

    // Click Skip TTN — build starts without LoRa credentials
    await page.getByRole('button', { name: /Skip TTN/i }).click()

    // Build completes and Flash button appears
    await expect(page.getByRole('button', { name: /Flash Sensor/ })).toBeVisible({ timeout: 10000 })

    // Click Flash — goes to Flash & Provision stage (no TTN data)
    await page.getByRole('button', { name: /Flash Sensor/ }).click()
    await expect(page.getByText('Flash & Provision Sensor')).toBeVisible()
  })

  test('create sensor error displays message', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: 'DB connection failed' }) })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')
    await fillAndSubmitForm(page)

    // Error message displayed, still on form (step 1)
    await expect(page.locator('.error-msg')).toBeVisible()
    await expect(page.getByText('Sensor Details')).toBeVisible()
  })

  test('full happy path: auto TTN + auto build → Flash Sensor button → flash stage', async ({ page }) => {
    await mockCreateSensor(page)
    await mockRegisterTTN(page)
    await mockBuildFirmware(page)
    await mockBuildStatus(page, 'done')

    await page.goto('/add')

    // Stage 1: form
    await expect(page.getByText('Sensor Details')).toBeVisible()
    await fillAndSubmitForm(page)

    // Stage 2: setup card — sensor created row visible
    await expect(page.getByText('Setting Up Sensor')).toBeVisible()
    await expect(page.getByText(SENSOR_UUID)).toBeVisible()

    // TTN credentials shown (masked)
    await expect(page.getByText('0011****6677')).toBeVisible()
    await expect(page.getByText('AABB****8899')).toBeVisible()
    // Copy buttons present
    await expect(page.getByRole('button', { name: 'Copy' }).first()).toBeVisible()

    // Flash button appears after build completes
    await expect(page.getByRole('button', { name: /Flash Sensor/ })).toBeVisible({ timeout: 10000 })

    // Stage 3: clicking Flash shows FlashAndProvisionStep
    await page.getByRole('button', { name: /Flash Sensor/ }).click()
    await expect(page.getByText('Flash & Provision Sensor')).toBeVisible()
    await expect(page.getByRole('link', { name: /Back to sensor list/ })).toBeVisible()

    // Cleanup
    const sensorWithTTN = { ...MOCK_SENSOR, ...MOCK_TTN }
    await page.unroute('**/api/sensors')
    await mockSensorListAndDelete(page, sensorWithTTN)
    await deleteSensorViaUI(page, MOCK_SENSOR.name)
  })
})
