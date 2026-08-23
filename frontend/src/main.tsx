import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import { App } from './App.tsx'
import { GameSocketProvider } from './lib/GameSocketProvider.tsx'
import { LanguageProvider } from './lib/i18n.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <LanguageProvider>
        <GameSocketProvider>
          <App />
        </GameSocketProvider>
      </LanguageProvider>
    </BrowserRouter>
  </StrictMode>,
)
