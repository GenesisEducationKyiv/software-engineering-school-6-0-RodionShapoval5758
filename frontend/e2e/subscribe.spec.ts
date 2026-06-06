import { test, expect, type APIRequestContext } from '@playwright/test'

const API_KEY = process.env.API_KEY ?? 'test-api-key'
const TEST_EMAIL = 'e2e@example.com'
const TEST_REPO = 'golang/go'
const MAILPIT_URL = 'http://localhost:8025'

async function extractConfirmURL(request: APIRequestContext): Promise<string> {
  for (let attempt = 0; attempt < 10; attempt++) {
    const res = await request.get(`${MAILPIT_URL}/api/v1/messages`)
    const data = await res.json()

    if (data.messages?.length > 0) {
      const id: string = data.messages[0].ID
      const msgRes = await request.get(`${MAILPIT_URL}/api/v1/message/${id}`)
      const msg = await msgRes.json()

      const match = (msg.HTML as string).match(/href="(http:\/\/localhost\/confirm\/[^"]+)"/)
      if (match) return match[1]
    }

    await new Promise(r => setTimeout(r, 500))
  }

  throw new Error('Confirmation email did not arrive within 5 seconds')
}

test.beforeEach(async ({ request }) => {
  await request.delete(`${MAILPIT_URL}/api/v1/messages`)
})

test('subscribe → confirm → view subscription', async ({ page, request }) => {
  // ── Step 1: validate API key ──────────────────────────────────────────────
  await page.goto('/')

  await page.getByPlaceholder('Enter your API key').fill(API_KEY)
  await page.getByRole('button', { name: 'Save' }).click()

  await expect(page.getByText('✓ Saved')).toBeVisible()

  // ── Step 2: subscribe ─────────────────────────────────────────────────────
  await page.getByPlaceholder('you@example.com').fill(TEST_EMAIL)
  await page.getByPlaceholder('owner/repo (e.g. golang/go)').fill(TEST_REPO)
  await page.getByRole('button', { name: 'Subscribe' }).click()

  await expect(page.getByText('Subscribed! Check your email to confirm.')).toBeVisible()

  // ── Step 3: confirm via email link ────────────────────────────────────────
  const confirmURL = await extractConfirmURL(request)
  await page.goto(confirmURL)

  await expect(
    page.getByText('Your subscription is confirmed! You will now receive release notifications.')
  ).toBeVisible()

  // ── Step 4: verify subscription appears as confirmed ──────────────────────
  await page.goto('/subscriptions')

  await page.getByPlaceholder('you@example.com').fill(TEST_EMAIL)
  await page.getByRole('button', { name: 'Look up' }).click()

  await expect(page.getByText(TEST_REPO)).toBeVisible()
  await expect(page.getByText('Confirmed')).toBeVisible()
})
