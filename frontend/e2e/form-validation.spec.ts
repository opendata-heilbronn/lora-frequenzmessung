import { test, expect } from '@playwright/test'

const MOCK_SENSOR = {
  uuid: 'aabbccdd',
  name: 'boundary-sensor',
  type: 'density',
  latitude: 90,
  longitude: 180,
  dev_eui: '',
  app_key: '',
  ttn_device_id: '',
  created_at: '2026-01-15T10:30:00Z',
}

test.describe('AddSensor form validation', () => {
  test.beforeEach(async ({ page }) => {
    // Block manifest requests so esp-web-tools does not make real calls
    await page.route('**/api/sensors/*/manifest.json', (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ name: 'test' }) })
    })
  })

  test('empty name (whitespace-only) shows validation error without calling API', async ({ page }) => {
    let postCalled = false
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        postCalled = true
        route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')

    // Fill name with spaces only — browser "required" passes, but JS trim() catches it
    await page.getByLabel('Name').fill('   ')
    await page.getByLabel('Latitude').fill('49')
    await page.getByLabel('Longitude').fill('9')
    await page.getByRole('button', { name: /Create Sensor/ }).click()

    await expect(page.locator('.error-msg')).toContainText('Name is required')
    // Still on step 1
    await expect(page.getByText('Sensor Details')).toBeVisible()
    expect(postCalled).toBe(false)
  })

  test('name longer than 64 characters shows validation error without calling API', async ({ page }) => {
    let postCalled = false
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        postCalled = true
        route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')

    // Remove maxlength so we can fill > 64 chars, then trigger Vue reactivity
    await page.locator('input[type="text"]').evaluate((el: HTMLInputElement) => {
      el.removeAttribute('maxlength')
    })
    await page.getByLabel('Name').fill('a'.repeat(65))
    await page.getByLabel('Latitude').fill('49')
    await page.getByLabel('Longitude').fill('9')
    await page.getByRole('button', { name: /Create Sensor/ }).click()

    await expect(page.locator('.error-msg')).toContainText('Name is required and must be 1-64 characters')
    await expect(page.getByText('Sensor Details')).toBeVisible()
    expect(postCalled).toBe(false)
  })

  test('latitude out of range shows validation error without calling API', async ({ page }) => {
    let postCalled = false
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        postCalled = true
        route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')

    // Disable HTML5 constraint validation so the form submits and JS validation runs
    await page.locator('form').evaluate((form: HTMLFormElement) => {
      form.noValidate = true
    })

    await page.getByLabel('Name').fill('valid-sensor')
    await page.getByLabel('Latitude').fill('91')
    await page.getByLabel('Longitude').fill('9')
    await page.getByRole('button', { name: /Create Sensor/ }).click()

    await expect(page.locator('.error-msg')).toContainText('Latitude must be between -90 and 90')
    await expect(page.getByText('Sensor Details')).toBeVisible()
    expect(postCalled).toBe(false)
  })

  test('longitude out of range shows validation error without calling API', async ({ page }) => {
    let postCalled = false
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        postCalled = true
        route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')

    await page.locator('form').evaluate((form: HTMLFormElement) => {
      form.noValidate = true
    })

    await page.getByLabel('Name').fill('valid-sensor')
    await page.getByLabel('Latitude').fill('49')
    await page.getByLabel('Longitude').fill('-181')
    await page.getByRole('button', { name: /Create Sensor/ }).click()

    await expect(page.locator('.error-msg')).toContainText('Longitude must be between -180 and 180')
    await expect(page.getByText('Sensor Details')).toBeVisible()
    expect(postCalled).toBe(false)
  })

  test('boundary values (lat=90, lon=180, name=64 chars) are accepted and advance to step 2', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(MOCK_SENSOR) })
      } else {
        route.continue()
      }
    })

    await page.goto('/add')

    // Disable HTML5 min/max validation so boundary values are accepted by the browser
    await page.locator('form').evaluate((form: HTMLFormElement) => {
      form.noValidate = true
    })

    // 64-char name is within maxlength="64", so no maxlength removal needed
    await page.getByLabel('Name').fill('x'.repeat(64))
    await page.getByLabel('Latitude').fill('90')
    await page.getByLabel('Longitude').fill('180')
    await page.getByRole('button', { name: /Create Sensor/ }).click()

    // Should advance to step 2
    await expect(page.getByText('Sensor Created')).toBeVisible()
    await expect(page.locator('.error-msg')).not.toBeVisible()
  })
})
