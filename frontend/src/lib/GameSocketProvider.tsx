import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { socket } from './socket'
import { clearActiveGameId, setPlayerId } from './session'
import { isGameState, isType } from './types'
import type { ChatMessage, ConnectedMessage, ErrorMessage, GameState, ServerMessage } from './types'

interface GameSocketValue {
  connected: boolean
  identity: ConnectedMessage | null
  game: GameState | null
  chat: ChatMessage[]
  error: string | null
  dismissError: () => void
  lastMessage: ServerMessage | null
  send: (payload: Record<string, unknown>) => void
  connect: () => void
  leaveGame: () => void
}

const GameSocketContext = createContext<GameSocketValue | null>(null)

export function GameSocketProvider({ children }: { children: ReactNode }) {
  const [connected, setConnected] = useState(socket.isOpen)
  const [identity, setIdentity] = useState<ConnectedMessage | null>(null)
  const [game, setGame] = useState<GameState | null>(null)
  const [chat, setChat] = useState<ChatMessage[]>([])
  const [error, setError] = useState<string | null>(null)
  const [lastMessage, setLastMessage] = useState<ServerMessage | null>(null)

  useEffect(() => {
    const offOpen = socket.onOpen(() => setConnected(true))
    const offClose = socket.onClose(() => setConnected(false))
    const offMessage = socket.onMessage((msg) => {
      setLastMessage(msg)
      if (isType<ConnectedMessage>(msg, 'connected')) {
        setIdentity(msg)
        setPlayerId(msg.playerId)
        if (!msg.gameId) {
          clearActiveGameId()
        }
        return
      }
      if (isGameState(msg)) {
        console.log('[game] state updated', {
          gameId: msg.gameId,
          status: msg.status,
          turnOf: msg.turnOf,
          myTurn: msg.turnOf === msg.players.white || msg.turnOf === msg.players.black,
        })
        setGame(msg)
        return
      }
      if (isType<ChatMessage>(msg, 'chat')) {
        setChat((prev) => [...prev, msg])
        return
      }
      if (isType<ErrorMessage>(msg, 'error')) {
        console.warn('[game] server error', msg.message)
        setError(msg.message)
        if (msg.message === 'game no longer exists') {
          console.log('[game] clearing stale game state and gameId')
          clearActiveGameId()
          setGame(null)
        }
      }
    })

    return () => {
      offOpen()
      offClose()
      offMessage()
    }
  }, [])

  const send = useCallback((payload: Record<string, unknown>) => {
    socket.send(payload)
  }, [])

  const dismissError = useCallback(() => setError(null), [])
  const connect = useCallback(() => socket.connect(), [])
  const leaveGame = useCallback(() => {
    setGame(null)
    setChat([])
  }, [])

  const value: GameSocketValue = {
    connected,
    identity,
    game,
    chat,
    error,
    dismissError,
    lastMessage,
    send,
    connect,
    leaveGame,
  }

  return <GameSocketContext.Provider value={value}>{children}</GameSocketContext.Provider>
}

export function useGameSocket(): GameSocketValue {
  const ctx = useContext(GameSocketContext)
  if (!ctx) {
    throw new Error('useGameSocket must be used within GameSocketProvider')
  }
  return ctx
}
