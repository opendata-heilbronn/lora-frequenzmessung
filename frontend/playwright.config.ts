import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: !process.env.CI,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'unit',
      use: {
        browserName: 'chromium',
        storageState: 'e2e/auth.storage.json',
      },
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
    command: process.env.CI ? 'npx vite preview --port 5173' : 'npm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
  },
})
