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

function decodeJwtSub(token: string): string | null {
  try {
    const payload = token.split('.')[1]
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4)
    const json = atob(padded)
    const claims = JSON.parse(json) as { sub?: string }
    return claims.sub ?? null
  } catch {
    return null
  }
}

export function getPlayerId(): string | null {
  const stored = localStorage.getItem(PLAYER_ID_KEY)
  if (stored) return stored

  const token = getToken()
  return token ? decodeJwtSub(token) : null
}

export function setPlayerId(playerId: string) {
  localStorage.setItem(PLAYER_ID_KEY, playerId)
}
