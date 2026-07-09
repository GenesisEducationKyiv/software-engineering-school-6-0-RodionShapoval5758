import { useEffect, useState } from 'react'
import { useAuth } from '../auth/useAuth'
import { subscribe, getSubscriptions } from '../api'
import StatusMessage from '../components/StatusMessage'

async function applySubscriptions(res, { setSubscriptions, setListError, logout }) {
  if (res.ok) {
    setSubscriptions(await res.json())
    setListError(null)
  } else if (res.status === 401) {
    await logout()
  } else {
    setListError('Could not load your subscriptions.')
  }
}

export default function Profile() {
  const { email, logout } = useAuth()
  const [repo, setRepo] = useState('')
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState(null)
  const [subscriptions, setSubscriptions] = useState(null)
  const [listError, setListError] = useState(null)

  useEffect(() => {
    getSubscriptions().then(res => applySubscriptions(res, { setSubscriptions, setListError, logout }))
  }, [logout])

  async function handleSubscribe(e) {
    e.preventDefault()
    setLoading(true)
    setStatus(null)

    const res = await subscribe(repo)

    if (res.ok) {
      setStatus({ type: 'success', message: 'Subscribed! Check your email to confirm.' })
      setRepo('')
      const listRes = await getSubscriptions()
      await applySubscriptions(listRes, { setSubscriptions, setListError, logout })
    } else if (res.status === 409) {
      setStatus({ type: 'error', message: 'You are already subscribed to that repository.' })
    } else if (res.status === 404) {
      setStatus({ type: 'error', message: 'Repository not found on GitHub.' })
    } else if (res.status === 400) {
      setStatus({ type: 'error', message: 'Invalid input. Use owner/repo format (e.g. golang/go).' })
    } else if (res.status === 401) {
      await logout()
    } else {
      setStatus({ type: 'error', message: 'Something went wrong. Please try again.' })
    }

    setLoading(false)
  }

  return (
    <div className="min-h-screen bg-gray-50 px-4 py-12">
      <div className="w-full max-w-lg mx-auto">
        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8 mb-6">
          <div className="flex items-start justify-between mb-6">
            <div>
              <h1 className="text-2xl font-semibold text-gray-900 mb-1">Your profile</h1>
              <p className="text-sm text-gray-500">{email}</p>
            </div>
            <button
              onClick={logout}
              className="text-sm text-gray-500 hover:text-gray-900 border border-gray-300 rounded-lg px-3 py-1.5 transition-colors"
            >
              Log out
            </button>
          </div>

          <form onSubmit={handleSubscribe} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Subscribe to a repository</label>
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
        </div>

        <h2 className="text-sm font-medium text-gray-500 mb-3 px-1">Your subscriptions</h2>

        {listError && <StatusMessage type="error" message={listError} />}

        {subscriptions && subscriptions.length === 0 && (
          <StatusMessage type="info" message="No active subscriptions yet." />
        )}

        {subscriptions && subscriptions.length > 0 && (
          <ul className="space-y-3">
            {subscriptions.map(sub => (
              <li
                key={sub.repo}
                className="bg-white rounded-xl border border-gray-200 px-5 py-4 flex items-center justify-between"
              >
                <div>
                  <p className="text-sm font-medium text-gray-900">{sub.repo}</p>
                  {sub.last_seen_tag && (
                    <p className="text-xs text-gray-400 mt-0.5">Last seen: {sub.last_seen_tag}</p>
                  )}
                </div>
                <span
                  className={`text-xs font-medium px-2 py-0.5 rounded-full ${
                    sub.confirmed
                      ? 'bg-green-100 text-green-700'
                      : 'bg-yellow-100 text-yellow-700'
                  }`}
                >
                  {sub.confirmed ? 'Confirmed' : 'Pending'}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
