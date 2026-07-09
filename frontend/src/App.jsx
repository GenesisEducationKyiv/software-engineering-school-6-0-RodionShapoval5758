import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './auth/AuthContext'
import { useAuth } from './auth/useAuth'
import ProtectedRoute from './components/ProtectedRoute'
import Login from './pages/Login'
import Register from './pages/Register'
import VerifyEmail from './pages/VerifyEmail'
import Profile from './pages/Profile'
import Confirm from './pages/Confirm'
import Unsubscribe from './pages/Unsubscribe'

function Root() {
  const { isAuthed } = useAuth()
  return <Navigate to={isAuthed ? '/profile' : '/login'} replace />
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Root />} />
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/verify-email/:token" element={<VerifyEmail />} />
          <Route path="/profile" element={<ProtectedRoute><Profile /></ProtectedRoute>} />
          <Route path="/confirm/:token" element={<Confirm />} />
          <Route path="/unsubscribe/:token" element={<Unsubscribe />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
