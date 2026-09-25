import type { APIRequestContext } from '@playwright/test'

const MAILPIT_URL = 'http://localhost:8025'

export async function resetMailbox(request: APIRequestContext) {
  await request.delete(`${MAILPIT_URL}/api/v1/messages`)
}

export async function findMessage(
  request: APIRequestContext,
  toEmail: string,
  subjectContains: string,
  { retries = 10, intervalMs = 500 } = {}
): Promise<string> {
  for (let attempt = 0; attempt < retries; attempt++) {
    const res = await request.get(`${MAILPIT_URL}/api/v1/search`, {
      params: { query: `to:${toEmail}`, limit: 10 },
    })
    const data = await res.json()
    const match = (data.messages ?? []).find((m: { Subject?: string }) =>
      m.Subject?.includes(subjectContains)
    )
    if (match) return match.ID

    await new Promise(r => setTimeout(r, intervalMs))
  }

  throw new Error(`No message with subject containing "${subjectContains}" arrived for ${toEmail}`)
}

export async function extractLink(
  request: APIRequestContext,
  messageId: string,
  path: 'verify-email' | 'confirm'
): Promise<string> {
  const res = await request.get(`${MAILPIT_URL}/api/v1/message/${messageId}`)
  const msg = await res.json()
  const match = (msg.HTML as string).match(new RegExp(`href="(http://localhost/${path}/[^"]+)"`))
  if (!match) throw new Error(`Could not find a /${path}/ link in the email body`)
  return match[1]
}
