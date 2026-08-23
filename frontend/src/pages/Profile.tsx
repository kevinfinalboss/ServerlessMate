import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import { isProfileResponse } from '../lib/types'
import type { ProfileResponse } from '../lib/types'

export function Profile() {
  const { playerId } = useParams()
  const { connected, lastMessage, send } = useGameSocket()
  const { t } = useTranslation()
  const [profile, setProfile] = useState<ProfileResponse | null>(null)

  useEffect(() => {
    if (connected) send({ action: 'getProfile', playerId })
  }, [connected, playerId, send])

  useEffect(() => {
    if (lastMessage && isProfileResponse(lastMessage)) {
      setProfile(lastMessage)
    }
  }, [lastMessage])

  if (!profile) {
    return <div className="p-6 text-center text-sm text-mute">{t('profile.loading')}</div>
  }

  return (
    <div className="mx-auto max-w-md p-6">
      <div className="rounded-2xl border border-ink-line bg-ink-raised p-6">
        <h1 className="font-display text-2xl font-medium text-paper">{profile.username}</h1>
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
