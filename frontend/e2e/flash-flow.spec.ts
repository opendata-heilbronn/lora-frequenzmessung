import { test, expect } from '@playwright/test'

// This spec verifies the happy-path Add Sensor flow up to Flash/Provision
// using mocked backend endpoints. It ensures the UI renders the correct
// manifest URL for esp-web-tools and allows proceeding to the Provision step.

test.describe('Add Sensor → Build → Flash → Provision flow (mocked)', () => {
  test('shows manifest url and reaches Provision step', async ({ page }) => {
    const sensorUUID = 'abcd1234'
    const sensorName = 'density-01'

    // Ensure auth guard passes
    await page.addInitScript(() => {
      localStorage.setItem('token', 'test-token')
    })

    // Mock API routes
    await page.route('**/api/sensors', async (route) => {
      if (route.request().method() === 'POST') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ uuid: sensorUUID, name: sensorName, type: 'pax' }),
        })
      }
      return route.fallback()
    })

    await page.route(`**/api/sensors/${sensorUUID}/register-ttn`, async (route) => {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          dev_eui: 'A1B2C3D4E5F60789',
          app_key: '00112233445566778899AABBCCDDEEFF',
          ttn_device_id: `test-${sensorUUID}`,
        }),
      })
    })

    await page.route(`**/api/sensors/${sensorUUID}/build-firmware`, async (route) => {
      return route.fulfill({
        status: 202,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'building' }),
      })
    })

    // Return done status immediately so polling stops quickly
    await page.route(`**/api/sensors/${sensorUUID}/build-status`, async (route) => {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'done' }),
      })
    })

    // Navigate to Add Sensor view
    await page.goto('/add')

    // Fill minimal required fields
    await page.getByLabel('Name').fill(sensorName)

    // Submit create sensor
    await page.getByRole('button', { name: 'Create Sensor →' }).click()

    // Step 2 should show UUID and name
    await expect(page.locator('dt', { hasText: 'UUID' }).locator('..')).toContainText(sensorUUID)
    await expect(page.locator('dt', { hasText: 'Name' }).locator('..')).toContainText(sensorName)

    // Proceed to TTN registration
    await page.getByRole('button', { name: 'Register with TTN →' }).click()

    // Wait for success card then click Build Firmware
    await expect(page.getByText('Device registered with TTN!')).toBeVisible()
    await page.getByRole('button', { name: 'Build Firmware →' }).click()

    // Build phase completes (polling sees done)
    await expect(page.getByText('Firmware compiled successfully!')).toBeVisible()

    // Move to Flash step
    await page.getByRole('button', { name: 'Flash Sensor →' }).click()

    // Assert the manifest URL on the custom element
    const manifestAttr = await page.locator('esp-web-install-button').getAttribute('manifest')
    expect(manifestAttr).toBe(`/api/sensors/${sensorUUID}/manifest.json`)

    // Proceed to Provision step
    await page.getByRole('button', { name: 'Provision →' }).click()

    // Verify Provision UI is rendered with read-only values
    await expect(page.getByText('Provision Device (NVS)')).toBeVisible()
    await expect(page.getByText('Sensor ID')).toBeVisible()
    await expect(page.locator('code', { hasText: sensorUUID })).toBeVisible()
  })
})
