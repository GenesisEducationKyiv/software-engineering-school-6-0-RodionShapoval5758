import { useState, useCallback } from 'react'
import { AuthContext } from './context'
import { getTokens, setTokens, clearTokens } from './tokens'
import { emailFromToken } from './jwt'
import { login as loginRequest, logout as logoutRequest } from '../api'

export function AuthProvider({ children }) {
  const [email, setEmail] = useState(() => emailFromToken(getTokens().accessToken))

  const login = useCallback(async (loginEmail, password) => {
    const res = await loginRequest(loginEmail, password)
    if (res.ok) {
      const pair = await res.json()
      setTokens(pair)
      setEmail(emailFromToken(pair.access_token))
    }
    return res
  }, [])

  const logout = useCallback(async () => {
    const { refreshToken } = getTokens()
    if (refreshToken) {
      await logoutRequest(refreshToken).catch(() => {})
    }
    clearTokens()
    setEmail(null)
  }, [])

  return (
    <AuthContext.Provider value={{ email, isAuthed: Boolean(email), login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
