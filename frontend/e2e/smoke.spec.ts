import { test, expect } from '@playwright/test'

test('unauthenticated visitor is redirected to login', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Log in' })).toBeVisible()
})
