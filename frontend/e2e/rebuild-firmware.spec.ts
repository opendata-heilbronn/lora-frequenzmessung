import { test, expect } from '@playwright/test'

const SENSORS = [
  {
    id: 1,
    uuid: 'aaaa-1111-2222-3333-444444444444',
    name: 'sensor-linked',
    latitude: 49.1438602,
    longitude: 9.2149624,
    type: 'pax',
    dev_eui: '0011223344556677',
    app_key: 'AABBCCDDEEFF00112233445566778899',
    ttn_device_id: 'sensor-linked',
    created_at: '2026-01-10T08:00:00Z',
  },
  {
    id: 2,
    uuid: 'bbbb-5555-6666-7777-888888888888',
    name: 'sensor-unlinked',
    latitude: 48.7758,
    longitude: 9.1829,
    type: 'pax',
    dev_eui: '',
    app_key: '',
    ttn_device_id: '',
    created_at: '2026-01-12T14:00:00Z',
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

test.describe('Rebuild Firmware', () => {
  test('rebuild button visible for each sensor', async ({ page }) => {
    await mockSensorList(page)
    await page.goto('/')

    for (const s of SENSORS) {
      const row = page.locator('tr', { hasText: s.name })
      await expect(row.getByRole('button', { name: 'Rebuild' })).toBeVisible()
    }
  })

  test('rebuild → build → flash flow', async ({ page }) => {
    await mockSensorList(page)

    const uuid = SENSORS[0].uuid
    let pollCount = 0

    await page.route(`**/api/sensors/${uuid}/build-firmware`, (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"building"}' })
    })
    await page.route(`**/api/sensors/${uuid}/build-status`, (route) => {
      pollCount++
      // First poll returns building, second returns done
      const status = pollCount >= 2 ? 'done' : 'building'
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status }) })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: SENSORS[0].name })
    await row.getByRole('button', { name: 'Rebuild' }).click()

    // Should show building state
    await expect(page.getByText('Downloading firmware…')).toBeVisible()

    // Wait for "done" state
    await expect(page.getByText('Firmware compiled successfully!')).toBeVisible({ timeout: 10000 })

    // Click Flash & Configure
    await page.getByRole('button', { name: 'Flash & Configure' }).click()

    // FlashAndProvisionStep should appear with flash button in idle phase
    await expect(page.locator('esp-web-install-button')).toBeVisible()

    // Close dismisses panel
    await page.getByRole('button', { name: 'Close' }).click()
    await expect(page.getByText('Firmware compiled successfully!')).not.toBeVisible()
  })

  test('rebuild error shows retry and close', async ({ page }) => {
    await mockSensorList(page)

    const uuid = SENSORS[0].uuid

    await page.route(`**/api/sensors/${uuid}/build-firmware`, (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"building"}' })
    })
    await page.route(`**/api/sensors/${uuid}/build-status`, (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'error', message: 'PlatformIO not found' }),
      })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: SENSORS[0].name })
    await row.getByRole('button', { name: 'Rebuild' }).click()

    // Wait for error state
    await expect(page.getByText('Build failed: PlatformIO not found')).toBeVisible({ timeout: 10000 })

    // Retry and Close buttons should be visible
    await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Close' })).toBeVisible()

    // Close dismisses panel
    await page.getByRole('button', { name: 'Close' }).click()
    await expect(page.getByText('Build failed')).not.toBeVisible()
  })

  test('close during build collapses panel', async ({ page }) => {
    await mockSensorList(page)

    const uuid = SENSORS[0].uuid

    await page.route(`**/api/sensors/${uuid}/build-firmware`, (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: '{"status":"building"}' })
    })
    await page.route(`**/api/sensors/${uuid}/build-status`, (route) => {
      // Always return building
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'building' }) })
    })

    await page.goto('/')

    const row = page.locator('tr', { hasText: SENSORS[0].name })
    await row.getByRole('button', { name: 'Rebuild' }).click()

    // Should show building state
    await expect(page.getByText('Downloading firmware…')).toBeVisible()

    // Close dismisses panel
    await page.getByRole('button', { name: 'Close' }).click()
    await expect(page.getByText('Downloading firmware…')).not.toBeVisible()
  })
})
