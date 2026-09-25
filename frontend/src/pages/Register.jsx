import { useState } from 'react'
import { Link } from 'react-router-dom'
import { register } from '../api'
import StatusMessage from '../components/StatusMessage'

export default function Register() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState(null)
  const [registered, setRegistered] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setLoading(true)
    setStatus(null)

    const res = await register(email, password)

    if (res.status === 201) {
      setRegistered(true)
    } else if (res.status === 409) {
      setStatus({ type: 'error', message: 'That email is already registered.' })
    } else if (res.status === 400) {
      let message = 'Invalid email or password.'
      try {
        const body = await res.json()
        if (body.error) message = body.error
      } catch { /* keep default message */ }
      setStatus({ type: 'error', message })
    } else {
      setStatus({ type: 'error', message: 'Something went wrong. Please try again.' })
    }

    setLoading(false)
  }

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-white rounded-2xl shadow-sm border border-gray-200 p-8">
        <h1 className="text-2xl font-semibold text-gray-900 mb-1">Create an account</h1>
        <p className="text-sm text-gray-500 mb-6">
          Register to subscribe to GitHub release notifications.
        </p>

        {registered ? (
          <StatusMessage
            type="success"
            message="Check your email for a verification link, then log in."
          />
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
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
              <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
              <input
                type="password"
                required
                minLength={8}
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder="At least 8 characters"
                className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500"
              />
            </div>

            <StatusMessage type={status?.type} message={status?.message} />

            <button
              type="submit"
              disabled={loading}
              className="w-full bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg px-4 py-2 transition-colors"
            >
              {loading ? 'Registering…' : 'Register'}
            </button>
          </form>
        )}

        <p className="mt-4 text-center text-sm text-gray-500">
          Already have an account?{' '}
          <Link to="/login" className="text-blue-600 hover:underline">
            Log in
          </Link>
        </p>
      </div>
    </div>
  )
}
