import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import { getPlayerId } from '../lib/session'
import { isProfileResponse, isType } from '../lib/types'
import type { ErrorMessage, FriendEventMessage, ProfileResponse } from '../lib/types'
import { IconUserPlus, IconCheck } from '../components/icons'

const FRIEND_EVENT_TYPES: FriendEventMessage['type'][] = [
  'friendRequestSent',
  'friendRequestAccepted',
  'playerBlocked',
  'friendRequestCancelled',
]

export function Profile() {
  const { playerId } = useParams()
  const { identity, connected, lastMessage, send } = useGameSocket()
  const { t } = useTranslation()
  const [profile, setProfile] = useState<ProfileResponse | null>(null)
  const [visibility, setVisibility] = useState<'public' | 'friends'>('public')
  const [birthDate, setBirthDate] = useState('')
  const [country, setCountry] = useState('')
  const [github, setGithub] = useState('')
  const [linkedIn, setLinkedIn] = useState('')
  const [saved, setSaved] = useState(false)
  const [friendError, setFriendError] = useState<string | null>(null)

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
    if (FRIEND_EVENT_TYPES.some((type) => isType<FriendEventMessage>(lastMessage, type))) {
      setFriendError(null)
      send({ action: 'getProfile', playerId })
      return
    }
    if (isType<ErrorMessage>(lastMessage, 'error')) {
      setFriendError(lastMessage.message)
    }
  }, [lastMessage, playerId, send])

  useEffect(() => {
    if (!profile || !isOwnProfile) return
    setVisibility(profile.visibility === 'friends' ? 'friends' : 'public')
    setBirthDate(profile.birthDate ?? '')
    setCountry(profile.country ?? '')
    setGithub(profile.github ?? '')
    setLinkedIn(profile.linkedIn ?? '')
  }, [profile, isOwnProfile])

  function addFriend() {
    if (!profile) return
    send({ action: 'sendRequest', friendId: profile.playerId })
  }

  function acceptRequest() {
    if (!profile) return
    send({ action: 'acceptRequest', friendId: profile.playerId })
  }

  function saveSettings() {
    send({ action: 'updateProfile', visibility, birthDate, country, github, linkedIn })
    setSaved(true)
  }

  if (!profile) {
    return <div className="p-6 text-center text-sm text-mute">{t('profile.loading')}</div>
  }

  return (
    <div className="mx-auto max-w-md p-6">
      <div className="rounded-2xl border border-ink-line bg-ink-raised p-6">
        <div className="flex items-center justify-between gap-3">
          <h1 className="font-display text-2xl font-medium text-paper">{profile.username}</h1>
          {!isOwnProfile && <FriendAction status={profile.friendshipStatus} onAdd={addFriend} onAccept={acceptRequest} t={t} />}
        </div>
        {friendError && <p className="mt-2 text-xs text-ember">{friendError}</p>}

        {profile.visible ? (
          <>
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

            {(profile.country || profile.github || profile.linkedIn || profile.birthDate) && (
              <dl className="mt-4 flex flex-col gap-2 border-t border-ink-line pt-4 text-sm">
                {profile.country && (
                  <div className="flex justify-between">
                    <dt className="text-mute">{t('profile.country')}</dt>
                    <dd className="text-paper">{profile.country}</dd>
                  </div>
                )}
                {profile.birthDate && (
                  <div className="flex justify-between">
                    <dt className="text-mute">{t('profile.birthDate')}</dt>
                    <dd className="font-mono text-paper">{profile.birthDate}</dd>
                  </div>
                )}
                {profile.github && (
                  <div className="flex justify-between">
                    <dt className="text-mute">GitHub</dt>
                    <dd>
                      <a
                        href={`https://github.com/${profile.github}`}
                        target="_blank"
                        rel="noreferrer"
                        className="text-lime hover:underline"
                      >
                        {profile.github}
                      </a>
                    </dd>
                  </div>
                )}
                {profile.linkedIn && (
                  <div className="flex justify-between">
                    <dt className="text-mute">LinkedIn</dt>
                    <dd>
                      <a
                        href={`https://linkedin.com/in/${profile.linkedIn}`}
                        target="_blank"
                        rel="noreferrer"
                        className="text-lime hover:underline"
                      >
                        {profile.linkedIn}
                      </a>
                    </dd>
                  </div>
                )}
              </dl>
            )}
          </>
        ) : (
          <p className="mt-4 text-sm text-mute">{t('profile.private')}</p>
        )}
      </div>

      {isOwnProfile && (
        <div className="mt-4 rounded-2xl border border-ink-line bg-ink-raised p-6">
          <h2 className="mb-4 font-display text-lg font-medium text-paper">{t('profile.settings.title')}</h2>

          <label className="mb-4 block text-sm">
            <span className="mb-1 block text-mute">{t('profile.settings.visibility')}</span>
            <select
              value={visibility}
              onChange={(e) => setVisibility(e.target.value as 'public' | 'friends')}
              className="w-full rounded-lg border border-ink-line bg-ink px-3 py-2 text-paper outline-none focus:border-lime"
            >
              <option value="public">{t('profile.settings.visibilityPublic')}</option>
              <option value="friends">{t('profile.settings.visibilityFriends')}</option>
            </select>
          </label>

          <p className="mb-2 text-xs text-mute">{t('profile.settings.optionalHint')}</p>

          <label className="mb-3 block text-sm">
            <span className="mb-1 block text-mute">{t('profile.birthDate')}</span>
            <input
              type="date"
              value={birthDate}
              onChange={(e) => setBirthDate(e.target.value)}
              className="w-full rounded-lg border border-ink-line bg-ink px-3 py-2 text-paper outline-none focus:border-lime"
            />
          </label>

          <label className="mb-3 block text-sm">
            <span className="mb-1 block text-mute">{t('profile.country')}</span>
            <input
              value={country}
              onChange={(e) => setCountry(e.target.value)}
              placeholder={t('profile.countryPlaceholder')}
              className="w-full rounded-lg border border-ink-line bg-ink px-3 py-2 text-paper outline-none focus:border-lime"
            />
          </label>

          <label className="mb-3 block text-sm">
            <span className="mb-1 block text-mute">GitHub</span>
            <input
              value={github}
              onChange={(e) => setGithub(e.target.value)}
              placeholder="octocat"
              className="w-full rounded-lg border border-ink-line bg-ink px-3 py-2 text-paper outline-none focus:border-lime"
            />
          </label>

          <label className="mb-4 block text-sm">
            <span className="mb-1 block text-mute">LinkedIn</span>
            <input
              value={linkedIn}
              onChange={(e) => setLinkedIn(e.target.value)}
              placeholder="janedoe"
              className="w-full rounded-lg border border-ink-line bg-ink px-3 py-2 text-paper outline-none focus:border-lime"
            />
          </label>

          <div className="flex items-center gap-3">
            <button
              onClick={() => {
                saveSettings()
              }}
              className="rounded-full bg-lime px-4 py-2 text-sm font-semibold text-ink hover:opacity-90"
            >
              {t('profile.settings.save')}
            </button>
            {saved && <span className="text-sm text-lime">{t('profile.settings.saved')}</span>}
          </div>
        </div>
      )}
    </div>
  )
}

function FriendAction({
  status,
  onAdd,
  onAccept,
  t,
}: {
  status: ProfileResponse['friendshipStatus']
  onAdd: () => void
  onAccept: () => void
  t: (key: Parameters<ReturnType<typeof useTranslation>['t']>[0]) => string
}) {
  if (status === 'accepted') {
    return (
      <span className="flex items-center gap-1.5 text-sm text-lime">
        <IconCheck className="h-4 w-4" />
        {t('profile.friends')}
      </span>
    )
  }
  if (status === 'pendingOutgoing') {
    return <span className="text-sm text-mute">{t('profile.friendRequestSent')}</span>
  }
  if (status === 'pendingIncoming') {
    return (
      <button
        onClick={onAccept}
        className="flex items-center gap-1.5 rounded-full bg-lime px-3 py-1.5 text-sm font-semibold text-ink hover:opacity-90"
      >
        <IconCheck className="h-4 w-4" />
        {t('profile.acceptRequest')}
      </button>
    )
  }
  if (status === 'blocked') {
    return null
  }
  return (
    <button
      onClick={onAdd}
      className="flex items-center gap-1.5 rounded-full bg-lime px-3 py-1.5 text-sm font-semibold text-ink hover:opacity-90"
    >
      <IconUserPlus className="h-4 w-4" />
      {t('profile.addFriend')}
    </button>
  )
}
