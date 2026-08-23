const TOKEN_KEY = 'sm.token'
const GAME_ID_KEY = 'sm.gameId'
const PLAYER_ID_KEY = 'sm.playerId'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export function isGuest(): boolean {
  return !getToken()
}

export function getActiveGameId(): string | null {
  return localStorage.getItem(GAME_ID_KEY)
}

export function setActiveGameId(gameId: string) {
  localStorage.setItem(GAME_ID_KEY, gameId)
}

export function clearActiveGameId() {
  localStorage.removeItem(GAME_ID_KEY)
}

export function getPlayerId(): string | null {
  return localStorage.getItem(PLAYER_ID_KEY)
}

export function setPlayerId(playerId: string) {
  localStorage.setItem(PLAYER_ID_KEY, playerId)
}
