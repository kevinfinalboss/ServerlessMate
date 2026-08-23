import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { AuthShell } from '../components/AuthShell'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'

export function Signup() {
  const navigate = useNavigate()
  const { connect } = useGameSocket()
  const { t } = useTranslation()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [notice, setNotice] = useState(false)

  function submit(e: FormEvent) {
    e.preventDefault()
    setNotice(true)
  }

  function playAsGuest() {
    connect()
    navigate('/play')
  }

  return (
    <AuthShell>
      <div className="rounded-2xl border border-ink-line bg-ink-raised p-8">
        <h1 className="font-display text-2xl font-medium text-paper">{t('signup.title')}</h1>
        <p className="mt-1 text-sm text-mute">{t('signup.subtitle')}</p>

        <form onSubmit={submit} className="mt-6 flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-left text-xs font-medium text-mute">
            {t('signup.username')}
            <input
              type="text"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="rounded-lg border border-ink-line bg-ink px-3 py-2 text-sm text-paper outline-none focus:border-lime"
            />
          </label>
          <label className="flex flex-col gap-1 text-left text-xs font-medium text-mute">
            {t('signup.email')}
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="rounded-lg border border-ink-line bg-ink px-3 py-2 text-sm text-paper outline-none focus:border-lime"
            />
          </label>
          <label className="flex flex-col gap-1 text-left text-xs font-medium text-mute">
            {t('signup.password')}
            <input
              type="password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="rounded-lg border border-ink-line bg-ink px-3 py-2 text-sm text-paper outline-none focus:border-lime"
            />
          </label>
          <button
            type="submit"
            className="mt-2 rounded-full bg-lime px-4 py-2.5 text-sm font-semibold text-ink transition-transform hover:scale-[1.01] active:scale-[0.98]"
          >
            {t('signup.submit')}
          </button>
        </form>

        {notice && (
          <p className="mt-4 rounded-lg border border-ember/40 bg-ember/10 px-3 py-2 text-xs leading-relaxed text-ember">
            {t('signup.notConnected')}
          </p>
        )}

        <div className="mt-6 border-t border-ink-line pt-4 text-center text-xs text-mute">
          {t('signup.hasAccount')}{' '}
          <button
            type="button"
            onClick={() => navigate('/login')}
            className="font-medium text-lime hover:underline"
          >
            {t('signup.loginLink')}
          </button>
        </div>
      </div>

      <button
        type="button"
        onClick={playAsGuest}
        className="mt-4 w-full text-center text-xs text-mute hover:text-paper"
      >
        {t('common.playAsGuest')} →
      </button>
    </AuthShell>
  )
}
