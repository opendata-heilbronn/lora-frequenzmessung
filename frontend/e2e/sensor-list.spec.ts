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
    last_battery_value: 82.5,
    last_battery_time: '2026-02-18T09:00:00Z',
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
    last_battery_value: null,
    last_battery_time: null,
  },
]

const MOCK_TTN = {
  dev_eui: 'FF11223344556677',
  app_key: 'FF00112233445566778899AABBCCDDEE',
  ttn_device_id: 'sensor-unlinked',
}

test.describe('SensorList', () => {
  test('shows mixed TTN status badges', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    // Linked sensor shows "Linked" badge (with toggle arrow)
    const linkedRow = page.locator('tr', { hasText: 'sensor-linked' })
    await expect(linkedRow.locator('.badge-linked')).toContainText('Linked')

    // Unlinked sensor shows "Not linked" badge and "Link TTN" button
    const unlinkedRow = page.locator('tr', { hasText: 'sensor-unlinked' })
    await expect(unlinkedRow.locator('.badge-unlinked')).toHaveText('Not linked')
    await expect(unlinkedRow.getByRole('button', { name: 'Link TTN' })).toBeVisible()
  })

  test('link TTN inline updates badge', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
      } else {
        route.continue()
      }
    })
    await page.route('**/api/sensors/bbbb-5555-6666-7777-888888888888/register-ttn', (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_TTN) })
    })

    await page.goto('/')

    const unlinkedRow = page.locator('tr', { hasText: 'sensor-unlinked' })
    await unlinkedRow.getByRole('button', { name: 'Link TTN' }).click()

    // Badge should change to "Linked" (with toggle arrow)
    await expect(unlinkedRow.locator('.badge-linked')).toContainText('Linked')
    // "Link TTN" button should be gone
    await expect(unlinkedRow.getByRole('button', { name: 'Link TTN' })).not.toBeVisible()
  })

  test('delete shows inline confirmation and removes row on confirm', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
      } else {
        route.continue()
      }
    })
    await page.route('**/api/sensors/aaaa-1111-2222-3333-444444444444', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    const linkedRow = page.locator('tr', { hasText: 'sensor-linked' })
    await linkedRow.getByRole('button', { name: 'Delete' }).click()

    // Inline confirmation should appear
    await expect(linkedRow.getByText('Delete?')).toBeVisible()
    await expect(linkedRow.getByRole('button', { name: 'Yes' })).toBeVisible()
    await expect(linkedRow.getByRole('button', { name: 'Cancel' })).toBeVisible()

    // Confirm deletion
    await linkedRow.getByRole('button', { name: 'Yes' }).click()
    await expect(linkedRow).not.toBeVisible()
  })

  test('delete cancel keeps the row visible', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    const unlinkedRow = page.locator('tr', { hasText: 'sensor-unlinked' })
    await unlinkedRow.getByRole('button', { name: 'Delete' }).click()

    // Cancel — row should remain
    await unlinkedRow.getByRole('button', { name: 'Cancel' }).click()
    await expect(unlinkedRow).toBeVisible()
    await expect(unlinkedRow.getByText('Delete?')).not.toBeVisible()
  })

  test('clicking Linked badge expands TTN info panel', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    const linkedRow = page.locator('tr', { hasText: 'sensor-linked' })

    // Panel is not visible initially
    await expect(page.locator('.ttn-info-panel')).not.toBeVisible()

    // Click "Linked" badge
    await linkedRow.locator('.badge-linked').click()

    // Panel shows all three TTN fields
    const panel = page.locator('.ttn-info-panel')
    await expect(panel).toBeVisible()
    await expect(panel).toContainText('sensor-linked')
    await expect(panel).toContainText('0011223344556677')
    await expect(panel).toContainText('AABBCCDDEEFF00112233445566778899')

    // Badge now shows ▲ (collapsed indicator)
    await expect(linkedRow.locator('.badge-linked')).toContainText('▲')
  })

  test('clicking Linked badge again collapses the TTN info panel', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    const linkedRow = page.locator('tr', { hasText: 'sensor-linked' })
    const badge = linkedRow.locator('.badge-linked')

    await badge.click()
    await expect(page.locator('.ttn-info-panel')).toBeVisible()

    // Click again to collapse
    await badge.click()
    await expect(page.locator('.ttn-info-panel')).not.toBeVisible()

    // Badge shows ▼ again
    await expect(badge).toContainText('▼')
  })

  test('only one TTN info panel open at a time', async ({ page }) => {
    const twoLinkedSensors = [
      SENSORS[0],
      {
        ...SENSORS[1],
        dev_eui: 'FF11223344556677',
        app_key: 'FF00112233445566778899AABBCCDDEE',
        ttn_device_id: 'sensor-second',
        name: 'sensor-second',
      },
    ]

    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(twoLinkedSensors) })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    const firstRow = page.locator('tr', { hasText: 'sensor-linked' })
    const secondRow = page.locator('tr', { hasText: 'sensor-second' })

    // Open first
    await firstRow.locator('.badge-linked').click()
    await expect(page.locator('.ttn-info-panel')).toHaveCount(1)
    await expect(page.locator('.ttn-info-panel')).toContainText('sensor-linked')

    // Open second — first should close
    await secondRow.locator('.badge-linked').click()
    await expect(page.locator('.ttn-info-panel')).toHaveCount(1)
    await expect(page.locator('.ttn-info-panel')).toContainText('sensor-second')
  })

  test('battery column shows percentage for sensor with battery data', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    // Sensor with battery data shows rounded percentage
    const linkedRow = page.locator('tr', { hasText: 'sensor-linked' })
    const batteryBadge = linkedRow.locator('.battery-badge')
    await expect(batteryBadge).toBeVisible()
    await expect(batteryBadge).toHaveText('83%')
  })

  test('battery column shows dash for sensor without battery data', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    // Sensor without battery data shows em-dash placeholder
    const unlinkedRow = page.locator('tr', { hasText: 'sensor-unlinked' })
    await expect(unlinkedRow.locator('.battery-unknown')).toHaveText('—')
    await expect(unlinkedRow.locator('.battery-badge')).not.toBeVisible()
  })

  test('battery badge uses correct colour class based on percentage', async ({ page }) => {
    const sensors = [
      { ...SENSORS[0], name: 'high-bat',  last_battery_value: 75.0, last_battery_time: '2026-02-18T10:00:00Z' },
      { ...SENSORS[0], id: 2, uuid: 'cccc', name: 'mid-bat',   last_battery_value: 40.0, last_battery_time: '2026-02-18T10:00:00Z' },
      { ...SENSORS[0], id: 3, uuid: 'dddd', name: 'low-bat',   last_battery_value: 15.0, last_battery_time: '2026-02-18T10:00:00Z' },
    ]

    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(sensors) })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    await expect(page.locator('tr', { hasText: 'high-bat' }).locator('.battery-high')).toBeVisible()
    await expect(page.locator('tr', { hasText: 'mid-bat'  }).locator('.battery-mid')).toBeVisible()
    await expect(page.locator('tr', { hasText: 'low-bat'  }).locator('.battery-low')).toBeVisible()
  })

  test('refresh button re-fetches sensor list', async ({ page }) => {
    let callCount = 0
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        callCount++
        // First load (onMounted): only the linked sensor; after refresh: both sensors
        const body = callCount === 1 ? JSON.stringify([SENSORS[0]]) : JSON.stringify(SENSORS)
        route.fulfill({ status: 200, contentType: 'application/json', body })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    // Initial load shows only the first sensor
    await expect(page.locator('tr', { hasText: 'sensor-linked' })).toBeVisible()
    await expect(page.locator('tr', { hasText: 'sensor-unlinked' })).not.toBeVisible()

    // Click refresh — second GET returns both sensors
    await page.getByRole('button', { name: 'Refresh' }).click()

    await expect(page.locator('tr', { hasText: 'sensor-unlinked' })).toBeVisible()
  })

  test('API error on initial load shows error message', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: 'DB connection failed' }) })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    await expect(page.locator('.status.error')).toBeVisible()
  })

  test('empty state shows message and link to /add', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: '[]' })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    await expect(page.getByText('No sensors registered yet')).toBeVisible()
    const addLink = page.getByRole('link', { name: /Add your first sensor/ })
    await expect(addLink).toBeVisible()
    await expect(addLink).toHaveAttribute('href', '/add')
  })
})
