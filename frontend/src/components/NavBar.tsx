import { NavLink } from 'react-router-dom'
import { Wordmark } from './Wordmark'
import { LanguageToggle } from './LanguageToggle'
import { useTranslation } from '../lib/i18n'

const linkClass = ({ isActive }: { isActive: boolean }) =>
  `rounded-full px-3 py-1.5 text-sm font-medium transition-colors ${
    isActive ? 'bg-felt text-paper' : 'text-mute hover:text-paper'
  }`

export function NavBar() {
  const { t } = useTranslation()

  return (
    <nav className="flex items-center justify-between gap-4 border-b border-ink-line px-4 py-3 sm:px-6">
      <div className="flex items-center gap-6">
        <Wordmark />
        <div className="hidden items-center gap-1 sm:flex">
          <NavLink to="/play" className={linkClass}>
            {t('nav.play')}
          </NavLink>
          <NavLink to="/leaderboard" className={linkClass}>
            {t('nav.leaderboard')}
          </NavLink>
          <NavLink to="/history" className={linkClass}>
            {t('nav.history')}
          </NavLink>
          <NavLink to="/profile" className={linkClass}>
            {t('nav.profile')}
          </NavLink>
        </div>
      </div>
      <LanguageToggle />
    </nav>
  )
}
