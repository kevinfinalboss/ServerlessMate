import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import { isType } from '../lib/types'
import type { LeaderboardEntry, LeaderboardMessage } from '../lib/types'

export function Leaderboard() {
  const { connected, lastMessage, send } = useGameSocket()
  const { t } = useTranslation()
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
      <h1 className="mb-4 font-display text-2xl font-medium text-paper">{t('leaderboard.title')}</h1>
      <ol className="divide-y divide-ink-line overflow-hidden rounded-2xl border border-ink-line bg-ink-raised">
        {entries.map((entry, i) => (
          <li key={entry.playerId} className="flex items-center justify-between px-4 py-3 text-sm text-paper">
            <Link to={`/profile/${entry.playerId}`} className="hover:text-lime">
              <span className="mr-2 font-mono text-mute">#{i + 1}</span>
              {entry.username}
            </Link>
            <span className="font-mono font-tabular text-lime">{entry.rating}</span>
          </li>
        ))}
        {entries.length === 0 && <li className="px-4 py-4 text-sm text-mute">{t('leaderboard.empty')}</li>}
      </ol>
    </div>
  )
}
