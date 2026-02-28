import { test, expect } from '@playwright/test'

test('debug prototype override', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('token', 'test-token')
    try {
      Object.defineProperty(Navigator.prototype, 'serial', { get: () => undefined, configurable: true })
      console.log('override succeeded')
    } catch(e) {
      console.log('override failed:', e)
    }
  })
  await page.goto('/add')
  const hasSerial = await page.evaluate(() => 'serial' in navigator)
  const serialVal = await page.evaluate(() => typeof (navigator as any).serial)
  console.log('serial in navigator after override:', hasSerial, 'type:', serialVal)
})
