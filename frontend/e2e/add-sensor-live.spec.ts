import { test, expect } from '@playwright/test'

// These tests hit the real backend — no mocked API responses.
// Requires: ManagementAPI running on :3002, backend on :3001, frontend dev server on :5173.
// Firmware builds take 30–60s, so we use generous timeouts.

let authToken = ''

test.describe('AddSensor live integration', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(120_000)

  test.beforeAll(async ({ request }) => {
    const resp = await request.post('/auth/login', {
      data: { username: 'admin', password: 'admin' },
    })
    expect(resp.ok()).toBeTruthy()
    authToken = (await resp.json()).token

    // Clean up any leftover sensors from previous failed runs
    const listResp = await request.get('/api/sensors', {
      headers: { Authorization: `Bearer ${authToken}` },
    })
    if (listResp.ok()) {
      const sensors: { uuid: string; name: string }[] = await listResp.json()
      for (const s of sensors) {
        if (s.name.startsWith('live-test-')) {
          await request.delete(`/api/sensors/${s.uuid}`, {
            headers: { Authorization: `Bearer ${authToken}` },
          })
        }
      }
    }
  })

  test.beforeEach(async ({ page }) => {
    // Inject the JWT before the app scripts run so the router guard passes
    await page.addInitScript((token: string) => {
      window.localStorage.setItem('token', token)
    }, authToken)
  })

  test.skip('skip TTN: create → build without LoRa → flash → delete', async ({ page }) => {
    await page.goto('/add')

    // Step 1: fill sensor details
    await page.getByLabel('Name').fill('live-test-no-ttn')
    await page.getByLabel('Latitude').fill('49.14')
    await page.getByLabel('Longitude').fill('9.21')
    await page.getByRole('button', { name: /Create Sensor/ }).click()

    // Step 2: sensor created
    await expect(page.getByText('Sensor Created')).toBeVisible()
    const sensorUUID = await page.locator('.info-row').filter({ hasText: 'UUID' }).locator('code').textContent()
    expect(sensorUUID).toBeTruthy()

    // Skip TTN
    await page.getByRole('button', { name: /Skip TTN/ }).click()

    // Step 4: build — should show "without LoRa" notice
    await expect(page.getByText('without LoRa')).toBeVisible()
    await expect(page.getByText('Compiling firmware with PlatformIO')).toBeVisible()

    // Wait for build to complete (up to 90s)
    await expect(page.getByText('Firmware compiled successfully')).toBeVisible({ timeout: 90_000 })

    // Step 5: flash
    await page.getByRole('button', { name: /Flash Sensor/ }).click()
    await expect(page.getByText('Flash Sensor via USB')).toBeVisible()

    // Cleanup: go back to sensor list, delete the sensor
    await page.getByRole('link', { name: /Back to sensor list/ }).click()
    await expect(page.locator('tr', { hasText: 'live-test-no-ttn' })).toBeVisible()

    page.on('dialog', (dialog) => dialog.accept())
    await page.locator('tr', { hasText: 'live-test-no-ttn' }).getByRole('button', { name: 'Delete' }).click()
    await expect(page.locator('tr', { hasText: 'live-test-no-ttn' })).not.toBeVisible()
  })

  test('with TTN: create → register TTN → build with LoRa → flash → delete', async ({ page }) => {
    await page.goto('/add')

    // Step 1: fill sensor details
    await page.getByLabel('Name').fill('live-test-with-ttn')
    await page.getByLabel('Latitude').fill('48.77')
    await page.getByLabel('Longitude').fill('9.18')
    await page.getByRole('button', { name: /Create Sensor/ }).click()

    // Step 2: sensor created
    await expect(page.getByText('Sensor Created')).toBeVisible()

    // Register with TTN
    await page.getByRole('button', { name: /Register with TTN/ }).click()

    // Step 3: TTN registration — wait for it to complete
    await expect(page.getByText('Device registered with TTN')).toBeVisible({ timeout: 30_000 })

    // Verify DevEUI and AppKey are shown (displayed masked: XXXX****XXXX)
    const devEUI = await page.locator('.info-row').filter({ hasText: 'DevEUI' }).locator('code').textContent()
    expect(devEUI).toMatch(/^[0-9A-F]{4}\*{4}[0-9A-F]{4}$/) // 8-byte hex, masked

    const appKey = await page.locator('.info-row').filter({ hasText: 'AppKey' }).locator('code').textContent()
    expect(appKey).toMatch(/^[0-9A-F]{4}\*{4}[0-9A-F]{4}$/) // 16-byte hex, masked

    // Build firmware
    await page.getByRole('button', { name: /Build Firmware/ }).click()

    // Step 4: should NOT show "without LoRa" notice
    await expect(page.getByText('without LoRa')).not.toBeVisible()
    await expect(page.getByText('Compiling firmware with PlatformIO')).toBeVisible()

    // Wait for build to complete (up to 90s)
    await expect(page.getByText('Firmware compiled successfully')).toBeVisible({ timeout: 90_000 })

    // Step 5: flash
    await page.getByRole('button', { name: /Flash Sensor/ }).click()
    await expect(page.getByText('Flash Sensor via USB')).toBeVisible()

    // Cleanup: go back to sensor list, delete the sensor (confirm should mention TTN)
    await page.getByRole('link', { name: /Back to sensor list/ }).click()
    await expect(page.locator('tr', { hasText: 'live-test-with-ttn' })).toBeVisible()

    // Verify TTN badge shows "Linked"
    await expect(
      page.locator('tr', { hasText: 'live-test-with-ttn' }).locator('.badge-linked')
    ).toContainText('Linked')

    let dialogMessage = ''
    page.on('dialog', async (dialog) => {
      dialogMessage = dialog.message()
      await dialog.accept()
    })
    await page.locator('tr', { hasText: 'live-test-with-ttn' }).getByRole('button', { name: 'Delete' }).click()
    await expect(page.locator('tr', { hasText: 'live-test-with-ttn' })).not.toBeVisible()

    // Delete confirm should mention TTN removal
    expect(dialogMessage).toContain('also remove it from TTN')
  })
})
