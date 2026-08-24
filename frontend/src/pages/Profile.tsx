import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import { getPlayerId } from '../lib/session'
import { isProfileResponse, isType } from '../lib/types'
import type { ErrorMessage, FriendEventMessage, ProfileResponse } from '../lib/types'
import { IconUserPlus, IconCheck } from '../components/icons'

export function Profile() {
  const { playerId } = useParams()
  const { identity, connected, lastMessage, send } = useGameSocket()
  const { t } = useTranslation()
  const [profile, setProfile] = useState<ProfileResponse | null>(null)
  const [friendStatus, setFriendStatus] = useState<'idle' | 'sent' | 'error'>('idle')

  const myPlayerId = identity?.playerId ?? getPlayerId()
  const isOwnProfile = !playerId || playerId === myPlayerId

  useEffect(() => {
    if (connected) send({ action: 'getProfile', playerId })
  }, [connected, playerId, send])

  useEffect(() => {
    if (!lastMessage) return
    if (isProfileResponse(lastMessage)) {
      setProfile(lastMessage)
      return
    }
    if (isType<FriendEventMessage>(lastMessage, 'friendRequestSent')) {
      setFriendStatus('sent')
      return
    }
    if (isType<ErrorMessage>(lastMessage, 'error')) {
      setFriendStatus('error')
    }
  }, [lastMessage])

  function addFriend() {
    if (!profile) return
    send({ action: 'sendRequest', friendId: profile.playerId })
  }

  if (!profile) {
    return <div className="p-6 text-center text-sm text-mute">{t('profile.loading')}</div>
  }

  return (
    <div className="mx-auto max-w-md p-6">
      <div className="rounded-2xl border border-ink-line bg-ink-raised p-6">
        <div className="flex items-center justify-between gap-3">
          <h1 className="font-display text-2xl font-medium text-paper">{profile.username}</h1>
          {!isOwnProfile &&
            (friendStatus === 'sent' ? (
              <span className="flex items-center gap-1.5 text-sm text-lime">
                <IconCheck className="h-4 w-4" />
                {t('profile.friendRequestSent')}
              </span>
            ) : (
              <button
                onClick={addFriend}
                className="flex items-center gap-1.5 rounded-full bg-lime px-3 py-1.5 text-sm font-semibold text-ink hover:opacity-90"
              >
                <IconUserPlus className="h-4 w-4" />
                {t('profile.addFriend')}
              </button>
            ))}
        </div>
        {friendStatus === 'error' && <p className="mt-2 text-xs text-ember">{t('profile.friendRequestFailed')}</p>}
        {profile.visible ? (
          <dl className="mt-4 grid grid-cols-2 gap-y-3 text-sm">
            <dt className="text-mute">{t('profile.rating')}</dt>
            <dd className="text-right font-mono font-tabular text-lime">{profile.rating}</dd>
            <dt className="text-mute">{t('profile.wins')}</dt>
            <dd className="text-right font-mono font-tabular text-paper">{profile.wins}</dd>
            <dt className="text-mute">{t('profile.losses')}</dt>
            <dd className="text-right font-mono font-tabular text-paper">{profile.losses}</dd>
            <dt className="text-mute">{t('profile.draws')}</dt>
            <dd className="text-right font-mono font-tabular text-paper">{profile.draws}</dd>
          </dl>
        ) : (
          <p className="mt-4 text-sm text-mute">{t('profile.private')}</p>
        )}
      </div>
    </div>
  )
}
