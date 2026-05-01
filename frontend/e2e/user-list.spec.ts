import { test, expect } from '@playwright/test'

const USERS = [
  { id: 'aaaaaaaa-0000-0000-0000-000000000001', username: 'alice', has_password: true },
  { id: 'aaaaaaaa-0000-0000-0000-000000000002', username: 'bob', has_password: false },
]

const NEW_USER = { id: 'aaaaaaaa-0000-0000-0000-000000000003', username: 'charlie', has_password: true }

test.describe('UserList', () => {
  test('shows user list', async ({ page }) => {
    await page.route('**/api/users**', (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ users: USERS, next_page_token: '' }) })
    })

    await page.goto('/users')

    await expect(page.getByText('alice')).toBeVisible()
    await expect(page.getByText('bob')).toBeVisible()
  })

  test('opens create user modal and submits', async ({ page }) => {
    let createCalled = false

    await page.route('**/api/users**', (route) => {
      if (route.request().method() === 'POST') {
        createCalled = true
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(NEW_USER) })
      } else {
        const updatedUsers = createCalled ? [...USERS, NEW_USER] : USERS
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ users: updatedUsers, next_page_token: '' }) })
      }
    })

    await page.goto('/users')

    await page.getByRole('button', { name: 'Create new User' }).click()
    await expect(page.getByRole('dialog')).toBeVisible()

    await page.getByLabel('Username').fill('charlie')
    await page.locator('input[type="password"]').fill('secret123')
    await page.getByRole('button', { name: 'Create', exact: true }).click()

    await expect(page.getByText('charlie')).toBeVisible()
    expect(createCalled).toBe(true)
  })

  test('delete user shows confirmation modal', async ({ page }) => {
    await page.route('**/api/users**', (route) => {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ users: USERS, next_page_token: '' }) })
    })

    await page.goto('/users')

    await page.getByText('alice').waitFor()
    const aliceRow = page.locator('tr', { hasText: 'alice' })
    await aliceRow.getByRole('button', { name: /delete/i }).click()

    await expect(page.getByText(/Confirm deletion of user alice/)).toBeVisible()
  })
})
