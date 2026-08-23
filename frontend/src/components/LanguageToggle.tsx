import { useTranslation } from '../lib/i18n'
import type { Language } from '../lib/i18n'

const OPTIONS: Language[] = ['en', 'pt']

export function LanguageToggle() {
  const { language, setLanguage } = useTranslation()

  return (
    <div className="inline-flex items-center rounded-full border border-ink-line bg-ink-raised p-0.5 text-xs font-semibold tracking-wide">
      {OPTIONS.map((option) => (
        <button
          key={option}
          type="button"
          onClick={() => setLanguage(option)}
          aria-pressed={language === option}
          className={`rounded-full px-2.5 py-1 transition-colors ${
            language === option ? 'bg-lime text-ink' : 'text-mute hover:text-paper'
          }`}
        >
          {option.toUpperCase()}
        </button>
      ))}
    </div>
  )
}
