import { createContext, useContext, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

export type Language = 'en' | 'pt'

const STORAGE_KEY = 'sm.lang'

const en = {
  'common.playAsGuest': 'Play as Guest',
  'common.login': 'Log in',
  'common.signup': 'Sign up',
  'common.logout': 'Log out',
  'common.back': 'Back',

  'home.headline': 'Chess that scales to zero.',
  'home.subhead':
    'Real-time multiplayer chess running on AWS Lambda and DynamoDB — nothing idling between moves. Play a friend, a stranger, or the engine.',
  'home.ctaGuest': 'Play as Guest',
  'home.ctaGuestHint': 'Free, no account needed',
  'home.feature.live.label': 'Live',
  'home.feature.live.desc': 'Moves sync over WebSockets the instant you make them.',
  'home.feature.practice.label': 'Practice',
  'home.feature.practice.desc': 'Two engine strengths, with commentary after every move.',
  'home.feature.ranked.label': 'Ranked',
  'home.feature.ranked.desc': 'Standard Elo rating and a global leaderboard.',
  'home.footer.tagline': 'A serverless chess platform, built to demonstrate real-time AWS architecture.',

  'nav.play': 'Play',
  'nav.leaderboard': 'Leaderboard',
  'nav.history': 'History',
  'nav.profile': 'Profile',

  'play.connectionStatus': 'Connection',
  'play.connected': 'connected',
  'play.disconnected': 'disconnected',
  'play.guestNotice': 'Guests can play another human, but not the engine, and games don’t affect rating.',
  'play.vsHuman.title': 'Play another player',
  'play.vsHuman.cta': 'Find a match',
  'play.vsAI.title': 'Play the engine',
  'play.vsAI.loginRequired': 'Log in to play the engine.',
  'play.vsAI.easy': 'Easy',
  'play.vsAI.hard': 'Hard',
  'play.vsAI.cta': 'Play',
  'play.status.searching': 'Looking for an opponent…',
  'play.status.creatingAI': 'Setting up your game…',

  'login.title': 'Welcome back',
  'login.subtitle': 'Log in to track your rating and play the engine.',
  'login.email': 'Email',
  'login.password': 'Password',
  'login.submit': 'Log in',
  'login.notConnected':
    'Cognito isn’t deployed yet, so this form can’t authenticate anyone. Play as a guest to try the app now.',
  'login.tokenToggle': 'Have a token instead?',
  'login.tokenLabel': 'Paste a Cognito ID token',
  'login.tokenSubmit': 'Use token',
  'login.noAccount': 'New here?',
  'login.signupLink': 'Create an account',

  'signup.title': 'Create your account',
  'signup.subtitle': 'Track your rating, build a friends list, and play the engine.',
  'signup.username': 'Username',
  'signup.email': 'Email',
  'signup.password': 'Password',
  'signup.submit': 'Sign up',
  'signup.notConnected':
    'Cognito isn’t deployed yet, so accounts can’t be created yet. Play as a guest to try the app now.',
  'signup.hasAccount': 'Already have an account?',
  'signup.loginLink': 'Log in',

  'game.loading': 'Loading game…',
  'game.white': 'White',
  'game.black': 'Black',
  'game.resign': 'Resign',
  'game.offerDraw': 'Offer draw',
  'game.acceptDraw': 'Accept draw',
  'game.over': 'Game over',
  'game.winner': 'Winner',
  'game.backToLobby': 'Back to lobby',
  'game.chatPlaceholder': 'Message',
  'game.send': 'Send',
  'game.noMessages': 'No messages yet.',

  'leaderboard.title': 'Leaderboard',
  'leaderboard.empty': 'No data yet.',

  'history.title': 'History',
  'history.empty': 'No games played yet.',
  'history.viewPgn': 'View PGN',
  'history.vsAI': 'Engine',
  'history.result.win': 'win',
  'history.result.loss': 'loss',
  'history.result.draw': 'draw',

  'profile.loading': 'Loading profile…',
  'profile.private': 'This profile is private.',
  'profile.rating': 'Rating',
  'profile.wins': 'Wins',
  'profile.losses': 'Losses',
  'profile.draws': 'Draws',
} satisfies Record<string, string>

const pt: Record<keyof typeof en, string> = {
  'common.playAsGuest': 'Jogar como convidado',
  'common.login': 'Entrar',
  'common.signup': 'Cadastrar',
  'common.logout': 'Sair',
  'common.back': 'Voltar',

  'home.headline': 'Xadrez que escala a zero.',
  'home.subhead':
    'Xadrez multiplayer em tempo real rodando em AWS Lambda e DynamoDB — nada fica ligado entre os lances. Jogue com um amigo, um estranho ou o motor.',
  'home.ctaGuest': 'Jogar como convidado',
  'home.ctaGuestHint': 'Grátis, sem precisar de conta',
  'home.feature.live.label': 'Tempo real',
  'home.feature.live.desc': 'Lances sincronizam via WebSocket no instante em que você joga.',
  'home.feature.practice.label': 'Treino',
  'home.feature.practice.desc': 'Dois níveis de motor, com comentário depois de cada lance.',
  'home.feature.ranked.label': 'Ranqueado',
  'home.feature.ranked.desc': 'Rating Elo padrão e um leaderboard global.',
  'home.footer.tagline': 'Uma plataforma de xadrez serverless, feita pra demonstrar arquitetura AWS em tempo real.',

  'nav.play': 'Jogar',
  'nav.leaderboard': 'Ranking',
  'nav.history': 'Histórico',
  'nav.profile': 'Perfil',

  'play.connectionStatus': 'Conexão',
  'play.connected': 'conectado',
  'play.disconnected': 'desconectado',
  'play.guestNotice': 'Convidados podem jogar contra outro humano, mas não contra o motor, e a partida não afeta rating.',
  'play.vsHuman.title': 'Jogar contra outro jogador',
  'play.vsHuman.cta': 'Entrar na fila',
  'play.vsAI.title': 'Jogar contra o motor',
  'play.vsAI.loginRequired': 'Faça login pra jogar contra o motor.',
  'play.vsAI.easy': 'Fácil',
  'play.vsAI.hard': 'Difícil',
  'play.vsAI.cta': 'Jogar',
  'play.status.searching': 'Procurando adversário…',
  'play.status.creatingAI': 'Preparando sua partida…',

  'login.title': 'Bem-vindo de volta',
  'login.subtitle': 'Entre pra acompanhar seu rating e jogar contra o motor.',
  'login.email': 'Email',
  'login.password': 'Senha',
  'login.submit': 'Entrar',
  'login.notConnected':
    'O Cognito ainda não foi implantado, então esse formulário não autentica ninguém de verdade. Jogue como convidado pra testar o app agora.',
  'login.tokenToggle': 'Tem um token?',
  'login.tokenLabel': 'Cole um ID token do Cognito',
  'login.tokenSubmit': 'Usar token',
  'login.noAccount': 'Novo por aqui?',
  'login.signupLink': 'Criar uma conta',

  'signup.title': 'Crie sua conta',
  'signup.subtitle': 'Acompanhe seu rating, monte uma lista de amigos e jogue contra o motor.',
  'signup.username': 'Nome de usuário',
  'signup.email': 'Email',
  'signup.password': 'Senha',
  'signup.submit': 'Cadastrar',
  'signup.notConnected':
    'O Cognito ainda não foi implantado, então ainda não dá pra criar contas de verdade. Jogue como convidado pra testar o app agora.',
  'signup.hasAccount': 'Já tem conta?',
  'signup.loginLink': 'Entrar',

  'game.loading': 'Carregando partida…',
  'game.white': 'Brancas',
  'game.black': 'Pretas',
  'game.resign': 'Desistir',
  'game.offerDraw': 'Oferecer empate',
  'game.acceptDraw': 'Aceitar empate',
  'game.over': 'Partida encerrada',
  'game.winner': 'Vencedor',
  'game.backToLobby': 'Voltar pro lobby',
  'game.chatPlaceholder': 'Mensagem',
  'game.send': 'Enviar',
  'game.noMessages': 'Sem mensagens ainda.',

  'leaderboard.title': 'Leaderboard',
  'leaderboard.empty': 'Sem dados ainda.',

  'history.title': 'Histórico',
  'history.empty': 'Nenhuma partida ainda.',
  'history.viewPgn': 'Ver PGN',
  'history.vsAI': 'Motor',
  'history.result.win': 'vitória',
  'history.result.loss': 'derrota',
  'history.result.draw': 'empate',

  'profile.loading': 'Carregando perfil…',
  'profile.private': 'Este perfil é privado.',
  'profile.rating': 'Rating',
  'profile.wins': 'Vitórias',
  'profile.losses': 'Derrotas',
  'profile.draws': 'Empates',
}

const dictionaries: Record<Language, Record<keyof typeof en, string>> = { en, pt }

type TranslationKey = keyof typeof en

interface LanguageValue {
  language: Language
  setLanguage: (lang: Language) => void
  t: (key: TranslationKey) => string
}

const LanguageContext = createContext<LanguageValue | null>(null)

function detectInitialLanguage(): Language {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'en' || stored === 'pt') return stored
  return navigator.language.toLowerCase().startsWith('pt') ? 'pt' : 'en'
}

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>(detectInitialLanguage)

  function setLanguage(lang: Language) {
    localStorage.setItem(STORAGE_KEY, lang)
    setLanguageState(lang)
  }

  const t = useMemo(() => {
    const dict = dictionaries[language]
    return (key: TranslationKey) => dict[key]
  }, [language])

  const value: LanguageValue = { language, setLanguage, t }

  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>
}

export function useTranslation(): LanguageValue {
  const ctx = useContext(LanguageContext)
  if (!ctx) {
    throw new Error('useTranslation must be used within LanguageProvider')
  }
  return ctx
}
