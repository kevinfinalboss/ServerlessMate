import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import { getPlayerId, isGuest, setActiveGameId } from '../lib/session'
import { isGameState, isProfileResponse, isType } from '../lib/types'
import type {
  FriendsMessage,
  HistoryEntry,
  HistoryMessage,
  LeaderboardEntry,
  LeaderboardMessage,
  MatchFoundMessage,
  ProfileResponse,
  QueueJoinedMessage,
  QueueLeftMessage,
  Result,
} from '../lib/types'
import { IconTrophy, IconUsers, IconHistory, IconSwords } from '../components/icons'

const TIME_CONTROLS = ['3+0', '5+0', '10+0']

const RESULT_COLOR: Record<Result, string> = {
  win: 'text-lime',
  loss: 'text-ember',
  draw: 'text-mute',
}

function Widget({
  icon,
  title,
  viewAllTo,
  viewAllLabel,
  children,
}: {
  icon: React.ReactNode
  title: string
  viewAllTo?: string
  viewAllLabel?: string
  children: React.ReactNode
}) {
  return (
    <div className="rounded-2xl border border-ink-line bg-ink-raised p-4">
      <div className="mb-3 flex items-center justify-between">
        <h2 className="flex items-center gap-2 text-sm font-medium text-paper">
          {icon}
          {title}
        </h2>
        {viewAllTo && (
          <Link to={viewAllTo} className="text-xs font-medium text-mute hover:text-lime">
            {viewAllLabel}
          </Link>
        )}
      </div>
      {children}
    </div>
  )
}

export function Lobby() {
  const navigate = useNavigate()
  const { identity, connected, lastMessage, send } = useGameSocket()
  const { t } = useTranslation()
  const [timeControl, setTimeControl] = useState(TIME_CONTROLS[1])
  const [level, setLevel] = useState<'easy' | 'hard'>('easy')
  const [status, setStatus] = useState<string | null>(null)
  const [searchingKey, setSearchingKey] = useState<string | null>(null)

  const [profile, setProfile] = useState<ProfileResponse | null>(null)
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([])
  const [history, setHistory] = useState<HistoryEntry[]>([])
  const [friends, setFriends] = useState<FriendsMessage | null>(null)

  const myPlayerId = identity?.playerId ?? getPlayerId()

  useEffect(() => {
    if (!connected) return
    send({ action: 'getProfile' })
    send({ action: 'leaderboard' })
    send({ action: 'history' })
    if (!isGuest()) send({ action: 'listFriends' })
  }, [connected, send])

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
      return
    }
    if (isProfileResponse(lastMessage) && lastMessage.playerId === myPlayerId) {
      setProfile(lastMessage)
      return
    }
    if (isType<LeaderboardMessage>(lastMessage, 'leaderboard')) {
      setLeaderboard(lastMessage.entries)
      return
    }
    if (isType<HistoryMessage>(lastMessage, 'history')) {
      setHistory(lastMessage.entries)
      return
    }
    if (isType<FriendsMessage>(lastMessage, 'friends')) {
      setFriends(lastMessage)
    }
  }, [lastMessage, navigate, t, myPlayerId])

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

  const pendingCount = friends ? friends.incomingRequests.length : 0

  return (
    <div className="mx-auto max-w-5xl p-6">
      <header className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="font-display text-2xl font-medium text-paper">{t('dashboard.greeting')}</h1>
          {isGuest() && <p className="mt-1 text-xs text-mute">{t('play.guestNotice')}</p>}
        </div>
        <p className="text-sm text-mute">
          {t('play.connectionStatus')}:{' '}
          <span className={connected ? 'text-lime' : 'text-ember'}>
            {connected ? t('play.connected') : t('play.disconnected')}
          </span>
        </p>
      </header>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="flex flex-col gap-4 lg:col-span-2">
          <div className="rounded-2xl border border-ink-line bg-ink-raised p-5">
            <h2 className="mb-3 flex items-center gap-2 font-display text-lg font-medium text-paper">
              <IconSwords className="h-5 w-5 text-lime" />
              {t('play.vsHuman.title')}
            </h2>
            <div className="flex flex-wrap items-center gap-3">
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

            <div className="my-4 border-t border-ink-line" />

            <h2 className="mb-3 font-display text-lg font-medium text-paper">{t('play.vsAI.title')}</h2>
            {isGuest() ? (
              <p className="text-sm text-mute">{t('play.vsAI.loginRequired')}</p>
            ) : (
              <div className="flex flex-wrap items-center gap-3">
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

            {status && (
              <div className="mt-4 flex items-center gap-3 rounded-xl bg-ink px-4 py-2.5">
                <p className="text-sm text-mute">{status}</p>
                {searchingKey && (
                  <button
                    onClick={cancelSearch}
                    className="ml-auto rounded-full border border-ink-line px-3 py-1 text-xs font-medium text-paper hover:bg-ink-line"
                  >
                    {t('play.cancelSearch')}
                  </button>
                )}
              </div>
            )}
          </div>

          <Widget
            icon={<IconHistory className="h-4 w-4 text-lime" />}
            title={t('dashboard.historyWidget.title')}
            viewAllTo="/history"
            viewAllLabel={t('dashboard.historyWidget.viewAll')}
          >
            {history.length === 0 ? (
              <p className="text-sm text-mute">{t('dashboard.historyWidget.empty')}</p>
            ) : (
              <ul className="divide-y divide-ink-line">
                {history.slice(0, 4).map((entry) => (
                  <li key={entry.gameId} className="flex items-center justify-between py-2 text-sm">
                    <span className={`font-medium ${RESULT_COLOR[entry.result]}`}>
                      {t(`history.result.${entry.result}`)}
                    </span>
                    <span className="text-mute">{entry.vsAI ? t('history.vsAI') : entry.opponentId}</span>
                  </li>
                ))}
              </ul>
            )}
          </Widget>
        </div>

        <div className="flex flex-col gap-4">
          {profile?.visible && (
            <Widget icon={<IconTrophy className="h-4 w-4 text-lime" />} title={t('dashboard.ratingWidget.title')}>
              <p className="font-mono text-3xl font-medium text-lime">{profile.rating}</p>
              <p className="mt-1 font-mono text-xs text-mute">
                {profile.wins}W · {profile.losses}L · {profile.draws}D
              </p>
            </Widget>
          )}

          {isGuest() && (
            <div className="rounded-2xl border border-felt bg-felt/15 p-4">
              <p className="text-sm font-medium text-paper">{t('dashboard.guestCta.title')}</p>
              <p className="mt-1 text-xs text-mute">{t('dashboard.guestCta.body')}</p>
              <Link
                to="/signup"
                className="mt-3 inline-block rounded-full bg-lime px-4 py-1.5 text-sm font-semibold text-ink hover:opacity-90"
              >
                {t('common.signup')}
              </Link>
            </div>
          )}

          {!isGuest() && (
            <Widget
              icon={<IconUsers className="h-4 w-4 text-lime" />}
              title={t('dashboard.friendsWidget.title')}
              viewAllTo="/friends"
              viewAllLabel={t('dashboard.friendsWidget.viewAll')}
            >
              {!friends || friends.friends.length === 0 ? (
                <p className="text-sm text-mute">{t('dashboard.friendsWidget.empty')}</p>
              ) : (
                <ul className="flex flex-col gap-1.5">
                  {friends.friends.slice(0, 4).map((f) => (
                    <li key={f.playerId} className="text-sm text-paper">
                      {f.username}
                    </li>
                  ))}
                </ul>
              )}
              {pendingCount > 0 && (
                <p className="mt-2 text-xs text-lime">
                  {pendingCount} {t('dashboard.friendsWidget.pending')}
                </p>
              )}
            </Widget>
          )}

          <Widget
            icon={<IconTrophy className="h-4 w-4 text-lime" />}
            title={t('dashboard.leaderboardWidget.title')}
            viewAllTo="/leaderboard"
            viewAllLabel={t('dashboard.leaderboardWidget.viewAll')}
          >
            {leaderboard.length === 0 ? (
              <p className="text-sm text-mute">{t('leaderboard.empty')}</p>
            ) : (
              <ol className="flex flex-col gap-1.5">
                {leaderboard.slice(0, 5).map((entry, i) => (
                  <li key={entry.playerId} className="flex items-center justify-between text-sm">
                    <Link to={`/profile/${entry.playerId}`} className="text-paper hover:text-lime">
                      <span className="mr-2 font-mono text-mute">#{i + 1}</span>
                      {entry.username}
                    </Link>
                    <span className="font-mono font-tabular text-lime">{entry.rating}</span>
                  </li>
                ))}
              </ol>
            )}
          </Widget>
        </div>
      </div>
    </div>
  )
}
