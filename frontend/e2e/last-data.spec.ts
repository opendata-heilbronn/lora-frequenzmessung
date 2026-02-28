import { test, expect, type Page } from '@playwright/test'

// Compute timestamps at module load so every test uses the same "now".
const NOW = Date.now()
function daysAgo(d: number): string { return new Date(NOW - d * 86_400_000).toISOString() }
function hoursAgo(h: number): string { return new Date(NOW - h * 3_600_000).toISOString() }

const BASE_SENSOR = {
  id: 1,
  uuid: 'aaaa-1111-2222-3333-444444444444',
  name: 'sensor-alpha',
  latitude: 49.143845,
  longitude: 9.214797,
  type: 'density',
  dev_eui: '',
  app_key: '',
  ttn_device_id: '',
  created_at: '2026-01-10T08:00:00Z',
  last_battery_value: null,
  last_battery_time: null,
}

/** Four representative sensors — one per freshness state. */
const SENSORS = [
  { ...BASE_SENSOR, id: 1, uuid: 'fresh-111', name: 'sensor-fresh', last_data_time: hoursAgo(2) },
  { ...BASE_SENSOR, id: 2, uuid: 'stale-222', name: 'sensor-stale', last_data_time: daysAgo(3) },
  { ...BASE_SENSOR, id: 3, uuid: 'old-33333', name: 'sensor-old',   last_data_time: daysAgo(10) },
  { ...BASE_SENSOR, id: 4, uuid: 'never-444', name: 'sensor-never', last_data_time: null },
]

async function mockSensors(page: Page, sensors = SENSORS) {
  await page.route('**/api/sensors', (route) => {
    if (route.request().method() === 'GET') {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(sensors) })
    } else {
      route.continue()
    }
  })
}

test.describe('Last Data column', () => {

  // ── Column presence ────────────────────────────────────────────────────────

  test('Last Data column header is visible in the sensor table', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    await expect(page.locator('th', { hasText: 'Last Data' })).toBeVisible()
  })

  test('every sensor row has a data-badge cell', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    for (const s of SENSORS) {
      const row = page.locator('tr', { hasText: s.name })
      await expect(row.locator('.data-badge')).toBeVisible()
    }
  })

  // ── Colour classes ─────────────────────────────────────────────────────────

  test('sensor with data 2 hours ago gets data-fresh class (green)', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    const badge = page.locator('tr', { hasText: 'sensor-fresh' }).locator('.data-badge')
    await expect(badge).toHaveClass(/data-fresh/)
    await expect(badge).not.toHaveClass(/data-stale|data-old|data-never/)
  })

  test('sensor with data 3 days ago gets data-stale class (yellow)', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    const badge = page.locator('tr', { hasText: 'sensor-stale' }).locator('.data-badge')
    await expect(badge).toHaveClass(/data-stale/)
    await expect(badge).not.toHaveClass(/data-fresh|data-old|data-never/)
  })

  test('sensor with data 10 days ago gets data-old class (red)', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    const badge = page.locator('tr', { hasText: 'sensor-old' }).locator('.data-badge')
    await expect(badge).toHaveClass(/data-old/)
    await expect(badge).not.toHaveClass(/data-fresh|data-stale|data-never/)
  })

  test('sensor with no data gets data-never class (black)', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    const badge = page.locator('tr', { hasText: 'sensor-never' }).locator('.data-badge')
    await expect(badge).toHaveClass(/data-never/)
    await expect(badge).not.toHaveClass(/data-fresh|data-stale|data-old/)
  })

  // ── Boundary: exactly at the 24-hour edge ─────────────────────────────────

  test('sensor with data just under 24h ago is fresh', async ({ page }) => {
    await mockSensors(page, [{ ...BASE_SENSOR, uuid: 'edge-111', name: 'sensor-edge', last_data_time: hoursAgo(23) }])
    await page.goto('/')
    await expect(page.locator('tr', { hasText: 'sensor-edge' }).locator('.data-badge')).toHaveClass(/data-fresh/)
  })

  test('sensor with data just over 24h ago is stale', async ({ page }) => {
    await mockSensors(page, [{ ...BASE_SENSOR, uuid: 'edge-222', name: 'sensor-edge', last_data_time: hoursAgo(25) }])
    await page.goto('/')
    await expect(page.locator('tr', { hasText: 'sensor-edge' }).locator('.data-badge')).toHaveClass(/data-stale/)
  })

  // ── Boundary: exactly at the 7-day edge ───────────────────────────────────

  test('sensor with data just under 7 days ago is stale', async ({ page }) => {
    await mockSensors(page, [{ ...BASE_SENSOR, uuid: 'edge-333', name: 'sensor-edge', last_data_time: daysAgo(6) }])
    await page.goto('/')
    await expect(page.locator('tr', { hasText: 'sensor-edge' }).locator('.data-badge')).toHaveClass(/data-stale/)
  })

  test('sensor with data just over 7 days ago is old', async ({ page }) => {
    await mockSensors(page, [{ ...BASE_SENSOR, uuid: 'edge-444', name: 'sensor-edge', last_data_time: daysAgo(8) }])
    await page.goto('/')
    await expect(page.locator('tr', { hasText: 'sensor-edge' }).locator('.data-badge')).toHaveClass(/data-old/)
  })

  // ── Label text ─────────────────────────────────────────────────────────────

  test('never sensor shows "Never" text', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    await expect(page.locator('tr', { hasText: 'sensor-never' }).locator('.data-badge')).toHaveText('Never')
  })

  test('fresh sensor shows hours-ago label', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    const badge = page.locator('tr', { hasText: 'sensor-fresh' }).locator('.data-badge')
    // 2 h ago → "2h ago"
    await expect(badge).toContainText('h ago')
    await expect(badge).not.toContainText('d ago')
  })

  test('stale sensor shows days-ago label', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    const badge = page.locator('tr', { hasText: 'sensor-stale' }).locator('.data-badge')
    // 3 days ago → "3d ago"
    await expect(badge).toContainText('d ago')
  })

  test('old sensor shows days-ago label', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    const badge = page.locator('tr', { hasText: 'sensor-old' }).locator('.data-badge')
    // 10 days ago → "10d ago"
    await expect(badge).toContainText('d ago')
  })

  // ── All four states visible simultaneously ─────────────────────────────────

  test('all four freshness states can appear in the same table', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    await expect(page.locator('.data-fresh')).toBeVisible()
    await expect(page.locator('.data-stale')).toBeVisible()
    await expect(page.locator('.data-old')).toBeVisible()
    await expect(page.locator('.data-never')).toBeVisible()
  })

  // ── Does not affect other columns ─────────────────────────────────────────

  test('Last Data column does not replace the Battery column', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')
    await expect(page.locator('th', { hasText: 'Battery' })).toBeVisible()
    await expect(page.locator('th', { hasText: 'Last Data' })).toBeVisible()
  })
})
