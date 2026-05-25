import { test, expect } from '@playwright/test'

test('home page loads', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText('GitHub Release Notifications')).toBeVisible()
})
