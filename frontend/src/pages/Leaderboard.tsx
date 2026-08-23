import { useEffect, useState } from 'react'
import { useGameSocket } from '../lib/GameSocketProvider'
import { isType } from '../lib/types'
import type { LeaderboardEntry, LeaderboardMessage } from '../lib/types'

export function Leaderboard() {
  const { connected, lastMessage, send } = useGameSocket()
  const [entries, setEntries] = useState<LeaderboardEntry[]>([])

  useEffect(() => {
    if (connected) send({ action: 'leaderboard' })
  }, [connected, send])

  useEffect(() => {
    if (lastMessage && isType<LeaderboardMessage>(lastMessage, 'leaderboard')) {
      setEntries(lastMessage.entries)
    }
  }, [lastMessage])

  return (
    <div className="mx-auto max-w-xl p-6">
      <h1 className="mb-4 text-xl font-semibold text-white">Leaderboard</h1>
      <ol className="divide-y divide-gray-800 rounded-md border border-gray-800 bg-gray-900">
        {entries.map((entry, i) => (
          <li key={entry.playerId} className="flex items-center justify-between px-4 py-2 text-sm text-gray-200">
            <span>
              #{i + 1} {entry.username}
            </span>
            <span className="font-mono text-emerald-400">{entry.rating}</span>
          </li>
        ))}
        {entries.length === 0 && <li className="px-4 py-3 text-sm text-gray-500">Sem dados ainda.</li>}
      </ol>
    </div>
  )
}
