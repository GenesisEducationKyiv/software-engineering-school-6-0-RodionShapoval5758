import { BrowserRouter, Routes, Route } from 'react-router-dom'
import Home from './pages/Home'
import Subscriptions from './pages/Subscriptions'
import Confirm from './pages/Confirm'
import Unsubscribe from './pages/Unsubscribe'

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/subscriptions" element={<Subscriptions />} />
        <Route path="/confirm/:token" element={<Confirm />} />
        <Route path="/unsubscribe/:token" element={<Unsubscribe />} />
      </Routes>
    </BrowserRouter>
  )
}
