import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useGameSocket } from '../lib/GameSocketProvider'
import { isProfileResponse } from '../lib/types'
import type { ProfileResponse } from '../lib/types'

export function Profile() {
  const { playerId } = useParams()
  const { connected, lastMessage, send } = useGameSocket()
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
    return <div className="p-6 text-center text-gray-400">Carregando perfil...</div>
  }

  return (
    <div className="mx-auto max-w-md p-6">
      <h1 className="mb-4 text-xl font-semibold text-white">{profile.username}</h1>
      {profile.visible ? (
        <dl className="grid grid-cols-2 gap-2 text-sm text-gray-300">
          <dt className="text-gray-500">Rating</dt>
          <dd>{profile.rating}</dd>
          <dt className="text-gray-500">Vitórias</dt>
          <dd>{profile.wins}</dd>
          <dt className="text-gray-500">Derrotas</dt>
          <dd>{profile.losses}</dd>
          <dt className="text-gray-500">Empates</dt>
          <dd>{profile.draws}</dd>
        </dl>
      ) : (
        <p className="text-sm text-gray-500">Este perfil é privado.</p>
      )}
    </div>
  )
}
