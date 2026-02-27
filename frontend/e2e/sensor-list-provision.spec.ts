import { test, expect } from '@playwright/test'

// Tests for the new Provision action on the Sensor List rebuild panel.
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

 test.describe('Sensor List → Rebuild panel with Provision', () => {
  test('shows both Flash Sensor and Provision after build', async ({ page }) => {
    await mockSensorList(page)

    const uuid = SENSORS[0].uuid
    let polls = 0

    await page.route(`**/api/sensors/${uuid}/build-firmware`, (route) => {
      route.fulfill({ status: 202, contentType: 'application/json', body: JSON.stringify({ status: 'building' }) })
    })

    await page.route(`**/api/sensors/${uuid}/build-status`, (route) => {
      polls++
      const status = polls >= 2 ? 'done' : 'building'
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status }) })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: SENSORS[0].name })
    await row.getByRole('button', { name: 'Rebuild' }).click()

    // Wait for build done state
    await expect(page.getByText('Firmware compiled successfully!')).toBeVisible({ timeout: 10000 })

    // Both actions should be visible
    await expect(page.getByRole('button', { name: 'Flash Sensor' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Provision' })).toBeVisible()
  })

  test('Provision section renders and shows Web Serial warning (no serial in CI)', async ({ page }) => {
    await mockSensorList(page)

    const uuid = SENSORS[0].uuid
    let polls = 0

    await page.route(`**/api/sensors/${uuid}/build-firmware`, (route) => {
      route.fulfill({ status: 202, contentType: 'application/json', body: JSON.stringify({ status: 'building' }) })
    })

    await page.route(`**/api/sensors/${uuid}/build-status`, (route) => {
      polls++
      const status = polls >= 2 ? 'done' : 'building'
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status }) })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: SENSORS[0].name })
    await row.getByRole('button', { name: 'Rebuild' }).click()

    await expect(page.getByText('Firmware compiled successfully!')).toBeVisible({ timeout: 10000 })

    // Open Provision panel
    await page.getByRole('button', { name: 'Provision' }).click()

    // In Playwright CI, navigator.serial is not available; the ProvisionStep shows only the warning card.
    await expect(page.getByText('Web Serial not supported')).toBeVisible()

    // Close dismisses the panel, resetting UI state
    await page.getByRole('button', { name: 'Close' }).click()
    await expect(page.getByText('Firmware compiled successfully!')).not.toBeVisible()
  })

  test('toggling between Flash and Provision hides the other section', async ({ page }) => {
    await mockSensorList(page)

    const uuid = SENSORS[0].uuid
    let polls = 0

    await page.route(`**/api/sensors/${uuid}/build-firmware`, (route) => {
      route.fulfill({ status: 202, contentType: 'application/json', body: JSON.stringify({ status: 'building' }) })
    })

    await page.route(`**/api/sensors/${uuid}/build-status`, (route) => {
      polls++
      const status = polls >= 2 ? 'done' : 'building'
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status }) })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: SENSORS[0].name })
    await row.getByRole('button', { name: 'Rebuild' }).click()
    await expect(page.getByText('Firmware compiled successfully!')).toBeVisible({ timeout: 10000 })

    // Show Flash
    await page.getByRole('button', { name: 'Flash Sensor' }).click()
    await expect(page.locator('esp-web-install-button')).toBeVisible()

    // Now switch to Provision
    await page.getByRole('button', { name: 'Close' }).click()
    await row.getByRole('button', { name: 'Rebuild' }).click()
    await expect(page.getByText('Firmware compiled successfully!')).toBeVisible({ timeout: 10000 })
    await page.getByRole('button', { name: 'Provision' }).click()

    // Flash section should not be visible in provision mode
    await expect(page.locator('esp-web-install-button')).toHaveCount(0)
  })
})
