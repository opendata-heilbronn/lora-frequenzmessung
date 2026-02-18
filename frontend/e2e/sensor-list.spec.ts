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

    // Linked sensor shows "Linked" badge
    const linkedRow = page.locator('tr', { hasText: 'sensor-linked' })
    await expect(linkedRow.locator('.badge-linked')).toHaveText('Linked')

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

    // Badge should change to "Linked"
    await expect(unlinkedRow.locator('.badge-linked')).toHaveText('Linked')
    // "Link TTN" button should be gone
    await expect(unlinkedRow.getByRole('button', { name: 'Link TTN' })).not.toBeVisible()
  })

  test('delete with TTN mentions TTN in confirm', async ({ page }) => {
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

    // Intercept dialog to capture the confirm message
    let dialogMessage = ''
    page.on('dialog', async (dialog) => {
      dialogMessage = dialog.message()
      await dialog.accept()
    })

    const linkedRow = page.locator('tr', { hasText: 'sensor-linked' })
    await linkedRow.getByRole('button', { name: 'Delete' }).click()

    expect(dialogMessage).toContain('also remove it from TTN')
    // Row should be removed
    await expect(linkedRow).not.toBeVisible()
  })

  test('delete without TTN does not mention TTN in confirm', async ({ page }) => {
    await page.route('**/api/sensors', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(SENSORS) })
      } else {
        route.continue()
      }
    })
    await page.route('**/api/sensors/bbbb-5555-6666-7777-888888888888', (route) => {
      if (route.request().method() === 'DELETE') {
        route.fulfill({ status: 200, contentType: 'application/json', body: '{}' })
      } else {
        route.continue()
      }
    })

    await page.goto('/')

    let dialogMessage = ''
    page.on('dialog', async (dialog) => {
      dialogMessage = dialog.message()
      await dialog.accept()
    })

    const unlinkedRow = page.locator('tr', { hasText: 'sensor-unlinked' })
    await unlinkedRow.getByRole('button', { name: 'Delete' }).click()

    expect(dialogMessage).not.toContain('TTN')
    await expect(unlinkedRow).not.toBeVisible()
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
