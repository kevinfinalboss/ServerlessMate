import { useEffect } from 'react'
import { Outlet } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { Sidebar } from './Sidebar'

export function AppLayout() {
  const { connect } = useGameSocket()

  useEffect(() => {
    connect()
  }, [connect])

  return (
    <div className="min-h-screen bg-ink sm:flex">
      <Sidebar />
      <main className="min-w-0 flex-1 pb-20 sm:pb-0">
        <Outlet />
      </main>
    </div>
  )
}
