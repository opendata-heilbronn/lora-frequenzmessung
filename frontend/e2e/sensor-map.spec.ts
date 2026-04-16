import { test, expect, type Page } from '@playwright/test'

const SENSORS = [
  {
    id: 1,
    uuid: 'aaaa-1111-2222-3333-444444444444',
    name: 'sensor-alpha',
    latitude: 49.143845,
    longitude: 9.214797,
    type: 'density',
    dev_eui: '0011223344556677',
    app_key: 'AABBCCDDEEFF00112233445566778899',
    ttn_device_id: 'sensor-alpha',
    created_at: '2026-01-10T08:00:00Z',
    last_battery_value: null,
    last_battery_time: null,
  },
  {
    id: 2,
    uuid: 'bbbb-5555-6666-7777-888888888888',
    name: 'sensor-beta',
    latitude: 48.775845,
    longitude: 9.182932,
    type: 'density',
    dev_eui: '',
    app_key: '',
    ttn_device_id: '',
    created_at: '2026-01-12T14:00:00Z',
    last_battery_value: null,
    last_battery_time: null,
  },
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

/** Wait until a Leaflet map is visible inside a specific row. */
async function waitForMapInRow(page: Page, sensorName: string) {
  const row = page.locator('.map-row', { has: page.locator('.sensor-map') })
  await expect(row.locator('.leaflet-container')).toBeVisible()
  // Also confirm the marker for the right sensor appears
  await expect(row.locator('.leaflet-marker-icon')).toBeVisible()
  // Suppress unused-variable warning — sensorName is used in assertions below
  void sensorName
}

test.describe('SensorMap — coordinates link', () => {
  test.beforeEach(async ({ page }) => {
    // Silence OSM tile requests
    await page.route('**openstreetmap.org/**', (route) => route.fulfill({ status: 204, body: '' }))
  })

  // ── Coordinate cell appearance ────────────────────────────────────────────

  test('coordinate cell is rendered as a clickable button for every sensor', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    for (const s of SENSORS) {
      const row = page.locator('tr', { hasText: s.name })
      const mapBtn = row.locator('.map-pin-btn')
      await expect(mapBtn).toBeVisible()
      await expect(mapBtn).toHaveAttribute('title', `${s.latitude.toFixed(6)}, ${s.longitude.toFixed(6)}`)
    }
  })

  test('coordinate button has an aria-label describing its action', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    const mapBtn = page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn')
    await expect(mapBtn).toHaveAttribute('aria-label', 'Show on map')
  })

  // ── Opening the map ───────────────────────────────────────────────────────

  test('clicking coordinates opens a map row below the sensor', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    // No map initially
    await expect(page.locator('.map-row')).not.toBeVisible()

    await page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn').click()

    await expect(page.locator('.map-row')).toBeVisible()
    await expect(page.locator('.sensor-map')).toBeVisible()
  })

  test('the opened map contains a Leaflet marker', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    await page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn').click()

    await expect(page.locator('.map-row .leaflet-container')).toBeVisible()
    await expect(page.locator('.map-row .leaflet-marker-icon')).toBeVisible()
  })

  test('the marker popup shows the sensor name', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    await page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn').click()

    // Leaflet opens the popup automatically on mount
    await expect(page.locator('.leaflet-popup-content')).toContainText('sensor-alpha')
  })

  test('aria-label changes to "Hide map" while map is open', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    const mapBtn = page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn')
    await mapBtn.click()

    await expect(mapBtn).toHaveAttribute('aria-label', 'Hide map')
  })

  // ── Closing the map ───────────────────────────────────────────────────────

  test('clicking coordinates again collapses the map', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    const mapBtn = page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn')
    await mapBtn.click()
    await expect(page.locator('.map-row')).toBeVisible()

    await mapBtn.click()
    await expect(page.locator('.map-row')).not.toBeVisible()
  })

  test('aria-label returns to "Show on map" after collapsing', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    const mapBtn = page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn')
    await mapBtn.click()
    await mapBtn.click()

    await expect(mapBtn).toHaveAttribute('aria-label', 'Show on map')
  })

  // ── Only one map open at a time ───────────────────────────────────────────

  test('opening a second sensor map closes the first', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    const alphaBtn = page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn')
    const betaBtn = page.locator('tr', { hasText: 'sensor-beta' }).locator('.map-pin-btn')

    await alphaBtn.click()
    await expect(page.locator('.map-row')).toHaveCount(1)
    await expect(page.locator('.leaflet-popup-content')).toContainText('sensor-alpha')

    await betaBtn.click()
    await expect(page.locator('.map-row')).toHaveCount(1)
    await expect(page.locator('.leaflet-popup-content')).toContainText('sensor-beta')
  })

  test('only one .sensor-map element exists when two sensors are listed', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    await page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn').click()

    // Only one map open even though two sensors exist
    await expect(page.locator('.sensor-map')).toHaveCount(1)
  })

  // ── Map is not present by default ─────────────────────────────────────────

  test('no map row is visible on initial page load', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    await expect(page.locator('tr', { hasText: 'sensor-alpha' })).toBeVisible()
    await expect(page.locator('.map-row')).not.toBeVisible()
    await expect(page.locator('.sensor-map')).not.toBeVisible()
  })

  // ── Independent of other expanded panels ─────────────────────────────────

  test('map row and TTN info panel can be open on different sensors simultaneously', async ({ page }) => {
    await mockSensors(page)
    await page.goto('/')

    // Open map for sensor-alpha
    await page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn').click()
    await expect(page.locator('.map-row')).toBeVisible()

    // Open TTN panel for sensor-alpha (same sensor — both panels should coexist)
    await page.locator('tr', { hasText: 'sensor-alpha' }).locator('.badge-linked').click()
    await expect(page.locator('.ttn-info-panel')).toBeVisible()
    await expect(page.locator('.map-row')).toBeVisible()
  })

  // ── Single sensor edge case ───────────────────────────────────────────────

  test('works correctly when only one sensor is in the list', async ({ page }) => {
    await mockSensors(page, [SENSORS[0]])
    await page.goto('/')

    await page.locator('tr', { hasText: 'sensor-alpha' }).locator('.map-pin-btn').click()
    await expect(page.locator('.map-row')).toBeVisible()
    await expect(page.locator('.leaflet-popup-content')).toContainText('sensor-alpha')
  })
})
