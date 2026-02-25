import { test, expect } from '@playwright/test'

// These tests hit the real backend — no mocked API responses.
// Requires: backend running on :3001, frontend dev server on :5173.
// Firmware builds take 30–60s, so we use generous timeouts.

test.describe('AddSensor live integration', () => {
  test.describe.configure({ mode: 'serial' })
  test.setTimeout(120_000)

  test('skip TTN: create → build without LoRa → flash → delete', async ({ page }) => {
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

    // Verify DevEUI and AppKey are shown
    const devEUI = await page.locator('.info-row').filter({ hasText: 'DevEUI' }).locator('code').textContent()
    expect(devEUI).toBeTruthy()
    expect(devEUI!.length).toBe(16) // 8 bytes hex

    const appKey = await page.locator('.info-row').filter({ hasText: 'AppKey' }).locator('code').textContent()
    expect(appKey).toBeTruthy()
    expect(appKey!.length).toBe(32) // 16 bytes hex

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
    ).toHaveText('Linked')

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
