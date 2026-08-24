import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import { isGuest, setActiveGameId } from '../lib/session'
import { isGameState, isType } from '../lib/types'
import type { MatchFoundMessage, QueueJoinedMessage, QueueLeftMessage } from '../lib/types'

const TIME_CONTROLS = ['3+0', '5+0', '10+0']

export function Lobby() {
  const navigate = useNavigate()
  const { connected, lastMessage, send } = useGameSocket()
  const { t } = useTranslation()
  const [timeControl, setTimeControl] = useState(TIME_CONTROLS[1])
  const [level, setLevel] = useState<'easy' | 'hard'>('easy')
  const [status, setStatus] = useState<string | null>(null)
  const [searchingKey, setSearchingKey] = useState<string | null>(null)

  useEffect(() => {
    if (!lastMessage) return
    if (isType<QueueJoinedMessage>(lastMessage, 'queueJoined')) {
      setSearchingKey(lastMessage.matchmakingKey)
      setStatus(t('play.status.searching'))
      return
    }
    if (isType<QueueLeftMessage>(lastMessage, 'queueLeft')) {
      setSearchingKey(null)
      setStatus(null)
      return
    }
    if (isType<MatchFoundMessage>(lastMessage, 'matchFound')) {
      setSearchingKey(null)
      setActiveGameId(lastMessage.gameId)
      navigate('/game')
      return
    }
    if (isGameState(lastMessage)) {
      setSearchingKey(null)
      setActiveGameId(lastMessage.gameId)
      navigate('/game')
    }
  }, [lastMessage, navigate, t])

  function joinQueue() {
    setStatus(t('play.status.searching'))
    send({ action: 'joinQueue', timeControl })
  }

  function cancelSearch() {
    if (!searchingKey) return
    send({ action: 'leaveQueue', matchmakingKey: searchingKey })
  }

  function playVsAI() {
    setStatus(t('play.status.creatingAI'))
    send({ action: 'start', level })
  }

  return (
    <div className="mx-auto flex max-w-xl flex-col gap-6 p-6">
      <div className="rounded-2xl border border-ink-line bg-ink-raised p-5">
        <p className="text-sm text-mute">
          {t('play.connectionStatus')}:{' '}
          <span className={connected ? 'text-lime' : 'text-ember'}>
            {connected ? t('play.connected') : t('play.disconnected')}
          </span>
        </p>
        {isGuest() && <p className="mt-2 text-xs text-mute">{t('play.guestNotice')}</p>}
      </div>

      <div className="rounded-2xl border border-ink-line bg-ink-raised p-5">
        <h2 className="mb-3 font-display text-lg font-medium text-paper">{t('play.vsHuman.title')}</h2>
        <div className="flex items-center gap-3">
          <select
            className="rounded-lg border border-ink-line bg-ink px-3 py-2 text-sm text-paper outline-none focus:border-lime"
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
            disabled={!connected || Boolean(status)}
            className="rounded-full bg-lime px-4 py-2 text-sm font-semibold text-ink transition-transform hover:scale-[1.02] active:scale-[0.98] disabled:opacity-50"
          >
            {t('play.vsHuman.cta')}
          </button>
        </div>
      </div>

      <div className="rounded-2xl border border-ink-line bg-ink-raised p-5">
        <h2 className="mb-3 font-display text-lg font-medium text-paper">{t('play.vsAI.title')}</h2>
        {isGuest() ? (
          <p className="text-sm text-mute">{t('play.vsAI.loginRequired')}</p>
        ) : (
          <div className="flex items-center gap-3">
            <select
              className="rounded-lg border border-ink-line bg-ink px-3 py-2 text-sm text-paper outline-none focus:border-lime"
              value={level}
              onChange={(e) => setLevel(e.target.value as 'easy' | 'hard')}
            >
              <option value="easy">{t('play.vsAI.easy')}</option>
              <option value="hard">{t('play.vsAI.hard')}</option>
            </select>
            <button
              onClick={playVsAI}
              disabled={!connected || Boolean(status)}
              className="rounded-full bg-lime px-4 py-2 text-sm font-semibold text-ink transition-transform hover:scale-[1.02] active:scale-[0.98] disabled:opacity-50"
            >
              {t('play.vsAI.cta')}
            </button>
          </div>
        )}
      </div>

      {status && (
        <div className="flex flex-col items-center gap-2">
          <p className="text-center text-sm text-mute">{status}</p>
          {searchingKey && (
            <button
              onClick={cancelSearch}
              className="rounded-full border border-ink-line px-4 py-1.5 text-sm font-medium text-paper hover:bg-ink-raised"
            >
              {t('play.cancelSearch')}
            </button>
          )}
        </div>
      )}
    </div>
  )
}
