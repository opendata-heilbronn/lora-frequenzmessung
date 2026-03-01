import { test, expect } from '@playwright/test'

// This spec verifies the happy-path Add Sensor flow up to the Flash & Provision step
// using mocked backend endpoints. It ensures the UI renders the correct
// manifest URL for esp-web-tools within the unified FlashAndProvisionStep component.

test.describe('Add Sensor → Build → Flash & Provision flow (mocked)', () => {
  test('shows manifest url in step 5 and has no separate Provision step', async ({ page }) => {
    const sensorUUID = 'abcd1234'
    const sensorName = 'density-01'

    // Ensure auth guard passes; also suppress Web Serial API so the
    // ProvisionStep reliably shows the "not supported" warning in headless Chromium.
    await page.addInitScript(() => {
      localStorage.setItem('token', 'test-token')
      try {
        Object.defineProperty(Navigator.prototype, 'serial', { get: () => undefined, configurable: true })
      } catch {
        // fallback: property already non-configurable — skip
      }
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

    // Submit — TTN + build chain run automatically
    await page.getByRole('button', { name: 'Create Sensor' }).click()

    // Setup stage shows sensor UUID
    await expect(page.getByText(sensorUUID)).toBeVisible()

    // Flash Sensor button appears once TTN + build complete (no manual button clicks needed)
    await expect(page.getByRole('button', { name: /Flash Sensor/ })).toBeVisible({ timeout: 10000 })
    await page.getByRole('button', { name: /Flash Sensor/ }).click()

    // Assert the manifest URL on the custom element (in FlashAndProvisionStep, idle phase)
    const manifestAttr = await page.locator('esp-web-install-button').getAttribute('manifest')
    expect(manifestAttr).toBe(`/api/sensors/${sensorUUID}/manifest.json`)

    // Verify there is no separate Provision step button — flashing auto-triggers provision
    await expect(page.getByRole('button', { name: /^Provision$/ })).not.toBeVisible()

  })
})
