export const BASE_URL = __ENV.BASE_URL || 'http://localhost';
export const MAILPIT_URL = __ENV.MAILPIT_URL || 'http://localhost:8025';

const apiKey = __ENV.API_KEY || 'genesis-summer-school';

export const authHeaders = {
  Authorization: `Bearer ${apiKey}`,
};

// For POST/PUT requests that also carry a JSON body.
export const jsonAuthHeaders = {
  'Content-Type': 'application/json',
  Authorization: `Bearer ${apiKey}`,
};
