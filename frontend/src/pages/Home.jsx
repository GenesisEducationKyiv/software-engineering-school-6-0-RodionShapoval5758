import { useState } from 'react'
import { subscribe, validateKey } from '../api'
import StatusMessage from '../components/StatusMessage'

export default function Home() {
  const [apiKey, setApiKey] = useState('')
  const [keyValid, setKeyValid] = useState(false)
  const [keyLoading, setKeyLoading] = useState(false)
  const [keyError, setKeyError] = useState(null)
  const [showKey, setShowKey] = useState(false)

  const [email, setEmail] = useState('')
  const [repo, setRepo] = useState('')
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState(null)

  async function handleSaveKey(e) {
    e.preventDefault()
    setKeyLoading(true)
    setKeyError(null)

    const res = await validateKey(apiKey)

    if (res.ok) {
      sessionStorage.setItem('apiKey', apiKey)
      setKeyValid(true)
    } else if (res.status === 401) {
      setKeyError('Invalid API key.')
    } else {
      setKeyError('Could not validate key. Try again.')
    }

    setKeyLoading(false)
  }

  async function handleSubscribe(e) {
    e.preventDefault()
    setLoading(true)
    setStatus(null)

    const res = await subscribe(email, repo, apiKey)

    if (res.ok) {
      setStatus({ type: 'success', message: 'Subscribed! Check your email to confirm.' })
      setEmail('')
      setRepo('')
    } else if (res.status === 409) {
      setStatus({ type: 'error', message: 'This email is already subscribed to that repository.' })
    } else if (res.status === 404) {
      setStatus({ type: 'error', message: 'Repository not found on GitHub.' })
    } else if (res.status === 400) {
      setStatus({ type: 'error', message: 'Invalid input. Use owner/repo format (e.g. golang/go).' })
    } else {
      setStatus({ type: 'error', message: 'Something went wrong. Please try again.' })
    }

    setLoading(false)
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-white rounded-2xl shadow-sm border border-gray-200 p-8">
        <h1 className="text-2xl font-semibold text-gray-900 mb-1">GitHub Release Notifications</h1>
        <p className="text-sm text-gray-500 mb-6">
          Subscribe to get notified by email when a new release drops.
        </p>

        <form onSubmit={handleSaveKey} className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">API Key</label>
            <div className="flex gap-2">
              <div className="relative flex-1">
                <input
                  type={showKey ? 'text' : 'password'}
                  required
                  value={apiKey}
                  onChange={e => setApiKey(e.target.value)}
                  disabled={keyValid}
                  placeholder="Enter your API key"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 pr-9 text-sm outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500 disabled:bg-gray-50 disabled:text-gray-400"
                />
                {!keyValid && (
                  <button
                    type="button"
                    onClick={() => setShowKey(v => !v)}
                    className="absolute inset-y-0 right-2 flex items-center text-gray-400 hover:text-gray-600"
                    aria-label={showKey ? 'Hide API key' : 'Show API key'}
                  >
                    {showKey ? (
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                      </svg>
                    ) : (
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                        <path strokeLinecap="round" strokeLinejoin="round" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                      </svg>
                    )}
                  </button>
                )}
              </div>
              {!keyValid && (
                <button
                  type="submit"
                  disabled={keyLoading || !apiKey}
                  className="bg-gray-900 hover:bg-gray-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg px-4 py-2 transition-colors"
                >
                  {keyLoading ? '…' : 'Save'}
                </button>
              )}
              {keyValid && (
                <span className="flex items-center text-sm text-green-600 font-medium px-2">
                  ✓ Saved
                </span>
              )}
            </div>
            {keyError && (
              <p className="mt-1.5 text-xs text-red-600">{keyError}</p>
            )}
          </div>
        </form>

        {keyValid && (
          <form onSubmit={handleSubscribe} className="space-y-4 mt-6 pt-6 border-t border-gray-100">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <input
                type="email"
                required
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="you@example.com"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Repository</label>
              <input
                type="text"
                required
                value={repo}
                onChange={e => setRepo(e.target.value)}
                placeholder="owner/repo (e.g. golang/go)"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </div>

            <StatusMessage type={status?.type} message={status?.message} />

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg px-4 py-2 transition-colors"
            >
              {loading ? 'Subscribing…' : 'Subscribe'}
            </button>
          </form>
        )}

        {keyValid && (
          <p className="mt-4 text-center text-sm text-gray-500">
            Already subscribed?{' '}
            <a href="/subscriptions" className="text-blue-600 hover:underline">
              View your subscriptions
            </a>
          </p>
        )}
      </div>
    </div>
  )
}
