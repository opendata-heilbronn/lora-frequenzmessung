import { test, expect, type Page } from '@playwright/test'

async function mockAuth(page: Page) {
  // Only intercept actual API calls — not source files like /src/auth/user.ts
  await page.route('**/auth/**', (route) => {
    if (route.request().url().includes('/src/')) {
      route.continue()
    } else {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ token: 'test-token' }) })
    }
  })
}

async function mockSerial(page: Page, lines: string[] = []) {
  await mockAuth(page)
  await page.addInitScript((scriptedLines: string[]) => {
    let lineIndex = 0
    const encoder = new TextEncoder()

    const mockPort = {
      open: async () => {},
      close: async () => {},
      readable: {
        getReader() {
          return {
            async read(): Promise<{ value: Uint8Array | undefined; done: boolean }> {
              if (lineIndex < scriptedLines.length) {
                const line = scriptedLines[lineIndex++] + '\n'
                return { value: encoder.encode(line), done: false }
              }
              await new Promise(() => {})
              return { value: undefined, done: true }
            },
            releaseLock() {},
            async cancel() {},
          }
        },
      },
    }

    const mockSerialApi = {
      async requestPort() { return mockPort },
    }

    Object.defineProperty(Navigator.prototype, 'serial', {
      get: () => mockSerialApi,
      configurable: true,
    })
  }, lines)
}

async function removeSerial(page: Page) {
  await mockAuth(page)
  await page.addInitScript(() => {
    Object.defineProperty(Navigator.prototype, 'serial', {
      get: () => undefined,
      configurable: true,
    })
  })
}

// T007: unsupported browser warning
test('shows Web Serial unsupported warning when navigator.serial unavailable', async ({ page }) => {
  await removeSerial(page)
  await page.goto('/debug')
  await expect(page.getByText(/Web Serial not supported/i)).toBeVisible()
  // Connect button still exists in toolbar but serial section is hidden — check warning is shown
  await expect(page.getByText(/Web Serial not supported/i)).toBeVisible()
})

// T008: connect flow — log lines appear with timestamp prefix
test('connect flow streams log lines with timestamps', async ({ page }) => {
  const lines = [
    'Battery: 3.80V (72.5%)',
    'PAX count: 42',
    'Payload: abc123,0,29.4000,1,72.5000',
  ]
  await mockSerial(page, lines)
  await page.goto('/debug')

  await page.getByRole('button', { name: /connect/i }).click()

  for (const line of lines) {
    await expect(page.locator('.log-pane').getByText(line)).toBeVisible({ timeout: 5000 })
  }
  await expect(page.locator('.log-pane')).toContainText(/\d{2}:\d{2}:\d{2}\.\d{3}/)
})

// T009: disconnect resets state
test('disconnect button closes port and resets to disconnected', async ({ page }) => {
  await mockSerial(page, ['Battery: 3.80V (72.5%)'])
  await page.goto('/debug')

  await page.getByRole('button', { name: /connect/i }).click()
  await expect(page.locator('.log-pane').getByText('Battery: 3.80V (72.5%)')).toBeVisible({ timeout: 5000 })

  await page.getByRole('button', { name: /disconnect/i }).click()
  await expect(page.getByRole('button', { name: /connect/i })).toBeVisible()
})

// T013: INIT state triggered by Battery line
test('Battery line activates INIT state node', async ({ page }) => {
  await mockSerial(page, ['Battery: 3.80V (72.5%)'])
  await page.goto('/debug')
  await page.getByRole('button', { name: /connect/i }).click()
  await expect(page.locator('.snode--done[data-state="INIT"]')).toBeVisible({ timeout: 5000 })
})

// T014: full cycle advances state diagram
test('full firmware cycle advances state diagram through all states', async ({ page }) => {
  const lines = [
    'Battery: 3.80V (72.5%)',
    'PAX count: 42',
    'Payload: abc123,0,29.4000,1,72.5000',
    'sendReceive state=0 downlinkLen=0',
    'TX ok',
    'Sleeping for 900s',
  ]
  await mockSerial(page, lines)
  await page.goto('/debug')
  await page.getByRole('button', { name: /connect/i }).click()

  await expect(page.locator('.snode--done[data-state="SLEEP"]')).toBeVisible({ timeout: 8000 })
})

// T017: error line gets red styling
test('TX error line is highlighted red in log pane', async ({ page }) => {
  await mockSerial(page, ['TX error -112'])
  await page.goto('/debug')
  await page.getByRole('button', { name: /connect/i }).click()
  await expect(page.locator('.log-line--error')).toBeVisible({ timeout: 5000 })
  await expect(page.locator('.log-line--error')).toContainText('TX error -112')
})

// T018: Radio init failed activates ERROR node
test('Radio init failed activates ERROR state node', async ({ page }) => {
  await mockSerial(page, ['Radio init failed'])
  await page.goto('/debug')
  await page.getByRole('button', { name: /connect/i }).click()
  await expect(page.locator('.snode--error[data-state="ERROR"]')).toBeVisible({ timeout: 5000 })
})
