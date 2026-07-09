import { authedFetch } from './auth/authedFetch'

const API_BASE = '/api'
const AUTH_BASE = '/auth'

function postJSON(path, body) {
  return fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function register(email, password) {
  return postJSON(`${AUTH_BASE}/register`, { email, password })
}

export async function login(email, password) {
  return postJSON(`${AUTH_BASE}/login`, { email, password })
}

export async function verifyEmail(token) {
  return fetch(`${AUTH_BASE}/verify-email/${token}`)
}

export async function logout(refreshToken) {
  return postJSON(`${AUTH_BASE}/logout`, { refresh_token: refreshToken })
}

export async function subscribe(repo) {
  return authedFetch(`${API_BASE}/subscribe`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ repo }),
  })
}

export async function getSubscriptions() {
  return authedFetch(`${API_BASE}/subscriptions`)
}

export async function confirmSubscription(token) {
  return fetch(`${API_BASE}/confirm/${token}`)
}

export async function unsubscribe(token) {
  return fetch(`${API_BASE}/unsubscribe/${token}`)
}
