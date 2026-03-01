import { test, expect, type Page, type Route } from '@playwright/test'

const SENSOR_UUID = 'aabbccdd'

const MOCK_SENSOR = {
  id: 1,
  uuid: SENSOR_UUID,
  name: 'ttn-test-sensor',
  type: 'density',
  latitude: 49.14,
  longitude: 9.21,
  dev_eui: '',
  app_key: '',
  ttn_device_id: '',
  created_at: '2026-02-01T10:00:00Z',
}

const MOCK_TTN = {
  dev_eui: '0011223344556677',
  app_key: 'AABBCCDDEEFF00112233445566778899',
  ttn_device_id: 'sensor-aabbccdd',
}

test.describe('TTN Registration', () => {
  test.beforeEach(async ({ page }) => {
    // Block all sensor sub-resource calls not explicitly mocked in each test
    // to prevent unmocked requests triggering 401 → /login redirect.
    // Test-level mocks (registered after beforeEach) take priority.
    await page.route('**/api/sensors/*/**', (route) => route.abort())
    await page.route('**/api/sensors/*/manifest.json', (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ name: 'test' }) })
    })
  })

  async function fillAndCreate(page: Page) {
    await page.getByLabel('Name').fill('ttn-test-sensor')
    await page.getByLabel('Latitude').fill('49.14')
    await page.getByLabel('Longitude').fill('9.21')
    await page.getByRole('button', { name: /Create Sensor/ }).click()
  }

  test('register-ttn sends POST to correct endpoint automatically after create', async ({ page }) => {
    const apiCalls: { method: string; url: string }[] = []

    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        apiCalls.push({ method: 'POST', url: route.request().url() })
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })

    await page.route(`**/api/sensors/${SENSOR_UUID}/register-ttn`, (route) => {
      apiCalls.push({ method: route.request().method(), url: route.request().url() })
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_TTN) })
    })

    await page.goto('/add')
    await fillAndCreate(page)

    // TTN fires automatically — credentials appear without any button click
    await expect(page.getByText('0011****6677')).toBeVisible()

    const ttnCall = apiCalls.find(c => c.url.includes('register-ttn'))
    expect(ttnCall).toBeTruthy()
    expect(ttnCall!.method).toBe('POST')
    expect(ttnCall!.url).toContain(`/api/sensors/${SENSOR_UUID}/register-ttn`)
  })

  test('TTN credentials are masked on screen with copy buttons', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })
    await page.route(`**/api/sensors/${SENSOR_UUID}/register-ttn`, (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_TTN) })
    })

    await page.goto('/add')
    await fillAndCreate(page)

    // DevEUI masked: first 4 + **** + last 4
    await expect(page.getByText('0011****6677')).toBeVisible()
    await expect(page.getByText('0011223344556677', { exact: true })).not.toBeVisible()

    // AppKey masked
    await expect(page.getByText('AABB****8899')).toBeVisible()
    await expect(page.getByText('AABBCCDDEEFF00112233445566778899', { exact: true })).not.toBeVisible()

    // Two Copy buttons (DevEUI + AppKey)
    const copyButtons = page.getByRole('button', { name: 'Copy' })
    await expect(copyButtons).toHaveCount(2)
  })

  test('TTN registration failure shows error with backend message', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })
    await page.route(`**/api/sensors/${SENSOR_UUID}/register-ttn`, (route) => {
      route.fulfill({
        status: 502,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'TTN registration failed: NS registration: HTTP 500' }),
      })
    })

    await page.goto('/add')
    await fillAndCreate(page)

    // Error appears automatically — no button click needed
    await expect(page.getByText('TTN registration failed: NS registration: HTTP 500')).toBeVisible()
    await expect(page.getByRole('button', { name: /Retry TTN/ })).toBeVisible()
  })

  test('TTN retry after failure re-sends registration request', async ({ page }) => {
    let ttnCallCount = 0

    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })
    await page.route(`**/api/sensors/${SENSOR_UUID}/register-ttn`, (route) => {
      ttnCallCount++
      if (ttnCallCount === 1) {
        route.fulfill({
          status: 502,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'TTN timeout' }),
        })
      } else {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_TTN),
        })
      }
    })

    await page.goto('/add')
    await fillAndCreate(page)

    // First attempt fails automatically
    await expect(page.getByText('TTN timeout')).toBeVisible()
    expect(ttnCallCount).toBe(1)

    // Click retry — second attempt succeeds
    await page.getByRole('button', { name: /Retry TTN/ }).click()
    await expect(page.getByText('0011****6677')).toBeVisible()
    expect(ttnCallCount).toBe(2)
  })

  test('sensor already linked to TTN shows 409 error', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })
    await page.route(`**/api/sensors/${SENSOR_UUID}/register-ttn`, (route) => {
      route.fulfill({
        status: 409,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'sensor already linked to TTN' }),
      })
    })

    await page.goto('/add')
    await fillAndCreate(page)

    // Error appears automatically
    await expect(page.getByText('already linked to TTN')).toBeVisible()
  })

  test('build firmware after TTN does not require manual steps', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })
    await page.route(`**/api/sensors/${SENSOR_UUID}/register-ttn`, (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_TTN) })
    })
    await page.route(`**/api/sensors/${SENSOR_UUID}/build-firmware`, (route) => {
      route.fulfill({ status: 202, contentType: 'application/json', body: JSON.stringify({ status: 'building' }) })
    })
    await page.route(`**/api/sensors/${SENSOR_UUID}/build-status`, (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'done', message: '' }) })
    })

    await page.goto('/add')
    await fillAndCreate(page)

    // TTN + build chain runs automatically — Flash button appears with no manual interaction
    await expect(page.getByRole('button', { name: /Flash Sensor/ })).toBeVisible({ timeout: 10000 })

    // "without LoRa" text never shown when TTN succeeded
    await expect(page.getByText('without LoRa')).not.toBeVisible()
  })

  test.skip('build firmware without TTN shows "without LoRa" notice', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })
    await page.route(`**/api/sensors/${SENSOR_UUID}/build-firmware`, (route) => {
      route.fulfill({ status: 202, contentType: 'application/json', body: JSON.stringify({ status: 'building' }) })
    })
    await page.route(`**/api/sensors/${SENSOR_UUID}/build-status`, (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'done', message: '' }) })
    })

    await page.goto('/add')
    await fillAndCreate(page)

    // Skip TTN
    await page.getByRole('button', { name: /Skip TTN/ }).click()

    // Should show "without LoRa" notice
    await expect(page.getByText('without LoRa')).toBeVisible()
    await expect(page.getByText('Firmware compiled successfully')).toBeVisible()
  })
})
