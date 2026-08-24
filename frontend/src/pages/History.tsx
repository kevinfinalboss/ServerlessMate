import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import { isType } from '../lib/types'
import type { HistoryEntry, HistoryMessage, ReplayMessage, Result } from '../lib/types'

const RESULT_COLOR: Record<Result, string> = {
  win: 'text-lime',
  loss: 'text-ember',
  draw: 'text-mute',
}

export function History() {
  const { connected, lastMessage, send } = useGameSocket()
  const { t } = useTranslation()
  const [entries, setEntries] = useState<HistoryEntry[]>([])
  const [replay, setReplay] = useState<ReplayMessage | null>(null)

  useEffect(() => {
    if (connected) send({ action: 'history' })
  }, [connected, send])

  useEffect(() => {
    if (!lastMessage) return
    if (isType<HistoryMessage>(lastMessage, 'history')) {
      setEntries(lastMessage.entries)
      return
    }
    if (isType<ReplayMessage>(lastMessage, 'replay')) {
      setReplay(lastMessage)
    }
  }, [lastMessage])

  function viewReplay(gameId: string) {
    send({ action: 'history', gameId })
  }

  return (
    <div className="mx-auto max-w-2xl p-6">
      <h1 className="mb-4 font-display text-2xl font-medium text-paper">{t('history.title')}</h1>
      <ul className="divide-y divide-ink-line overflow-hidden rounded-2xl border border-ink-line bg-ink-raised">
        {entries.map((entry) => (
          <li key={entry.gameId} className="flex items-center justify-between px-4 py-3 text-sm">
            <span className={`font-medium ${RESULT_COLOR[entry.result]}`}>{t(`history.result.${entry.result}`)}</span>
            <span className="text-mute">
              vs{' '}
              {entry.vsAI ? (
                t('history.vsAI')
              ) : (
                <Link to={`/profile/${entry.opponentId}`} className="hover:text-lime">
                  {entry.opponentId}
                </Link>
              )}
            </span>
            <button
              onClick={() => viewReplay(entry.gameId)}
              className="rounded-full bg-ink-line px-3 py-1 text-xs font-medium text-paper hover:bg-felt"
            >
              {t('history.viewPgn')}
            </button>
          </li>
        ))}
        {entries.length === 0 && <li className="px-4 py-4 text-sm text-mute">{t('history.empty')}</li>}
      </ul>

      {replay && (
        <div className="mt-4 rounded-2xl border border-ink-line bg-ink-raised p-4">
          <p className="mb-2 font-mono text-xs text-mute">{replay.gameId}</p>
          <pre className="font-mono text-sm whitespace-pre-wrap text-paper">{replay.pgn}</pre>
        </div>
      )}
    </div>
  )
}
