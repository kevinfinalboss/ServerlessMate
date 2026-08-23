import { useEffect } from 'react'
import { Outlet } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { NavBar } from './NavBar'

export function AppLayout() {
  const { connect } = useGameSocket()

  useEffect(() => {
    connect()
  }, [connect])

  return (
    <div className="min-h-screen bg-ink">
      <NavBar />
      <Outlet />
    </div>
  )
}
