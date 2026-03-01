import { test, expect } from '@playwright/test'

// Tests for the Flash & Configure (FlashAndProvisionStep) action on the Sensor List rebuild panel.
// These run in the "unit" Playwright project against the Vite dev server
// with mocked backend routes.

const SENSORS = [
  {
    id: 1,
    uuid: 'abcd1234',
    name: 'density-linked',
    latitude: 49.1438602,
    longitude: 9.2149624,
    type: 'pax',
    dev_eui: '0011223344556677',
    app_key: 'AABBCCDDEEFF00112233445566778899',
    ttn_device_id: 'density-linked',
    created_at: '2026-01-10T08:00:00Z',
    last_battery_value: 88,
    last_battery_time: '2026-02-10T08:00:00Z',
    last_data_time: '2026-02-10T08:00:00Z',
  },
]

function mockSensorList(page: import('@playwright/test').Page) {
  return page.route('**/api/sensors', (route) => {
    if (route.request().method() === 'GET') {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
    } else {
      route.continue()
    }
  })
}

async function mockBuildDone(page: import('@playwright/test').Page, uuid: string) {
  let polls = 0
  await page.route(`**/api/sensors/${uuid}/build-firmware`, (route) => {
    route.fulfill({ status: 202, contentType: 'application/json', body: JSON.stringify({ status: 'building' }) })
  })
  await page.route(`**/api/sensors/${uuid}/build-status`, (route) => {
    polls++
    const status = polls >= 2 ? 'done' : 'building'
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status }) })
  })
}

test.describe('Sensor List → Rebuild panel with Flash & Configure', () => {
  test('shows single Flash & Configure button (no separate Flash/Provision) after build', async ({ page }) => {
    await mockSensorList(page)
    await mockBuildDone(page, SENSORS[0].uuid)

    await page.goto('/')

    const row = page.locator('tr', { hasText: SENSORS[0].name })
    await row.getByRole('button', { name: 'Rebuild' }).click()

    // Wait for build done state
    await expect(page.getByText('Firmware compiled successfully!')).toBeVisible({ timeout: 10000 })

    // Single unified button replaces separate Flash Sensor + Provision
    await expect(page.getByRole('button', { name: 'Flash & Configure' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Flash Sensor' })).not.toBeVisible()
    await expect(page.getByRole('button', { name: 'Provision' })).not.toBeVisible()
  })

  test('Flash & Configure opens FlashAndProvisionStep with custom flash button', async ({ page }) => {
    await mockSensorList(page)
    await mockBuildDone(page, SENSORS[0].uuid)

    await page.goto('/')

    const row = page.locator('tr', { hasText: SENSORS[0].name })
    await row.getByRole('button', { name: 'Rebuild' }).click()

    await expect(page.getByText('Firmware compiled successfully!')).toBeVisible({ timeout: 10000 })
    await page.getByRole('button', { name: 'Flash & Configure' }).click()

    // FlashAndProvisionStep idle phase: custom flash button is shown
    await expect(page.getByRole('button', { name: /Connect & Flash/ })).toBeVisible()

    // Close dismisses the panel
    await page.getByRole('button', { name: 'Close' }).click()
    await expect(page.getByText('Firmware compiled successfully!')).not.toBeVisible()
  })

  test('Flash & Configure shows Web Serial warning in unsupported browser', async ({ page }) => {
    // Remove navigator.serial entirely so 'serial' in navigator is false
    await page.addInitScript(() => {
      try {
        // Delete from prototype so the property doesn't exist at all
        delete (Navigator.prototype as any).serial
      } catch {
        // If not deletable, define it as non-existent via a configurable getter that throws
      }
    })

    await mockSensorList(page)
    await mockBuildDone(page, SENSORS[0].uuid)

    await page.goto('/')

    const row = page.locator('tr', { hasText: SENSORS[0].name })
    await row.getByRole('button', { name: 'Rebuild' }).click()

    await expect(page.getByText('Firmware compiled successfully!')).toBeVisible({ timeout: 10000 })
    await page.getByRole('button', { name: 'Flash & Configure' }).click()

    // Web Serial warning should appear instead of the flash button
    await expect(page.getByText('Web Serial not supported')).toBeVisible()
    await expect(page.getByRole('button', { name: /Connect & Flash/ })).not.toBeVisible()
  })
})
