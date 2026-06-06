import http from 'k6/http';
import { sleep } from 'k6';
import { MAILPIT_URL } from '../config.js';

// Search mailpit for a message sent to `toEmail` whose subject contains
// `subjectContains`. Polls up to `retries` times with `intervalMs` between
// each attempt (mail delivery is async). Returns the message ID or null.
export function findMessage(toEmail, subjectContains, { retries = 10, intervalMs = 1000 } = {}) {
  const query = encodeURIComponent(`to:${toEmail}`);

  for (let i = 0; i < retries; i++) {
    const res = http.get(`${MAILPIT_URL}/api/v1/search?query=${query}&limit=10`);
    if (res.status !== 200) {
      sleep(intervalMs / 1000);
      continue;
    }

    const data = res.json();
    const messages = data.messages || [];
    const match = messages.find(
      (m) => m.Subject && m.Subject.includes(subjectContains)
    );
    if (match) {
      return match.ID;
    }

    sleep(intervalMs / 1000);
  }

  return null;
}

// Fetch the full message body (plain text) by ID.
// Returns the Text field (plain-text body) or empty string.
export function getBody(messageId) {
  const res = http.get(`${MAILPIT_URL}/api/v1/message/${messageId}`);
  if (res.status !== 200) return '';
  const data = res.json();
  return data.Text || data.HTML || '';
}

// Extract a confirm or unsubscribe token from an email body.
// The link format is: {MAIN_URL}/{kind}/{token}
// kind is 'confirm' or 'unsubscribe'.
export function extractToken(body, kind) {
  // Match /{kind}/{token} where token ends at whitespace, quotes, or < (HTML).
  const pattern = new RegExp(`/${kind}/([^\\s"'<>\\r\\n]+)`);
  const match = body.match(pattern);
  return match ? match[1] : null;
}
