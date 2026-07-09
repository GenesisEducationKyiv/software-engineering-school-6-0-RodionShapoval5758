import { test, expect } from '@playwright/test'
import { resetMailbox, findMessage, extractLink } from './mailpit'

const TEST_EMAIL = `e2e-${Date.now()}@example.com`
const TEST_PASSWORD = 'e2e-test-passw0rd'
const TEST_REPO = 'golang/go'

test.beforeEach(async ({ request }) => {
  await resetMailbox(request)
})

test('register → verify → log in → subscribe → confirm', async ({ page, request }) => {
  // ── Step 1: register ────────────────────────────────────────────────────
  await page.goto('/register')
  await page.getByPlaceholder('you@example.com').fill(TEST_EMAIL)
  await page.getByPlaceholder('At least 8 characters').fill(TEST_PASSWORD)
  await page.getByRole('button', { name: 'Register' }).click()

  await expect(page.getByText('Check your email for a verification link, then log in.')).toBeVisible()

  // ── Step 2: verify email via the link Mailpit received ───────────────────
  const verifyMsgId = await findMessage(request, TEST_EMAIL, 'Verify your email address')
  const verifyLink = await extractLink(request, verifyMsgId, 'verify-email')
  await page.goto(verifyLink)

  await expect(page.getByText('Your email is verified. You can now log in.')).toBeVisible()

  // ── Step 3: log in ────────────────────────────────────────────────────────
  await page.getByRole('link', { name: 'Go to login' }).click()
  await page.getByPlaceholder('you@example.com').fill(TEST_EMAIL)
  await page.getByPlaceholder('••••••••').fill(TEST_PASSWORD)
  await page.getByRole('button', { name: 'Log in' }).click()

  await expect(page).toHaveURL('/profile')
  await expect(page.getByText(TEST_EMAIL)).toBeVisible()

  // ── Step 4: subscribe ────────────────────────────────────────────────────
  await page.getByPlaceholder('owner/repo (e.g. golang/go)').fill(TEST_REPO)
  await page.getByRole('button', { name: 'Subscribe' }).click()

  await expect(page.getByText('Subscribed! Check your email to confirm.')).toBeVisible()

  // ── Step 5: confirm via the link Mailpit received ─────────────────────────
  const confirmMsgId = await findMessage(request, TEST_EMAIL, `Confirm subscription: ${TEST_REPO}`)
  const confirmLink = await extractLink(request, confirmMsgId, 'confirm')
  await page.goto(confirmLink)

  await expect(
    page.getByText('Your subscription is confirmed! You will now receive release notifications.')
  ).toBeVisible()

  // ── Step 6: confirmed subscription now shows on the profile page ─────────
  await page.goto('/profile')
  await expect(page.getByText(TEST_REPO)).toBeVisible()
  await expect(page.getByText('Confirmed')).toBeVisible()
})
