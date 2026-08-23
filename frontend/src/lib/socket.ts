import { getActiveGameId, getToken } from './session'
import type { ServerMessage } from './types'

const WS_URL = import.meta.env.VITE_WS_URL as string | undefined

export type SocketListener = (msg: ServerMessage) => void
export type SimpleListener = () => void

class GameSocket {
  private ws: WebSocket | null = null
  private messageListeners = new Set<SocketListener>()
  private openListeners = new Set<SimpleListener>()
  private closeListeners = new Set<SimpleListener>()

  connect() {
    if (!WS_URL) {
      console.error('VITE_WS_URL is not configured')
      return
    }
    if (this.ws && this.ws.readyState <= WebSocket.OPEN) {
      return
    }

    const params = new URLSearchParams()
    const token = getToken()
    if (token) params.set('token', token)
    const gameId = getActiveGameId()
    if (gameId) params.set('gameId', gameId)

    const query = params.toString()
    const ws = new WebSocket(query ? `${WS_URL}?${query}` : WS_URL)

    ws.onopen = () => {
      this.openListeners.forEach((fn) => fn())
    }
    ws.onclose = () => {
      this.closeListeners.forEach((fn) => fn())
    }
    ws.onmessage = (event) => {
      let data: ServerMessage
      try {
        data = JSON.parse(event.data as string) as ServerMessage
      } catch {
        return
      }
      this.messageListeners.forEach((fn) => fn(data))
    }

    this.ws = ws
  }

  disconnect() {
    this.ws?.close()
    this.ws = null
  }

  send(payload: Record<string, unknown>) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(payload))
    }
  }

  get isOpen() {
    return this.ws?.readyState === WebSocket.OPEN
  }

  onMessage(fn: SocketListener) {
    this.messageListeners.add(fn)
    return () => this.messageListeners.delete(fn)
  }

  onOpen(fn: SimpleListener) {
    this.openListeners.add(fn)
    return () => this.openListeners.delete(fn)
  }

  onClose(fn: SimpleListener) {
    this.closeListeners.add(fn)
    return () => this.closeListeners.delete(fn)
  }
}

export const socket = new GameSocket()
