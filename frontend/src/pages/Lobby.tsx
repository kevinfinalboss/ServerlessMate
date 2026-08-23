import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { clearToken, isGuest, setActiveGameId, setToken } from '../lib/session'
import { isGameState, isType } from '../lib/types'
import type { MatchFoundMessage, QueueJoinedMessage } from '../lib/types'

const TIME_CONTROLS = ['3+0', '5+0', '10+0']

export function Lobby() {
  const navigate = useNavigate()
  const { connected, lastMessage, send } = useGameSocket()
  const [timeControl, setTimeControl] = useState(TIME_CONTROLS[1])
  const [level, setLevel] = useState<'easy' | 'hard'>('easy')
  const [status, setStatus] = useState<string | null>(null)
  const [tokenInput, setTokenInput] = useState('')

  useEffect(() => {
    if (!lastMessage) return
    if (isType<QueueJoinedMessage>(lastMessage, 'queueJoined')) {
      setStatus('Procurando adversário...')
      return
    }
    if (isType<MatchFoundMessage>(lastMessage, 'matchFound')) {
      setActiveGameId(lastMessage.gameId)
      navigate('/game')
      return
    }
    if (isGameState(lastMessage)) {
      setActiveGameId(lastMessage.gameId)
      navigate('/game')
    }
  }, [lastMessage, navigate])

  function joinQueue() {
    setStatus('Entrando na fila...')
    send({ action: 'joinQueue', timeControl })
  }

  function playVsAI() {
    setStatus('Criando partida contra a IA...')
    send({ action: 'start', level })
  }

  function saveToken() {
    if (tokenInput.trim()) {
      setToken(tokenInput.trim())
      window.location.reload()
    }
  }

  function logout() {
    clearToken()
    window.location.reload()
  }

  return (
    <div className="mx-auto flex max-w-xl flex-col gap-6 p-6">
      <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
        <p className="text-sm text-gray-400">
          Status da conexão:{' '}
          <span className={connected ? 'text-emerald-400' : 'text-red-400'}>
            {connected ? 'conectado' : 'desconectado'}
          </span>
        </p>
        {isGuest() ? (
          <div className="mt-3 flex gap-2">
            <input
              className="flex-1 rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-white"
              placeholder="Cole um token JWT do Cognito (opcional)"
              value={tokenInput}
              onChange={(e) => setTokenInput(e.target.value)}
            />
            <button
              onClick={saveToken}
              className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-500"
            >
              Entrar
            </button>
          </div>
        ) : (
          <div className="mt-3 flex items-center justify-between">
            <span className="text-sm text-gray-300">Autenticado</span>
            <button
              onClick={logout}
              className="rounded-md bg-gray-800 px-3 py-1.5 text-sm text-gray-300 hover:bg-gray-700"
            >
              Sair
            </button>
          </div>
        )}
        {isGuest() && (
          <p className="mt-2 text-xs text-gray-500">
            Convidados podem jogar contra outro humano, mas não contra a IA nem afetam rating.
          </p>
        )}
      </div>

      <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
        <h2 className="mb-3 text-lg font-semibold text-white">Jogar contra outro jogador</h2>
        <div className="flex items-center gap-3">
          <select
            className="rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-white"
            value={timeControl}
            onChange={(e) => setTimeControl(e.target.value)}
          >
            {TIME_CONTROLS.map((tc) => (
              <option key={tc} value={tc}>
                {tc}
              </option>
            ))}
          </select>
          <button
            onClick={joinQueue}
            disabled={!connected}
            className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-500 disabled:opacity-50"
          >
            Entrar na fila
          </button>
        </div>
      </div>

      <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
        <h2 className="mb-3 text-lg font-semibold text-white">Jogar contra a IA</h2>
        {isGuest() ? (
          <p className="text-sm text-gray-500">Faça login pra jogar contra a IA.</p>
        ) : (
          <div className="flex items-center gap-3">
            <select
              className="rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-white"
              value={level}
              onChange={(e) => setLevel(e.target.value as 'easy' | 'hard')}
            >
              <option value="easy">Fácil</option>
              <option value="hard">Difícil</option>
            </select>
            <button
              onClick={playVsAI}
              disabled={!connected}
              className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-500 disabled:opacity-50"
            >
              Jogar
            </button>
          </div>
        )}
      </div>

      {status && <p className="text-center text-sm text-gray-400">{status}</p>}
    </div>
  )
}
