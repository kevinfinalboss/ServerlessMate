import { NavLink, useNavigate } from 'react-router-dom'
import { Wordmark } from './Wordmark'
import { LanguageToggle } from './LanguageToggle'
import { IconPlay, IconTrophy, IconHistory, IconUsers, IconUser, IconLogOut } from './icons'
import { useTranslation } from '../lib/i18n'
import { clearActiveGameId, clearToken } from '../lib/session'
import { socket } from '../lib/socket'

const NAV_ITEMS = [
  { to: '/play', labelKey: 'nav.play' as const, Icon: IconPlay },
  { to: '/friends', labelKey: 'nav.friends' as const, Icon: IconUsers },
  { to: '/leaderboard', labelKey: 'nav.leaderboard' as const, Icon: IconTrophy },
  { to: '/history', labelKey: 'nav.history' as const, Icon: IconHistory },
  { to: '/profile', labelKey: 'nav.profile' as const, Icon: IconUser },
]

export function Sidebar() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  function logout() {
    socket.disconnect()
    clearToken()
    clearActiveGameId()
    navigate('/')
  }

  return (
    <>
      <aside className="sticky top-0 hidden h-screen w-60 shrink-0 flex-col border-r border-ink-line bg-ink-raised px-4 py-6 sm:flex">
        <Wordmark className="px-2" />
        <nav className="mt-8 flex flex-1 flex-col gap-1">
          {NAV_ITEMS.map(({ to, labelKey, Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-xl border-l-2 px-3 py-2.5 text-sm font-medium transition-colors ${
                  isActive
                    ? 'border-lime bg-felt/25 text-paper'
                    : 'border-transparent text-mute hover:border-ink-line hover:text-paper'
                }`
              }
            >
              <Icon className="h-[18px] w-[18px] shrink-0" />
              {t(labelKey)}
            </NavLink>
          ))}
        </nav>
        <div className="flex items-center justify-between gap-2 border-t border-ink-line pt-4">
          <LanguageToggle />
          <button
            onClick={logout}
            aria-label={t('common.logout')}
            title={t('common.logout')}
            className="rounded-lg p-2 text-mute transition-colors hover:bg-ink-line hover:text-paper"
          >
            <IconLogOut className="h-[18px] w-[18px]" />
          </button>
        </div>
      </aside>

      <div className="flex items-center justify-between border-b border-ink-line px-4 py-3 sm:hidden">
        <Wordmark />
        <div className="flex items-center gap-2">
          <LanguageToggle />
          <button
            onClick={logout}
            aria-label={t('common.logout')}
            className="rounded-lg p-2 text-mute hover:bg-ink-line hover:text-paper"
          >
            <IconLogOut className="h-[18px] w-[18px]" />
          </button>
        </div>
      </div>

      <nav className="fixed inset-x-0 bottom-0 z-10 flex items-center justify-around border-t border-ink-line bg-ink-raised py-2 sm:hidden">
        {NAV_ITEMS.map(({ to, labelKey, Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex flex-col items-center gap-0.5 px-3 py-1 text-[11px] font-medium ${
                isActive ? 'text-lime' : 'text-mute'
              }`
            }
          >
            <Icon className="h-5 w-5" />
            {t(labelKey)}
          </NavLink>
        ))}
      </nav>
    </>
  )
}
