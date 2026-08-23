import { useEffect, useState } from 'react'
import { useGameSocket } from '../lib/GameSocketProvider'
import { isType } from '../lib/types'
import type { HistoryEntry, HistoryMessage, ReplayMessage } from '../lib/types'

const RESULT_COLOR: Record<string, string> = {
  win: 'text-emerald-400',
  loss: 'text-red-400',
  draw: 'text-gray-400',
}

export function History() {
  const { connected, lastMessage, send } = useGameSocket()
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
      <h1 className="mb-4 text-xl font-semibold text-white">Histórico</h1>
      <ul className="divide-y divide-gray-800 rounded-md border border-gray-800 bg-gray-900">
        {entries.map((entry) => (
          <li key={entry.gameId} className="flex items-center justify-between px-4 py-2 text-sm">
            <span className={RESULT_COLOR[entry.result]}>{entry.result}</span>
            <span className="text-gray-400">vs {entry.vsAI ? 'IA' : entry.opponentId}</span>
            <button
              onClick={() => viewReplay(entry.gameId)}
              className="rounded-md bg-gray-700 px-2 py-1 text-xs text-white hover:bg-gray-600"
            >
              Ver PGN
            </button>
          </li>
        ))}
        {entries.length === 0 && <li className="px-4 py-3 text-sm text-gray-500">Nenhuma partida ainda.</li>}
      </ul>

      {replay && (
        <div className="mt-4 rounded-md border border-gray-800 bg-gray-900 p-4">
          <p className="mb-2 text-sm text-gray-400">PGN — {replay.gameId}</p>
          <pre className="whitespace-pre-wrap text-sm text-gray-200">{replay.pgn}</pre>
        </div>
      )}
    </div>
  )
}
