import type { ReactNode } from 'react'
import { Wordmark } from './Wordmark'
import { LanguageToggle } from './LanguageToggle'

export function AuthShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col bg-ink">
      <header className="flex items-center justify-between px-4 py-5 sm:px-8">
        <Wordmark />
        <LanguageToggle />
      </header>
      <main className="flex flex-1 items-center justify-center px-4 pb-16">
        <div className="w-full max-w-sm">{children}</div>
      </main>
    </div>
  )
}
