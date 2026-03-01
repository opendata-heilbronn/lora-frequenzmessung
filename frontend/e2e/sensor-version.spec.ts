import { test, expect } from '@playwright/test'

const BASE_SENSOR = {
  id: 1,
  uuid: 'aaaa1111',
  name: 'sensor-with-version',
  latitude: 49.1438602,
  longitude: 9.2149624,
  type: 'pax',
  dev_eui: '0011223344556677',
  app_key: 'AABBCCDDEEFF00112233445566778899',
  ttn_device_id: 'sensor-aaaa1111',
  created_at: '2026-01-10T08:00:00Z',
  last_battery_value: 82.5,
  last_battery_time: '2026-02-18T09:00:00Z',
  last_data_time: '2026-02-18T10:00:00Z',
  firmware_version: '1.2.0',
  firmware_version_time: '2026-02-18T10:00:00Z',
}

const SENSOR_NO_VERSION = {
  ...BASE_SENSOR,
  id: 2,
  uuid: 'bbbb2222',
  name: 'sensor-no-version',
  ttn_device_id: 'sensor-bbbb2222',
  firmware_version: null,
  firmware_version_time: null,
}

test.describe('Sensor firmware version', () => {
  test('shows version badge when firmware_version is set', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([BASE_SENSOR]),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: 'sensor-with-version' })
    await expect(row.locator('.firmware-badge')).toBeVisible()
    await expect(row.locator('.firmware-badge')).toHaveText('1.2.0')
  })

  test('shows "Unknown" when firmware_version is null', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([SENSOR_NO_VERSION]),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: 'sensor-no-version' })
    await expect(row.locator('.firmware-badge')).toBeVisible()
    await expect(row.locator('.firmware-badge')).toHaveText('Unknown')
  })

  test('request-version button sends POST and shows success toast', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([BASE_SENSOR]),
        })
      } else {
        route.continue()
      }
    })

    let requestVersionCalled = false
    await page.route('**/api/sensors/aaaa1111/request-version', (route) => {
      requestVersionCalled = true
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'version request queued — sensor will report on next wake' }),
      })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: 'sensor-with-version' })
    await row.locator('.version-refresh-btn').click()

    expect(requestVersionCalled).toBe(true)
    await expect(page.locator('.page-success')).toBeVisible()
    await expect(page.locator('.page-success')).toContainText('Version request queued')
  })

  test('request-version shows error toast on failure', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([BASE_SENSOR]),
        })
      } else {
        route.continue()
      }
    })

    await page.route('**/api/sensors/aaaa1111/request-version', (route) => {
      route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'TTN not configured' }),
      })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: 'sensor-with-version' })
    await row.locator('.version-refresh-btn').click()

    await expect(page.locator('.page-error')).toBeVisible()
    await expect(page.locator('.page-error')).toContainText('TTN not configured')
  })

  test('Refresh All Versions button sends POST to request-all-versions', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([BASE_SENSOR]),
        })
      } else {
        route.continue()
      }
    })

    let allVersionsCalled = false
    await page.route('**/api/sensors/request-all-versions', (route) => {
      allVersionsCalled = true
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'version request queued for all linked sensors' }),
      })
    })

    await page.goto('/')

    await page.getByRole('button', { name: 'Refresh All Versions' }).click()

    expect(allVersionsCalled).toBe(true)
    await expect(page.locator('.page-success')).toBeVisible()
    await expect(page.locator('.page-success')).toContainText('all linked sensors')
  })

  test('firmware column header is present', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([BASE_SENSOR]),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    await expect(page.locator('th', { hasText: 'Firmware' })).toBeVisible()
  })
})
