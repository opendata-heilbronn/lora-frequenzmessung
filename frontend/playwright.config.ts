import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'unit',
      use: { browserName: 'chromium' },
      testIgnore: /.*-live\.spec\.ts/,
    },
    {
      name: 'integration',
      use: { browserName: 'chromium' },
      testMatch: /.*-live\.spec\.ts/,
      fullyParallel: false,
      dependencies: ['unit'],
    },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
  },
})
