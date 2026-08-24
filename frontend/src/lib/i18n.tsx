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
  'nav.friends': 'Friends',
  'nav.leaderboard': 'Leaderboard',
  'nav.history': 'History',
  'nav.profile': 'Profile',

  'dashboard.greeting': 'Ready for a game?',
  'dashboard.ratingWidget.title': 'Your rating',
  'dashboard.guestCta.title': 'Playing as a guest',
  'dashboard.guestCta.body': 'Create an account to keep your rating, add friends, and challenge the engine.',
  'dashboard.friendsWidget.title': 'Friends',
  'dashboard.friendsWidget.empty': 'No friends yet.',
  'dashboard.friendsWidget.pending': 'pending',
  'dashboard.friendsWidget.viewAll': 'View all',
  'dashboard.leaderboardWidget.title': 'Top rated',
  'dashboard.leaderboardWidget.viewAll': 'Full leaderboard',
  'dashboard.historyWidget.title': 'Recent games',
  'dashboard.historyWidget.empty': 'No games yet — play one!',
  'dashboard.historyWidget.viewAll': 'Full history',

  'play.disconnected': 'Connection lost — trying to reconnect…',
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
  'play.cancelSearch': 'Cancel search',

  'login.title': 'Welcome back',
  'login.subtitle': 'Log in to track your rating and play the engine.',
  'login.email': 'Email',
  'login.password': 'Password',
  'login.submit': 'Log in',
  'login.submitting': 'Logging in…',
  'login.needsConfirmation': 'This account hasn’t been confirmed yet.',
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
  'signup.passwordHint': '8+ characters, with uppercase, lowercase, and a number.',
  'signup.submit': 'Sign up',
  'signup.submitting': 'Creating your account…',
  'signup.hasAccount': 'Already have an account?',
  'signup.loginLink': 'Log in',

  'auth.confirmTitle': 'Confirm your email',
  'auth.confirmSubtitle': 'Enter the code we emailed you.',
  'auth.confirmCodeLabel': 'Confirmation code',
  'auth.confirmSubmit': 'Confirm',
  'auth.resendCode': 'Resend code',
  'auth.resendSent': 'Code sent — check your email.',
  'auth.errors.usernameExists': 'An account with this email already exists.',
  'auth.errors.notAuthorized': 'Incorrect email or password.',
  'auth.errors.userNotFound': 'No account found with this email.',
  'auth.errors.userNotConfirmed': 'Please confirm your email before logging in.',
  'auth.errors.codeMismatch': 'That code isn’t correct.',
  'auth.errors.codeExpired': 'That code has expired — request a new one.',
  'auth.errors.invalidPassword': 'Password must be 8+ characters with uppercase, lowercase, and a number.',
  'auth.errors.tooManyRequests': 'Too many attempts — try again shortly.',
  'auth.errors.generic': 'Something went wrong. Try again.',

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
  'game.you': 'You',
  'game.vs': 'vs',
  'game.ai': 'The engine',
  'game.result.checkmate': 'Checkmate.',
  'game.result.stalemate': 'Draw by stalemate.',
  'game.result.draw_agreement': 'Draw by agreement.',
  'game.result.draw_repetition': 'Draw by repetition.',
  'game.result.draw_move_rule': 'Draw by the 50-move rule.',
  'game.result.draw_insufficient_material': 'Draw by insufficient material.',
  'game.result.timeoutSuffix': 'ran out of time.',
  'game.result.resignedSuffix': 'resigned.',
  'game.result.abandonedSuffix': 'abandoned the game.',

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
  'profile.addFriend': 'Add friend',
  'profile.friendRequestSent': 'Request sent',
  'profile.acceptRequest': 'Accept request',
  'profile.friends': 'Friends',
  'profile.country': 'Country',
  'profile.countryPlaceholder': 'e.g. Brazil',
  'profile.birthDate': 'Birth date',
  'profile.settings.title': 'Profile settings',
  'profile.settings.visibility': 'Who can see your stats',
  'profile.settings.visibilityPublic': 'Anyone',
  'profile.settings.visibilityFriends': 'Friends only',
  'profile.settings.optionalHint': 'Everything below is optional.',
  'profile.settings.save': 'Save',
  'profile.settings.saved': 'Saved.',

  'friends.title': 'Friends',
  'friends.loginRequired': 'Log in to add friends and see who’s playing.',
  'friends.empty': 'No friends yet — add one below.',
  'friends.incoming.title': 'Requests for you',
  'friends.incoming.empty': 'No pending requests.',
  'friends.outgoing.title': 'Sent requests',
  'friends.outgoing.empty': 'Nothing waiting on the other end.',
  'friends.accept': 'Accept',
  'friends.decline': 'Decline',
  'friends.cancel': 'Cancel',
  'friends.addTitle': 'Add a friend',
  'friends.addPlaceholder': 'Player ID',
  'friends.addCta': 'Send request',
  'friends.addHint': 'Find a player ID on the leaderboard or in a match history entry.',
  'friends.addSent': 'Request sent.',
  'friends.addError': 'Couldn’t send that request.',
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
  'nav.friends': 'Amigos',
  'nav.leaderboard': 'Ranking',
  'nav.history': 'Histórico',
  'nav.profile': 'Perfil',

  'dashboard.greeting': 'Pronto pra jogar?',
  'dashboard.ratingWidget.title': 'Seu rating',
  'dashboard.guestCta.title': 'Jogando como convidado',
  'dashboard.guestCta.body': 'Crie uma conta pra manter seu rating, adicionar amigos e desafiar a IA.',
  'dashboard.friendsWidget.title': 'Amigos',
  'dashboard.friendsWidget.empty': 'Nenhum amigo ainda.',
  'dashboard.friendsWidget.pending': 'pendente(s)',
  'dashboard.friendsWidget.viewAll': 'Ver todos',
  'dashboard.leaderboardWidget.title': 'Melhores ratings',
  'dashboard.leaderboardWidget.viewAll': 'Ranking completo',
  'dashboard.historyWidget.title': 'Partidas recentes',
  'dashboard.historyWidget.empty': 'Nenhuma partida ainda — que tal jogar uma?',
  'dashboard.historyWidget.viewAll': 'Histórico completo',

  'play.disconnected': 'Conexão perdida — tentando reconectar…',
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
  'play.cancelSearch': 'Cancelar busca',

  'login.title': 'Bem-vindo de volta',
  'login.subtitle': 'Entre pra acompanhar seu rating e jogar contra o motor.',
  'login.email': 'Email',
  'login.password': 'Senha',
  'login.submit': 'Entrar',
  'login.submitting': 'Entrando…',
  'login.needsConfirmation': 'Essa conta ainda não foi confirmada.',
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
  'signup.passwordHint': '8+ caracteres, com maiúscula, minúscula e número.',
  'signup.submit': 'Cadastrar',
  'signup.submitting': 'Criando sua conta…',
  'signup.hasAccount': 'Já tem conta?',
  'signup.loginLink': 'Entrar',

  'auth.confirmTitle': 'Confirme seu email',
  'auth.confirmSubtitle': 'Digite o código que enviamos pro seu email.',
  'auth.confirmCodeLabel': 'Código de confirmação',
  'auth.confirmSubmit': 'Confirmar',
  'auth.resendCode': 'Reenviar código',
  'auth.resendSent': 'Código enviado — confira seu email.',
  'auth.errors.usernameExists': 'Já existe uma conta com esse email.',
  'auth.errors.notAuthorized': 'Email ou senha incorretos.',
  'auth.errors.userNotFound': 'Nenhuma conta encontrada com esse email.',
  'auth.errors.userNotConfirmed': 'Confirme seu email antes de entrar.',
  'auth.errors.codeMismatch': 'Esse código não está correto.',
  'auth.errors.codeExpired': 'Esse código expirou — peça um novo.',
  'auth.errors.invalidPassword': 'A senha precisa ter 8+ caracteres, com maiúscula, minúscula e número.',
  'auth.errors.tooManyRequests': 'Muitas tentativas — tente de novo daqui a pouco.',
  'auth.errors.generic': 'Algo deu errado. Tente de novo.',

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
  'game.you': 'Você',
  'game.vs': 'vs',
  'game.ai': 'A IA',
  'game.result.checkmate': 'Xeque-mate.',
  'game.result.stalemate': 'Empate por afogamento.',
  'game.result.draw_agreement': 'Empate por acordo.',
  'game.result.draw_repetition': 'Empate por repetição.',
  'game.result.draw_move_rule': 'Empate pela regra dos 50 lances.',
  'game.result.draw_insufficient_material': 'Empate por material insuficiente.',
  'game.result.timeoutSuffix': 'ficaram sem tempo.',
  'game.result.resignedSuffix': 'desistiram.',
  'game.result.abandonedSuffix': 'abandonaram a partida.',

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
  'profile.addFriend': 'Adicionar amigo',
  'profile.friendRequestSent': 'Pedido enviado',
  'profile.acceptRequest': 'Aceitar pedido',
  'profile.friends': 'Amigos',
  'profile.country': 'País',
  'profile.countryPlaceholder': 'ex: Brasil',
  'profile.birthDate': 'Data de nascimento',
  'profile.settings.title': 'Configurações do perfil',
  'profile.settings.visibility': 'Quem pode ver suas estatísticas',
  'profile.settings.visibilityPublic': 'Qualquer pessoa',
  'profile.settings.visibilityFriends': 'Só amigos',
  'profile.settings.optionalHint': 'Tudo abaixo é opcional.',
  'profile.settings.save': 'Salvar',
  'profile.settings.saved': 'Salvo.',

  'friends.title': 'Amigos',
  'friends.loginRequired': 'Entra na sua conta pra adicionar amigos e ver quem tá jogando.',
  'friends.empty': 'Nenhum amigo ainda — adiciona um aí embaixo.',
  'friends.incoming.title': 'Pedidos pra você',
  'friends.incoming.empty': 'Nenhum pedido pendente.',
  'friends.outgoing.title': 'Pedidos enviados',
  'friends.outgoing.empty': 'Nada esperando resposta do outro lado.',
  'friends.accept': 'Aceitar',
  'friends.decline': 'Recusar',
  'friends.cancel': 'Cancelar',
  'friends.addTitle': 'Adicionar amigo',
  'friends.addPlaceholder': 'ID do jogador',
  'friends.addCta': 'Enviar pedido',
  'friends.addHint': 'Encontra o ID de um jogador no ranking ou num item do histórico de partidas.',
  'friends.addSent': 'Pedido enviado.',
  'friends.addError': 'Não deu pra enviar esse pedido.',
}

const dictionaries: Record<Language, Record<keyof typeof en, string>> = { en, pt }

export type TranslationKey = keyof typeof en

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
