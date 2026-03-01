import { test, expect } from '@playwright/test'

const SENSOR_WITH_TTN = {
  id: 1,
  uuid: 'aaaa1111',
  name: 'sensor-ota-linked',
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
  firmware_version: '1.2',
  firmware_version_time: '2026-02-18T10:00:00Z',
}

const SENSOR_NO_TTN = {
  ...SENSOR_WITH_TTN,
  id: 2,
  uuid: 'bbbb2222',
  name: 'sensor-ota-unlinked',
  ttn_device_id: '',
  dev_eui: '',
  app_key: '',
}

function mockSensors(page: any, sensors: any[]) {
  return page.route('**/api/sensors', (route: any) => {
    if (route.request().method() === 'GET') {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(sensors),
      })
    } else {
      route.continue()
    }
  })
}

test.describe('Trigger OTA', () => {
  test('Trigger OTA button is disabled when sensor has no TTN device', async ({ page }) => {
    await mockSensors(page, [SENSOR_NO_TTN])
    await page.goto('/')

    const row = page.locator('tr', { hasText: 'sensor-ota-unlinked' })
    const otaBtn = row.getByRole('button', { name: 'Trigger OTA' })
    await expect(otaBtn).toBeVisible()
    await expect(otaBtn).toBeDisabled()
  })

  test('Trigger OTA sends POST and shows success toast with version info', async ({ page }) => {
    await mockSensors(page, [SENSOR_WITH_TTN])

    let otaCalled = false
    await page.route('**/api/sensors/aaaa1111/trigger-ota', (route) => {
      otaCalled = true
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          message: 'OTA queued (1.2 → 1.3) — sensor updates on next wake',
          tag: 'firmware-pax-v1.3',
        }),
      })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: 'sensor-ota-linked' })
    await row.getByRole('button', { name: 'Trigger OTA' }).click()

    expect(otaCalled).toBe(true)
    await expect(page.locator('.page-success')).toBeVisible()
    await expect(page.locator('.page-success')).toContainText('OTA queued')
    await expect(page.locator('.page-success')).toContainText('1.2')
    await expect(page.locator('.page-success')).toContainText('1.3')
  })

  test('Trigger OTA shows info card when response is 409 (already latest)', async ({ page }) => {
    await mockSensors(page, [SENSOR_WITH_TTN])

    await page.route('**/api/sensors/aaaa1111/trigger-ota', (route) => {
      route.fulfill({
        status: 409,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'sensor is already on the latest version (1.2)' }),
      })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: 'sensor-ota-linked' })
    await row.getByRole('button', { name: 'Trigger OTA' }).click()

    await expect(page.locator('.page-info')).toBeVisible()
    await expect(page.locator('.page-info')).toContainText('latest version')
    // Must NOT show error card for a 409
    await expect(page.locator('.page-error')).not.toBeVisible()
  })

  test('Trigger OTA shows error toast on 500 failure', async ({ page }) => {
    await mockSensors(page, [SENSOR_WITH_TTN])

    await page.route('**/api/sensors/aaaa1111/trigger-ota', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'downlink failed: TTN unreachable' }),
      })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: 'sensor-ota-linked' })
    await row.getByRole('button', { name: 'Trigger OTA' }).click()

    await expect(page.locator('.page-error')).toBeVisible()
    await expect(page.locator('.page-error')).toContainText('downlink failed')
  })
})
