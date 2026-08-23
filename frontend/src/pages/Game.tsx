import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Chessboard } from 'react-chessboard'
import type { PieceDropHandlerArgs } from 'react-chessboard'
import { Chess } from 'chess.js'
import { useGameSocket } from '../lib/GameSocketProvider'
import { clearActiveGameId, getActiveGameId, getPlayerId } from '../lib/session'

const AI_PLAYER_ID = 'AI'

function formatClock(ms: number) {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

export function Game() {
  const navigate = useNavigate()
  const { identity, game, chat, error, dismissError, send } = useGameSocket()
  const [chatInput, setChatInput] = useState('')
  const aiTriggeredForFEN = useRef<string | null>(null)

  const myPlayerId = identity?.playerId ?? getPlayerId()

  useEffect(() => {
    if (!getActiveGameId()) {
      navigate('/')
    }
  }, [navigate])

  useEffect(() => {
    if (!game || !game.vsAI) return
    if (game.status !== 'in_progress') return
    if (game.turnOf !== AI_PLAYER_ID) return
    if (aiTriggeredForFEN.current === game.fen) return
    aiTriggeredForFEN.current = game.fen
    send({ action: 'move' })
  }, [game, send])

  const myColor: 'white' | 'black' | null =
    game && myPlayerId
      ? game.players.white === myPlayerId
        ? 'white'
        : game.players.black === myPlayerId
          ? 'black'
          : null
      : null

  const isMyTurn = Boolean(game && myPlayerId && game.turnOf === myPlayerId)

  const onPieceDrop = useCallback(
    ({ piece, sourceSquare, targetSquare }: PieceDropHandlerArgs) => {
      if (!game || !targetSquare || !isMyTurn) return false

      const chess = new Chess(game.fen)
      const isPromotion =
        piece.pieceType.toLowerCase().endsWith('p') &&
        (targetSquare.endsWith('8') || targetSquare.endsWith('1'))

      try {
        chess.move({ from: sourceSquare, to: targetSquare, promotion: isPromotion ? 'q' : undefined })
      } catch {
        return false
      }

      send({ action: 'move', move: `${sourceSquare}${targetSquare}${isPromotion ? 'q' : ''}` })
      return true
    },
    [game, isMyTurn, send],
  )

  function resign() {
    send({ action: 'resign' })
  }

  function offerDraw() {
    send({ action: 'offerDraw' })
  }

  function acceptDraw() {
    send({ action: 'acceptDraw' })
  }

  function sendChat() {
    if (!chatInput.trim()) return
    send({ action: 'chat', message: chatInput.trim() })
    setChatInput('')
  }

  function leaveGame() {
    clearActiveGameId()
    navigate('/')
  }

  if (!game) {
    return <div className="p-6 text-center text-gray-400">Carregando partida...</div>
  }

  const isOver = game.status !== 'in_progress'
  const opponentOfferedDraw = Boolean(game.drawOfferedBy) && game.drawOfferedBy !== myPlayerId

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-4 p-6 md:flex-row">
      <div className="flex-1">
        <Chessboard
          options={{
            id: 'main-board',
            position: game.fen,
            boardOrientation: myColor ?? 'white',
            onPieceDrop,
            allowDragging: isMyTurn && !isOver,
          }}
        />
        <div className="mt-4 flex items-center justify-between text-white">
          <span>Brancas: {formatClock(game.whiteTimeMs)}</span>
          <span>Pretas: {formatClock(game.blackTimeMs)}</span>
        </div>
        {game.comment && <p className="mt-2 text-sm italic text-emerald-400">"{game.comment}"</p>}
        {isOver ? (
          <div className="mt-4 rounded-md bg-gray-800 p-3 text-center text-white">
            <p className="font-semibold">Partida encerrada: {game.status}</p>
            {game.winner && <p className="text-sm text-gray-400">Vencedor: {game.winner}</p>}
            <button onClick={leaveGame} className="mt-2 rounded-md bg-emerald-600 px-4 py-1.5 text-sm">
              Voltar pro lobby
            </button>
          </div>
        ) : (
          <div className="mt-4 flex gap-2">
            <button onClick={resign} className="rounded-md bg-red-700 px-3 py-1.5 text-sm text-white hover:bg-red-600">
              Desistir
            </button>
            <button
              onClick={offerDraw}
              className="rounded-md bg-gray-700 px-3 py-1.5 text-sm text-white hover:bg-gray-600"
            >
              Oferecer empate
            </button>
            {opponentOfferedDraw && (
              <button
                onClick={acceptDraw}
                className="rounded-md bg-emerald-700 px-3 py-1.5 text-sm text-white hover:bg-emerald-600"
              >
                Aceitar empate
              </button>
            )}
          </div>
        )}
      </div>

      <div className="flex w-full flex-col md:w-72">
        <div className="flex-1 overflow-y-auto rounded-md border border-gray-800 bg-gray-900 p-3">
          {chat.length === 0 && <p className="text-sm text-gray-500">Sem mensagens ainda.</p>}
          {chat.map((msg, i) => (
            <p key={i} className="mb-1 text-sm text-gray-300">
              <span className="font-semibold text-gray-500">{msg.playerId}: </span>
              {msg.message}
            </p>
          ))}
        </div>
        <div className="mt-2 flex gap-2">
          <input
            className="flex-1 rounded-md border border-gray-700 bg-gray-800 px-2 py-1.5 text-sm text-white"
            value={chatInput}
            onChange={(e) => setChatInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && sendChat()}
            placeholder="Mensagem"
          />
          <button onClick={sendChat} className="rounded-md bg-gray-700 px-3 py-1.5 text-sm text-white hover:bg-gray-600">
            Enviar
          </button>
        </div>
      </div>

      {error && (
        <button
          onClick={dismissError}
          className="fixed bottom-4 left-1/2 -translate-x-1/2 rounded-md bg-red-800 px-4 py-2 text-sm text-white shadow-lg"
        >
          {error}
        </button>
      )}
    </div>
  )
}
