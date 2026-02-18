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

  page.on('dialog', (dialog) => dialog.accept())
  await page.locator('tr', { hasText: sensorName }).getByRole('button', { name: 'Delete' }).click()
  await expect(page.locator('tr', { hasText: sensorName })).not.toBeVisible()
}

async function fillAndCreateSensor(page: Page) {
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

  test('skip TTN path: create → skip TTN → build → flash → delete', async ({ page }) => {
    await mockCreateSensor(page)
    await mockBuildFirmware(page)
    await mockBuildStatus(page, 'done')

    await page.goto('/add')
    await fillAndCreateSensor(page)

    // Step 2: sensor created
    await expect(page.getByText('Sensor Created')).toBeVisible()
    await expect(page.locator('code').filter({ hasText: SENSOR_UUID })).toBeVisible()

    // Click "Skip TTN"
    await page.getByRole('button', { name: /Skip TTN/ }).click()

    // Step 4: build — should show "without LoRa" notice
    await expect(page.getByText('without LoRa')).toBeVisible()

    // Wait for build "done"
    await expect(page.getByText('Firmware compiled successfully')).toBeVisible()

    // Go to flash step
    await page.getByRole('button', { name: /Flash Sensor/ }).click()

    // Step 5: flash UI
    await expect(page.getByText('Flash Sensor via USB')).toBeVisible()
    await expect(page.getByRole('link', { name: /Back to sensor list/ })).toBeVisible()

    // Cleanup: unroute create mock (it intercepts GET too), set up list+delete mocks, then delete
    await page.unroute('**/api/sensors')
    await mockSensorListAndDelete(page, MOCK_SENSOR)
    await deleteSensorViaUI(page, MOCK_SENSOR.name)
  })

  test('TTN path: create → register TTN → build → flash → delete', async ({ page }) => {
    await mockCreateSensor(page)
    await mockRegisterTTN(page)
    await mockBuildFirmware(page)
    await mockBuildStatus(page, 'done')

    await page.goto('/add')
    await fillAndCreateSensor(page)

    // Step 2: click Register with TTN
    await page.getByRole('button', { name: /Register with TTN/ }).click()

    // Step 3: TTN registration result
    await expect(page.getByText('Device registered with TTN')).toBeVisible()
    await expect(page.getByText(MOCK_TTN.dev_eui, { exact: true })).toBeVisible()
    await expect(page.getByText(MOCK_TTN.app_key, { exact: true })).toBeVisible()

    // Click Build Firmware
    await page.getByRole('button', { name: /Build Firmware/ }).click()

    // Step 4: build done (no "without LoRa" notice)
    await expect(page.getByText('without LoRa')).not.toBeVisible()
    await expect(page.getByText('Firmware compiled successfully')).toBeVisible()

    // Go to flash step
    await page.getByRole('button', { name: /Flash Sensor/ }).click()
    await expect(page.getByText('Flash Sensor via USB')).toBeVisible()

    // Cleanup: swap mocks for sensor list and delete the sensor
    const sensorWithTTN = { ...MOCK_SENSOR, ...MOCK_TTN }
    await page.unroute('**/api/sensors')
    await mockSensorListAndDelete(page, sensorWithTTN)
    await deleteSensorViaUI(page, MOCK_SENSOR.name)
  })

  test('TTN failure shows error and retry button', async ({ page }) => {
    await mockCreateSensor(page)
    await mockRegisterTTN(page, {
      status: 502,
      body: JSON.stringify({ error: 'TTN API unavailable' }),
    })

    await page.goto('/add')
    await fillAndCreateSensor(page)
    await page.getByRole('button', { name: /Register with TTN/ }).click()

    // Error shown
    await expect(page.getByText('TTN registration failed')).toBeVisible()
    await expect(page.getByText('TTN API unavailable')).toBeVisible()

    // Retry button visible
    await expect(page.getByRole('button', { name: /Retry TTN/ })).toBeVisible()

    // "Build Firmware" button also visible (proceed despite error)
    await expect(page.getByRole('button', { name: /Build Firmware/ })).toBeVisible()
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
    await fillAndCreateSensor(page)

    // Error message displayed, still on step 1
    await expect(page.locator('.error-msg')).toBeVisible()
    await expect(page.getByText('Sensor Details')).toBeVisible()
  })

  test('step indicators progress correctly', async ({ page }) => {
    await mockCreateSensor(page)
    await mockRegisterTTN(page)
    await mockBuildFirmware(page)
    await mockBuildStatus(page, 'done')

    await page.goto('/add')

    // Step 1 active
    const steps = page.locator('.step')
    await expect(steps.nth(0)).toHaveClass(/active/)
    await expect(steps.nth(1)).not.toHaveClass(/active|done/)

    await fillAndCreateSensor(page)

    // Step 2 active, step 1 done
    await expect(steps.nth(0)).toHaveClass(/done/)
    await expect(steps.nth(1)).toHaveClass(/active/)

    await page.getByRole('button', { name: /Register with TTN/ }).click()
    await expect(page.getByText('Device registered with TTN')).toBeVisible()

    // Step 3 active, steps 1-2 done
    await expect(steps.nth(0)).toHaveClass(/done/)
    await expect(steps.nth(1)).toHaveClass(/done/)
    await expect(steps.nth(2)).toHaveClass(/active/)

    await page.getByRole('button', { name: /Build Firmware/ }).click()
    await expect(page.getByText('Firmware compiled successfully')).toBeVisible()

    // Step 4 active, steps 1-3 done
    await expect(steps.nth(2)).toHaveClass(/done/)
    await expect(steps.nth(3)).toHaveClass(/active/)

    await page.getByRole('button', { name: /Flash Sensor/ }).click()

    // Step 5 active, steps 1-4 done
    await expect(steps.nth(3)).toHaveClass(/done/)
    await expect(steps.nth(4)).toHaveClass(/active/)
  })
})
