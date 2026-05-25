import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getSubscriptions } from '../api'
import StatusMessage from '../components/StatusMessage'

export default function Subscriptions() {
  const navigate = useNavigate()
  const [email, setEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState(null)
  const [subscriptions, setSubscriptions] = useState(null)

  async function handleSubmit(e) {
    e.preventDefault()

    const apiKey = sessionStorage.getItem('apiKey')
    if (!apiKey) {
      navigate('/')
      return
    }

    setLoading(true)
    setStatus(null)
    setSubscriptions(null)

    const res = await getSubscriptions(email, apiKey)

    if (res.ok) {
      const data = await res.json()
      setSubscriptions(data)
      if (data.length === 0) {
        setStatus({ type: 'info', message: 'No active subscriptions found for this email.' })
      }
    } else if (res.status === 401) {
      sessionStorage.removeItem('apiKey')
      navigate('/')
    } else if (res.status === 400) {
      setStatus({ type: 'error', message: 'Invalid email address.' })
    } else {
      setStatus({ type: 'error', message: 'Something went wrong. Please try again.' })
    }

    setLoading(false)
  }

  return (
    <div className="min-h-screen bg-gray-50 px-4 py-12">
      <div className="w-full max-w-lg mx-auto">
        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 p-8 mb-6">
          <h1 className="text-2xl font-semibold text-gray-900 mb-1">Your Subscriptions</h1>
          <p className="text-sm text-gray-500 mb-6">Enter your email to see your active subscriptions.</p>

          <form onSubmit={handleSubmit} className="flex gap-2">
            <input
              type="email"
              required
              value={email}
              onChange={e => setEmail(e.target.value)}
              placeholder="you@example.com"
              className="flex-1 rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
            />
            <button
              type="submit"
              disabled={loading}
              className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg px-4 py-2 transition-colors"
            >
              {loading ? '…' : 'Look up'}
            </button>
          </form>

          <StatusMessage type={status?.type} message={status?.message} />
        </div>

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

        <p className="mt-6 text-center text-sm text-gray-500">
          <a href="/" className="text-blue-600 hover:underline">← Subscribe to a new repo</a>
        </p>
      </div>
    </div>
  )
}
