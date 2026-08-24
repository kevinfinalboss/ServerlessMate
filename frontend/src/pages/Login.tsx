import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { AuthShell } from '../components/AuthShell'
import { CognitoError, confirmSignUp, mapCognitoError, resendConfirmationCode, signIn } from '../lib/cognito'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import type { TranslationKey } from '../lib/i18n'
import { setToken } from '../lib/session'

export function Login() {
  const navigate = useNavigate()
  const { connect } = useGameSocket()
  const { t } = useTranslation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showTokenField, setShowTokenField] = useState(false)
  const [tokenInput, setTokenInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<TranslationKey | null>(null)
  const [needsConfirmation, setNeedsConfirmation] = useState(false)
  const [code, setCode] = useState('')
  const [resent, setResent] = useState(false)

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setNeedsConfirmation(false)
    setLoading(true)
    try {
      const { idToken } = await signIn(email, password)
      setToken(idToken)
      connect()
      navigate('/play')
    } catch (err) {
      if (err instanceof CognitoError && err.type === 'UserNotConfirmedException') {
        setNeedsConfirmation(true)
      } else {
        setError(mapCognitoError(err))
      }
    } finally {
      setLoading(false)
    }
  }

  async function submitConfirm(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await confirmSignUp(email, code)
      const { idToken } = await signIn(email, password)
      setToken(idToken)
      connect()
      navigate('/play')
    } catch (err) {
      setError(mapCognitoError(err))
    } finally {
      setLoading(false)
    }
  }

  async function resend() {
    setError(null)
    setResent(false)
    try {
      await resendConfirmationCode(email)
      setResent(true)
    } catch (err) {
      setError(mapCognitoError(err))
    }
  }

  function useToken() {
    if (!tokenInput.trim()) return
    setToken(tokenInput.trim())
    connect()
    navigate('/play')
  }

  function playAsGuest() {
    connect()
    navigate('/play')
  }

  return (
    <AuthShell>
      <div className="rounded-2xl border border-ink-line bg-ink-raised p-8">
        <h1 className="font-display text-2xl font-medium text-paper">{t('login.title')}</h1>
        <p className="mt-1 text-sm text-mute">{t('login.subtitle')}</p>

        <form onSubmit={submit} className="mt-6 flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-left text-xs font-medium text-mute">
            {t('login.email')}
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="rounded-lg border border-ink-line bg-ink px-3 py-2 text-sm text-paper outline-none focus:border-lime"
            />
          </label>
          <label className="flex flex-col gap-1 text-left text-xs font-medium text-mute">
            {t('login.password')}
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="rounded-lg border border-ink-line bg-ink px-3 py-2 text-sm text-paper outline-none focus:border-lime"
            />
          </label>
          <button
            type="submit"
            disabled={loading}
            className="mt-2 rounded-full bg-lime px-4 py-2.5 text-sm font-semibold text-ink transition-transform hover:scale-[1.01] active:scale-[0.98] disabled:opacity-60"
          >
            {loading ? t('login.submitting') : t('login.submit')}
          </button>
        </form>

        {error && (
          <p className="mt-4 rounded-lg border border-ember/40 bg-ember/10 px-3 py-2 text-xs leading-relaxed text-ember">
            {t(error)}
          </p>
        )}

        {needsConfirmation && (
          <div className="mt-4 rounded-lg border border-ink-line bg-ink px-3 py-3">
            <p className="text-xs text-mute">{t('login.needsConfirmation')}</p>
            <form onSubmit={submitConfirm} className="mt-3 flex flex-col gap-2">
              <label className="flex flex-col gap-1 text-left text-xs font-medium text-mute">
                {t('auth.confirmCodeLabel')}
                <input
                  type="text"
                  required
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  className="rounded-lg border border-ink-line bg-ink-raised px-3 py-2 text-sm text-paper outline-none focus:border-lime"
                />
              </label>
              <button
                type="submit"
                disabled={loading}
                className="rounded-full bg-lime px-4 py-2 text-sm font-semibold text-ink transition-transform hover:scale-[1.01] active:scale-[0.98] disabled:opacity-60"
              >
                {t('auth.confirmSubmit')}
              </button>
            </form>
            {resent && <p className="mt-2 text-xs text-lime">{t('auth.resendSent')}</p>}
            <button
              type="button"
              onClick={resend}
              className="mt-2 text-xs text-mute underline decoration-dotted hover:text-paper"
            >
              {t('auth.resendCode')}
            </button>
          </div>
        )}

        <button
          type="button"
          onClick={() => setShowTokenField((v) => !v)}
          className="mt-4 text-xs text-mute underline decoration-dotted hover:text-paper"
        >
          {t('login.tokenToggle')}
        </button>
        {showTokenField && (
          <div className="mt-2 flex gap-2">
            <input
              value={tokenInput}
              onChange={(e) => setTokenInput(e.target.value)}
              placeholder={t('login.tokenLabel')}
              className="flex-1 rounded-lg border border-ink-line bg-ink px-3 py-2 text-xs text-paper outline-none focus:border-lime"
            />
            <button
              type="button"
              onClick={useToken}
              className="rounded-lg bg-ink-line px-3 py-2 text-xs font-medium text-paper hover:bg-felt"
            >
              {t('login.tokenSubmit')}
            </button>
          </div>
        )}

        <div className="mt-6 border-t border-ink-line pt-4 text-center text-xs text-mute">
          {t('login.noAccount')}{' '}
          <button
            type="button"
            onClick={() => navigate('/signup')}
            className="font-medium text-lime hover:underline"
          >
            {t('login.signupLink')}
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
