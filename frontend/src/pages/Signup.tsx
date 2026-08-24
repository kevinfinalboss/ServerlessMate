import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { AuthShell } from '../components/AuthShell'
import { confirmSignUp, mapCognitoError, resendConfirmationCode, signIn, signUp } from '../lib/cognito'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import type { TranslationKey } from '../lib/i18n'
import { setToken } from '../lib/session'

export function Signup() {
  const navigate = useNavigate()
  const { connect } = useGameSocket()
  const { t } = useTranslation()
  const [step, setStep] = useState<'form' | 'confirm'>('form')
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<TranslationKey | null>(null)
  const [resent, setResent] = useState(false)

  async function loginAfterConfirm() {
    const { idToken } = await signIn(email, password)
    setToken(idToken)
    connect()
    navigate('/play')
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const { confirmed } = await signUp(email, password, username)
      if (confirmed) {
        await loginAfterConfirm()
      } else {
        setStep('confirm')
      }
    } catch (err) {
      setError(mapCognitoError(err))
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
      await loginAfterConfirm()
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

  function playAsGuest() {
    connect()
    navigate('/play')
  }

  return (
    <AuthShell>
      <div className="rounded-2xl border border-ink-line bg-ink-raised p-8">
        {step === 'form' ? (
          <>
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
                <span className="text-[11px] font-normal text-mute">{t('signup.passwordHint')}</span>
              </label>
              <button
                type="submit"
                disabled={loading}
                className="mt-2 rounded-full bg-lime px-4 py-2.5 text-sm font-semibold text-ink transition-transform hover:scale-[1.01] active:scale-[0.98] disabled:opacity-60"
              >
                {loading ? t('signup.submitting') : t('signup.submit')}
              </button>
            </form>

            {error && (
              <p className="mt-4 rounded-lg border border-ember/40 bg-ember/10 px-3 py-2 text-xs leading-relaxed text-ember">
                {t(error)}
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
          </>
        ) : (
          <>
            <h1 className="font-display text-2xl font-medium text-paper">{t('auth.confirmTitle')}</h1>
            <p className="mt-1 text-sm text-mute">{t('auth.confirmSubtitle')}</p>

            <form onSubmit={submitConfirm} className="mt-6 flex flex-col gap-3">
              <label className="flex flex-col gap-1 text-left text-xs font-medium text-mute">
                {t('auth.confirmCodeLabel')}
                <input
                  type="text"
                  required
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  className="rounded-lg border border-ink-line bg-ink px-3 py-2 text-sm text-paper outline-none focus:border-lime"
                />
              </label>
              <button
                type="submit"
                disabled={loading}
                className="mt-2 rounded-full bg-lime px-4 py-2.5 text-sm font-semibold text-ink transition-transform hover:scale-[1.01] active:scale-[0.98] disabled:opacity-60"
              >
                {loading ? t('login.submitting') : t('auth.confirmSubmit')}
              </button>
            </form>

            {error && (
              <p className="mt-4 rounded-lg border border-ember/40 bg-ember/10 px-3 py-2 text-xs leading-relaxed text-ember">
                {t(error)}
              </p>
            )}
            {resent && (
              <p className="mt-4 rounded-lg border border-lime/40 bg-lime/10 px-3 py-2 text-xs leading-relaxed text-lime">
                {t('auth.resendSent')}
              </p>
            )}

            <button
              type="button"
              onClick={resend}
              className="mt-4 text-xs text-mute underline decoration-dotted hover:text-paper"
            >
              {t('auth.resendCode')}
            </button>
          </>
        )}
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
