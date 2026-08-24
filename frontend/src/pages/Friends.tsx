import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import { isGuest } from '../lib/session'
import { isType } from '../lib/types'
import type { FriendEntry, FriendEventMessage, FriendsMessage } from '../lib/types'
import { IconCheck, IconX, IconUserPlus } from '../components/icons'

function EntryRow({ entry, actions }: { entry: FriendEntry; actions?: React.ReactNode }) {
  return (
    <li className="flex items-center justify-between gap-3 px-4 py-3">
      <Link to={`/profile/${entry.playerId}`} className="truncate text-sm font-medium text-paper hover:text-lime">
        {entry.username}
      </Link>
      {actions && <div className="flex shrink-0 gap-2">{actions}</div>}
    </li>
  )
}

export function Friends() {
  const { connected, lastMessage, send } = useGameSocket()
  const { t } = useTranslation()
  const [friends, setFriends] = useState<FriendEntry[]>([])
  const [incoming, setIncoming] = useState<FriendEntry[]>([])
  const [outgoing, setOutgoing] = useState<FriendEntry[]>([])
  const [addPlayerId, setAddPlayerId] = useState('')
  const [addStatus, setAddStatus] = useState<'sent' | 'error' | null>(null)

  useEffect(() => {
    if (connected && !isGuest()) send({ action: 'listFriends' })
  }, [connected, send])

  useEffect(() => {
    if (!lastMessage) return
    if (isType<FriendsMessage>(lastMessage, 'friends')) {
      setFriends(lastMessage.friends)
      setIncoming(lastMessage.incomingRequests)
      setOutgoing(lastMessage.outgoingRequests)
      return
    }
    const eventTypes: FriendEventMessage['type'][] = [
      'friendRequestSent',
      'friendRequestAccepted',
      'playerBlocked',
      'friendRequestCancelled',
    ]
    if (eventTypes.some((type) => isType<FriendEventMessage>(lastMessage, type))) {
      send({ action: 'listFriends' })
    }
  }, [lastMessage, send])

  function addFriend() {
    const friendId = addPlayerId.trim()
    if (!friendId) return
    send({ action: 'sendRequest', friendId })
    setAddPlayerId('')
    setAddStatus('sent')
  }

  function accept(friendId: string) {
    send({ action: 'acceptRequest', friendId })
  }

  function cancel(friendId: string) {
    send({ action: 'cancelRequest', friendId })
  }

  if (isGuest()) {
    return (
      <div className="mx-auto max-w-2xl p-6">
        <h1 className="mb-3 font-display text-2xl font-medium text-paper">{t('friends.title')}</h1>
        <p className="text-sm text-mute">{t('friends.loginRequired')}</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl p-6">
      <h1 className="mb-6 font-display text-2xl font-medium text-paper">{t('friends.title')}</h1>

      {incoming.length > 0 && (
        <section className="mb-6">
          <h2 className="mb-2 text-sm font-medium text-mute">{t('friends.incoming.title')}</h2>
          <ul className="divide-y divide-ink-line overflow-hidden rounded-2xl border border-ink-line bg-ink-raised">
            {incoming.map((entry) => (
              <EntryRow
                key={entry.playerId}
                entry={entry}
                actions={
                  <>
                    <button
                      onClick={() => accept(entry.playerId)}
                      aria-label={t('friends.accept')}
                      className="rounded-full bg-lime p-1.5 text-ink hover:opacity-90"
                    >
                      <IconCheck className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => cancel(entry.playerId)}
                      aria-label={t('friends.decline')}
                      className="rounded-full bg-ink-line p-1.5 text-paper hover:bg-ember/80"
                    >
                      <IconX className="h-4 w-4" />
                    </button>
                  </>
                }
              />
            ))}
          </ul>
        </section>
      )}

      <section className="mb-6">
        <h2 className="mb-2 text-sm font-medium text-mute">{t('friends.title')}</h2>
        <ul className="divide-y divide-ink-line overflow-hidden rounded-2xl border border-ink-line bg-ink-raised">
          {friends.map((entry) => (
            <EntryRow key={entry.playerId} entry={entry} />
          ))}
          {friends.length === 0 && <li className="px-4 py-4 text-sm text-mute">{t('friends.empty')}</li>}
        </ul>
      </section>

      {outgoing.length > 0 && (
        <section className="mb-6">
          <h2 className="mb-2 text-sm font-medium text-mute">{t('friends.outgoing.title')}</h2>
          <ul className="divide-y divide-ink-line overflow-hidden rounded-2xl border border-ink-line bg-ink-raised">
            {outgoing.map((entry) => (
              <EntryRow
                key={entry.playerId}
                entry={entry}
                actions={
                  <button
                    onClick={() => cancel(entry.playerId)}
                    className="rounded-full bg-ink-line px-3 py-1 text-xs font-medium text-paper hover:bg-ember/80"
                  >
                    {t('friends.cancel')}
                  </button>
                }
              />
            ))}
          </ul>
        </section>
      )}

      <section className="rounded-2xl border border-ink-line bg-ink-raised p-4">
        <h2 className="mb-1 flex items-center gap-2 text-sm font-medium text-paper">
          <IconUserPlus className="h-4 w-4 text-lime" />
          {t('friends.addTitle')}
        </h2>
        <p className="mb-3 text-xs text-mute">{t('friends.addHint')}</p>
        <div className="flex gap-2">
          <input
            value={addPlayerId}
            onChange={(e) => {
              setAddPlayerId(e.target.value)
              setAddStatus(null)
            }}
            onKeyDown={(e) => e.key === 'Enter' && addFriend()}
            placeholder={t('friends.addPlaceholder')}
            className="flex-1 rounded-lg border border-ink-line bg-ink px-3 py-2 text-sm text-paper outline-none focus:border-lime"
          />
          <button
            onClick={addFriend}
            className="rounded-lg bg-lime px-4 py-2 text-sm font-semibold text-ink hover:opacity-90"
          >
            {t('friends.addCta')}
          </button>
        </div>
        {addStatus === 'sent' && <p className="mt-2 text-xs text-lime">{t('friends.addSent')}</p>}
      </section>
    </div>
  )
}
