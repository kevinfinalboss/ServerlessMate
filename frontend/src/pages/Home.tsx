import { useNavigate } from 'react-router-dom'
import { Wordmark } from '../components/Wordmark'
import { LanguageToggle } from '../components/LanguageToggle'
import { AmbientBoard } from '../components/AmbientBoard'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'

const FEATURES = [
  { key: 'live' as const },
  { key: 'practice' as const },
  { key: 'ranked' as const },
]

export function Home() {
  const navigate = useNavigate()
  const { connect } = useGameSocket()
  const { t } = useTranslation()

  function playAsGuest() {
    connect()
    navigate('/play')
  }

  return (
    <div className="min-h-screen bg-ink">
      <header className="flex items-center justify-between px-4 py-5 sm:px-8">
        <Wordmark />
        <div className="flex items-center gap-2 sm:gap-3">
          <LanguageToggle />
          <button
            onClick={() => navigate('/login')}
            className="rounded-full px-3 py-1.5 text-sm font-medium text-paper hover:text-lime"
          >
            {t('common.login')}
          </button>
          <button
            onClick={() => navigate('/signup')}
            className="rounded-full bg-felt px-4 py-1.5 text-sm font-medium text-paper transition-colors hover:bg-felt-dim"
          >
            {t('common.signup')}
          </button>
        </div>
      </header>

      <main className="mx-auto flex max-w-6xl flex-col-reverse items-center gap-12 px-6 py-12 lg:flex-row lg:items-center lg:justify-between lg:py-24">
        <div className="max-w-xl text-center lg:text-left">
          <h1 className="font-display text-4xl leading-[1.05] font-medium text-paper sm:text-5xl lg:text-6xl">
            {t('home.headline')}
          </h1>
          <p className="mt-5 text-base leading-relaxed text-mute sm:text-lg">{t('home.subhead')}</p>

          <div className="mt-8 flex flex-col items-center gap-2 lg:items-start">
            <button
              onClick={playAsGuest}
              className="rounded-full bg-lime px-7 py-3 text-base font-semibold text-ink shadow-lg shadow-lime/20 transition-transform hover:scale-[1.02] active:scale-[0.98]"
            >
              {t('home.ctaGuest')}
            </button>
            <span className="text-xs text-mute">{t('home.ctaGuestHint')}</span>
          </div>
        </div>

        <AmbientBoard />
      </main>

      <section className="border-t border-ink-line">
        <div className="mx-auto grid max-w-6xl gap-8 px-6 py-14 sm:grid-cols-3 sm:px-8">
          {FEATURES.map((feature) => (
            <div key={feature.key}>
              <p className="font-display text-sm font-semibold tracking-wide text-lime uppercase">
                {t(`home.feature.${feature.key}.label`)}
              </p>
              <p className="mt-2 text-sm leading-relaxed text-mute">
                {t(`home.feature.${feature.key}.desc`)}
              </p>
            </div>
          ))}
        </div>
      </section>

      <footer className="border-t border-ink-line px-6 py-8 text-center sm:px-8">
        <p className="text-xs text-mute">{t('home.footer.tagline')}</p>
      </footer>
    </div>
  )
}
