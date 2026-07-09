import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { confirmSubscription } from '../api'
import StatusMessage from '../components/StatusMessage'

export default function Confirm() {
  const { token } = useParams()
  const [status, setStatus] = useState(null)

  useEffect(() => {
    confirmSubscription(token).then(res => {
      if (res.ok) {
        setStatus({ type: 'success', message: 'Your subscription is confirmed! You will now receive release notifications.' })
      } else if (res.status === 404) {
        setStatus({ type: 'error', message: 'Token not found. It may have already been used.' })
      } else {
        setStatus({ type: 'error', message: 'Invalid or expired token.' })
      }
    })
  }, [token])

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center px-4">
      <div className="w-full max-w-md bg-white rounded-2xl shadow-sm border border-gray-200 p-8 text-center">
        <h1 className="text-2xl font-semibold text-gray-900 mb-6">Confirming subscription…</h1>
        {status ? (
          <>
            <StatusMessage type={status.type} message={status.message} />
            <a href="/" className="mt-6 inline-block text-sm text-blue-600 hover:underline">
              Back to home
            </a>
          </>
        ) : (
          <p className="text-sm text-gray-400">Please wait…</p>
        )}
      </div>
    </div>
  )
}
