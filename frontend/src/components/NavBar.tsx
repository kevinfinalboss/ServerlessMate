import { NavLink } from 'react-router-dom'

const linkClass = ({ isActive }: { isActive: boolean }) =>
  `px-3 py-2 rounded-md text-sm font-medium ${
    isActive ? 'bg-emerald-600 text-white' : 'text-gray-300 hover:bg-gray-800 hover:text-white'
  }`

export function NavBar() {
  return (
    <nav className="flex items-center gap-2 border-b border-gray-800 px-4 py-3">
      <span className="mr-4 text-lg font-semibold text-white">ServerlessMate</span>
      <NavLink to="/" end className={linkClass}>
        Lobby
      </NavLink>
      <NavLink to="/leaderboard" className={linkClass}>
        Leaderboard
      </NavLink>
      <NavLink to="/history" className={linkClass}>
        History
      </NavLink>
      <NavLink to="/profile" className={linkClass}>
        Profile
      </NavLink>
    </nav>
  )
}
