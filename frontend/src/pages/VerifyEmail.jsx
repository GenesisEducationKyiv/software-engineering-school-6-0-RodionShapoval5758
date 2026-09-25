import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { verifyEmail } from '../api'
import StatusMessage from '../components/StatusMessage'

export default function VerifyEmail() {
  const { token } = useParams()
  const [status, setStatus] = useState(null)

  useEffect(() => {
    verifyEmail(token).then(res => {
      if (res.ok) {
        setStatus({ type: 'success', message: 'Your email is verified. You can now log in.' })
      } else if (res.status === 410) {
        setStatus({ type: 'error', message: 'This link has expired or was already used.' })
      } else if (res.status === 404) {
        setStatus({ type: 'error', message: 'Verification link not found.' })
      } else {
        setStatus({ type: 'error', message: 'Invalid verification link.' })
      }
    })
  }, [token])

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-white rounded-2xl shadow-sm border border-gray-200 p-8 text-center">
        <h1 className="text-2xl font-semibold text-gray-900 mb-6">Verifying your email…</h1>
        {status ? (
          <>
            <StatusMessage type={status.type} message={status.message} />
            <Link to="/login" className="mt-6 inline-block text-sm text-blue-600 hover:underline">
              Go to login
            </Link>
          </>
        ) : (
          <p className="text-sm text-gray-400">Please wait…</p>
        )}
      </div>
    </div>
  )
}
