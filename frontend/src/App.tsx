import { Route, Routes } from 'react-router-dom'
import { AppLayout } from './components/AppLayout'
import { Home } from './pages/Home'
import { Login } from './pages/Login'
import { Signup } from './pages/Signup'
import { Lobby } from './pages/Lobby'
import { Game } from './pages/Game'
import { Leaderboard } from './pages/Leaderboard'
import { History } from './pages/History'
import { Profile } from './pages/Profile'

export function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/login" element={<Login />} />
      <Route path="/signup" element={<Signup />} />
      <Route element={<AppLayout />}>
        <Route path="/play" element={<Lobby />} />
        <Route path="/game" element={<Game />} />
        <Route path="/leaderboard" element={<Leaderboard />} />
        <Route path="/history" element={<History />} />
        <Route path="/profile" element={<Profile />} />
        <Route path="/profile/:playerId" element={<Profile />} />
      </Route>
    </Routes>
  )
}
