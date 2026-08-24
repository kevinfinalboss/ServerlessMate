import { getActiveGameId, getToken } from './session'
import type { ServerMessage } from './types'

const WS_URL = import.meta.env.VITE_WS_URL as string | undefined

export type SocketListener = (msg: ServerMessage) => void
export type SimpleListener = () => void

const INITIAL_RECONNECT_DELAY_MS = 1000
const MAX_RECONNECT_DELAY_MS = 15000

function log(...args: unknown[]) {
  console.log('[ws]', ...args)
}

class GameSocket {
  private ws: WebSocket | null = null
  private messageListeners = new Set<SocketListener>()
  private openListeners = new Set<SimpleListener>()
  private closeListeners = new Set<SimpleListener>()
  private intentionalClose = false
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectDelayMs = INITIAL_RECONNECT_DELAY_MS

  connect() {
    if (!WS_URL) {
      console.error('VITE_WS_URL is not configured')
      return
    }
    if (this.ws && this.ws.readyState <= WebSocket.OPEN) {
      log('connect() skipped, already', this.ws.readyState === WebSocket.OPEN ? 'open' : 'connecting')
      return
    }

    this.intentionalClose = false
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }

    const params = new URLSearchParams()
    const token = getToken()
    if (token) params.set('token', token)
    const gameId = getActiveGameId()
    if (gameId) params.set('gameId', gameId)

    log('connecting', { hasToken: Boolean(token), gameId: gameId ?? null })

    const query = params.toString()
    const ws = new WebSocket(query ? `${WS_URL}?${query}` : WS_URL)

    ws.onopen = () => {
      log('open')
      this.reconnectDelayMs = INITIAL_RECONNECT_DELAY_MS
      this.openListeners.forEach((fn) => fn())
    }
    ws.onclose = (event) => {
      log('closed', { code: event.code, reason: event.reason || null, wasClean: event.wasClean })
      this.closeListeners.forEach((fn) => fn())
      if (!this.intentionalClose) {
        this.scheduleReconnect()
      }
    }
    ws.onerror = () => {
      log('error event (see close event for details)')
    }
    ws.onmessage = (event) => {
      let data: ServerMessage
      try {
        data = JSON.parse(event.data as string) as ServerMessage
      } catch {
        log('received unparseable message', event.data)
        return
      }
      log('message received', data)
      this.messageListeners.forEach((fn) => fn(data))
    }

    this.ws = ws
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return
    log(`reconnecting in ${this.reconnectDelayMs}ms`)
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, this.reconnectDelayMs)
    this.reconnectDelayMs = Math.min(this.reconnectDelayMs * 2, MAX_RECONNECT_DELAY_MS)
  }

  disconnect() {
    log('disconnect() called intentionally')
    this.intentionalClose = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.ws?.close()
    this.ws = null
  }

  send(payload: Record<string, unknown>) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      log('send', payload)
      this.ws.send(JSON.stringify(payload))
      return
    }
    console.warn('[ws] send() dropped, socket not open', { readyState: this.ws?.readyState ?? null, payload })
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
