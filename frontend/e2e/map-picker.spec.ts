import { test, expect, type Page } from '@playwright/test'

const DEFAULT_LAT = 49.143845257365456
const DEFAULT_LNG = 9.214797255696741

const MOCK_SENSOR = {
  uuid: 'maptest1',
  name: 'map-test',
  type: 'density',
  latitude: DEFAULT_LAT,
  longitude: DEFAULT_LNG,
  dev_eui: '',
  app_key: '',
  ttn_device_id: '',
  created_at: '2026-01-15T10:30:00Z',
}

/** Wait until Leaflet has finished mounting (it adds the leaflet-container class). */
async function waitForMap(page: Page) {
  await expect(page.locator('.leaflet-container')).toBeVisible()
}

/** Click a point on the map at fractional (fx, fy) offsets within the bounding box. */
async function clickMap(page: Page, fx: number, fy: number) {
  const bbox = await page.locator('.map-container').boundingBox()
  expect(bbox).not.toBeNull()
  await page.mouse.click(bbox!.x + bbox!.width * fx, bbox!.y + bbox!.height * fy)
}

test.describe('MapPicker', () => {
  test.beforeEach(async ({ page }) => {
    // Suppress manifest requests so esp-web-tools doesn't make real calls
    await page.route('**/api/sensors/*/manifest.json', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ name: 'test' }) }),
    )
    // Return 204 for OSM tile requests — tiles won't render but map events still work
    await page.route('**openstreetmap.org/**', (route) => route.fulfill({ status: 204, body: '' }))
  })

  // ── Rendering ──────────────────────────────────────────────────────────────

  test('map container is rendered on step 1', async ({ page }) => {
    await page.goto('/add')
    await expect(page.locator('.map-container')).toBeVisible()
  })

  test('Leaflet initialises and renders the default marker', async ({ page }) => {
    await page.goto('/add')
    await waitForMap(page)
    // The draggable marker icon is proof Leaflet fully initialised
    await expect(page.locator('.leaflet-marker-icon')).toBeVisible()
  })

  test('map is not shown on steps after step 1', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')
    await waitForMap(page)
    await expect(page.locator('.map-container')).toBeVisible()

    await page.getByLabel('Name').fill('map-test')
    await page.getByRole('button', { name: /Create Sensor/ }).click()

    // Step 2 — map should be gone
    await expect(page.getByText('Sensor Created')).toBeVisible()
    await expect(page.locator('.map-container')).not.toBeVisible()
  })

  // ── Default coordinates ───────────────────────────────────────────────────

  test('latitude and longitude inputs default to the configured Heilbronn location', async ({ page }) => {
    await page.goto('/add')

    const lat = Number(await page.getByLabel('Latitude').inputValue())
    const lng = Number(await page.getByLabel('Longitude').inputValue())

    expect(lat).toBeCloseTo(DEFAULT_LAT, 4)
    expect(lng).toBeCloseTo(DEFAULT_LNG, 4)
  })

  test('default coordinates are not 0, 0', async ({ page }) => {
    await page.goto('/add')

    const lat = Number(await page.getByLabel('Latitude').inputValue())
    const lng = Number(await page.getByLabel('Longitude').inputValue())

    expect(lat).not.toBe(0)
    expect(lng).not.toBe(0)
  })

  // ── Map click → inputs ────────────────────────────────────────────────────

  test('clicking on the map changes the Latitude and Longitude inputs', async ({ page }) => {
    await page.goto('/add')
    await waitForMap(page)

    const latBefore = await page.getByLabel('Latitude').inputValue()
    const lngBefore = await page.getByLabel('Longitude').inputValue()

    // Click well off-centre (bottom-left quadrant) to produce different coords
    await clickMap(page, 0.2, 0.8)

    const latAfter = await page.getByLabel('Latitude').inputValue()
    const lngAfter = await page.getByLabel('Longitude').inputValue()

    expect(latAfter).not.toBe(latBefore)
    expect(lngAfter).not.toBe(lngBefore)
  })

  test('coordinates produced by a map click are valid lat/lng numbers', async ({ page }) => {
    await page.goto('/add')
    await waitForMap(page)

    await clickMap(page, 0.7, 0.3)

    const lat = Number(await page.getByLabel('Latitude').inputValue())
    const lng = Number(await page.getByLabel('Longitude').inputValue())

    expect(Number.isFinite(lat)).toBe(true)
    expect(Number.isFinite(lng)).toBe(true)
    expect(lat).toBeGreaterThan(-90)
    expect(lat).toBeLessThan(90)
    expect(lng).toBeGreaterThan(-180)
    expect(lng).toBeLessThan(180)
  })

  test('clicking the map twice uses the most recent click position', async ({ page }) => {
    await page.goto('/add')
    await waitForMap(page)

    await clickMap(page, 0.2, 0.2)
    const latFirst = await page.getByLabel('Latitude').inputValue()

    await clickMap(page, 0.8, 0.8)
    const latSecond = await page.getByLabel('Latitude').inputValue()

    // The two clicks are in opposite quadrants → different latitudes
    expect(latSecond).not.toBe(latFirst)
  })

  // ── Input → map sync ──────────────────────────────────────────────────────

  test('typing a new Latitude value updates the input', async ({ page }) => {
    await page.goto('/add')

    const latInput = page.getByLabel('Latitude')
    await latInput.fill('48.5')
    await latInput.blur()

    await expect(latInput).toHaveValue('48.5')
  })

  test('typing a new Longitude value updates the input', async ({ page }) => {
    await page.goto('/add')

    const lngInput = page.getByLabel('Longitude')
    await lngInput.fill('10.123')
    await lngInput.blur()

    await expect(lngInput).toHaveValue('10.123')
  })

  test('clicking the map updates inputs even after typing coordinates manually', async ({ page }) => {
    await page.goto('/add')
    await waitForMap(page)

    // Type a specific latitude into the input
    await page.getByLabel('Latitude').fill('48.9')
    expect(await page.getByLabel('Latitude').inputValue()).toBe('48.9')

    // Now click somewhere off-center on the map — the map is panned to lat=48.9
    // so clicking below-left of center will produce a latitude < 48.9
    await clickMap(page, 0.2, 0.8)

    // The input must now reflect the click result, not the typed string '48.9'
    const latAfterClick = await page.getByLabel('Latitude').inputValue()
    expect(latAfterClick).not.toBe('48.9')
  })

  // ── API submission ────────────────────────────────────────────────────────

  test('map-click coordinates are sent to the API on create', async ({ page }) => {
    let postedLat: number | null = null
    let postedLng: number | null = null

    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        const body = JSON.parse(route.request().postData() ?? '{}')
        postedLat = body.latitude
        postedLng = body.longitude
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ ...MOCK_SENSOR, latitude: body.latitude, longitude: body.longitude }),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')
    await waitForMap(page)

    await clickMap(page, 0.7, 0.3)

    const expectedLat = Number(await page.getByLabel('Latitude').inputValue())
    const expectedLng = Number(await page.getByLabel('Longitude').inputValue())

    await page.getByLabel('Name').fill('map-test')
    await page.getByRole('button', { name: /Create Sensor/ }).click()
    await expect(page.getByText('Sensor Created')).toBeVisible()

    expect(postedLat).not.toBeNull()
    expect(postedLng).not.toBeNull()
    expect(postedLat!).toBeCloseTo(expectedLat, 5)
    expect(postedLng!).toBeCloseTo(expectedLng, 5)
  })

  test('manually typed coordinates are sent to the API when map is not clicked', async ({ page }) => {
    let postedLat: number | null = null
    let postedLng: number | null = null

    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        const body = JSON.parse(route.request().postData() ?? '{}')
        postedLat = body.latitude
        postedLng = body.longitude
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ ...MOCK_SENSOR, name: 'manual-test', latitude: body.latitude, longitude: body.longitude }),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')

    await page.getByLabel('Name').fill('manual-test')
    await page.getByLabel('Latitude').fill('48.123')
    await page.getByLabel('Longitude').fill('10.456')

    // Disable HTML5 min/max constraint validation so the form submits
    await page.locator('form').evaluate((form: HTMLFormElement) => { form.noValidate = true })
    await page.getByRole('button', { name: /Create Sensor/ }).click()

    await expect(page.getByText('Sensor Created')).toBeVisible()

    expect(postedLat).toBeCloseTo(48.123, 3)
    expect(postedLng).toBeCloseTo(10.456, 3)
  })

  // ── Locate me ─────────────────────────────────────────────────────────────

  test('locate me button is rendered on the map', async ({ page }) => {
    await page.goto('/add')
    await waitForMap(page)
    await expect(page.getByRole('button', { name: /current location/i })).toBeVisible()
  })

  test('clicking locate me button updates coordinates to GPS location', async ({ page, context }) => {
    const GPS_LAT = 48.1234
    const GPS_LNG = 11.5678

    await context.grantPermissions(['geolocation'])
    await context.setGeolocation({ latitude: GPS_LAT, longitude: GPS_LNG })

    await page.goto('/add')
    await waitForMap(page)

    const latBefore = await page.getByLabel('Latitude').inputValue()

    await page.getByRole('button', { name: /current location/i }).click()

    // Wait for the coordinate inputs to reflect the GPS position
    await expect(page.getByLabel('Latitude')).not.toHaveValue(latBefore)

    const lat = Number(await page.getByLabel('Latitude').inputValue())
    const lng = Number(await page.getByLabel('Longitude').inputValue())

    expect(lat).toBeCloseTo(GPS_LAT, 3)
    expect(lng).toBeCloseTo(GPS_LNG, 3)
  })

  test('default Heilbronn coordinates are sent when neither map nor inputs are changed', async ({ page }) => {
    let postedLat: number | null = null
    let postedLng: number | null = null

    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        const body = JSON.parse(route.request().postData() ?? '{}')
        postedLat = body.latitude
        postedLng = body.longitude
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_SENSOR),
        })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')
    await waitForMap(page)

    await page.getByLabel('Name').fill('map-test')
    await page.getByRole('button', { name: /Create Sensor/ }).click()
    await expect(page.getByText('Sensor Created')).toBeVisible()

    expect(postedLat).toBeCloseTo(DEFAULT_LAT, 4)
    expect(postedLng).toBeCloseTo(DEFAULT_LNG, 4)
  })
})
