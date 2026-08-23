import { Route, Routes } from 'react-router-dom'
import { NavBar } from './components/NavBar'
import { Lobby } from './pages/Lobby'
import { Game } from './pages/Game'
import { Leaderboard } from './pages/Leaderboard'
import { History } from './pages/History'
import { Profile } from './pages/Profile'

export function App() {
  return (
    <div className="min-h-screen bg-gray-950">
      <NavBar />
      <Routes>
        <Route path="/" element={<Lobby />} />
        <Route path="/game" element={<Game />} />
        <Route path="/leaderboard" element={<Leaderboard />} />
        <Route path="/history" element={<History />} />
        <Route path="/profile" element={<Profile />} />
        <Route path="/profile/:playerId" element={<Profile />} />
      </Routes>
    </div>
  )
}
