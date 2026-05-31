export const BASE_URL = __ENV.BASE_URL || 'http://localhost';

const apiKey = __ENV.API_KEY || 'genesis-summer-school';

export const authHeaders = {
  Authorization: `Bearer ${apiKey}`,
};
