import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Chessboard } from 'react-chessboard'
import type { PieceDropHandlerArgs } from 'react-chessboard'
import { Chess } from 'chess.js'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
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
  const { t } = useTranslation()
  const [chatInput, setChatInput] = useState('')
  const aiTriggeredForFEN = useRef<string | null>(null)

  const myPlayerId = identity?.playerId ?? getPlayerId()

  useEffect(() => {
    if (!getActiveGameId()) {
      navigate('/play')
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
    navigate('/play')
  }

  if (!game) {
    return <div className="p-6 text-center text-sm text-mute">{t('game.loading')}</div>
  }

  const isOver = game.status !== 'in_progress'
  const opponentOfferedDraw = Boolean(game.drawOfferedBy) && game.drawOfferedBy !== myPlayerId

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-4 p-6 md:flex-row">
      <div className="flex-1">
        <div className="overflow-hidden rounded-2xl border border-ink-line bg-ink-raised p-3">
          <Chessboard
            options={{
              id: 'main-board',
              position: game.fen,
              boardOrientation: myColor ?? 'white',
              onPieceDrop,
              allowDragging: isMyTurn && !isOver,
              darkSquareStyle: { backgroundColor: 'var(--color-felt)' },
              lightSquareStyle: { backgroundColor: 'var(--color-paper)' },
            }}
          />
        </div>
        <div className="mt-4 flex items-center justify-between font-mono font-tabular text-paper">
          <span>
            {t('game.white')}: {formatClock(game.whiteTimeMs)}
          </span>
          <span>
            {t('game.black')}: {formatClock(game.blackTimeMs)}
          </span>
        </div>
        {game.comment && <p className="mt-2 text-sm text-lime italic">&ldquo;{game.comment}&rdquo;</p>}
        {isOver ? (
          <div className="mt-4 rounded-2xl border border-ink-line bg-ink-raised p-4 text-center">
            <p className="font-display font-medium text-paper">
              {t('game.over')}: {game.status}
            </p>
            {game.winner && (
              <p className="mt-1 text-sm text-mute">
                {t('game.winner')}: {game.winner}
              </p>
            )}
            <button
              onClick={leaveGame}
              className="mt-3 rounded-full bg-lime px-4 py-1.5 text-sm font-semibold text-ink"
            >
              {t('game.backToLobby')}
            </button>
          </div>
        ) : (
          <div className="mt-4 flex gap-2">
            <button
              onClick={resign}
              className="rounded-full bg-ember/90 px-3 py-1.5 text-sm font-medium text-ink hover:bg-ember"
            >
              {t('game.resign')}
            </button>
            <button
              onClick={offerDraw}
              className="rounded-full bg-ink-line px-3 py-1.5 text-sm font-medium text-paper hover:bg-felt"
            >
              {t('game.offerDraw')}
            </button>
            {opponentOfferedDraw && (
              <button
                onClick={acceptDraw}
                className="rounded-full bg-lime px-3 py-1.5 text-sm font-semibold text-ink"
              >
                {t('game.acceptDraw')}
              </button>
            )}
          </div>
        )}
      </div>

      <div className="flex w-full flex-col md:w-72">
        <div className="flex-1 overflow-y-auto rounded-2xl border border-ink-line bg-ink-raised p-3">
          {chat.length === 0 && <p className="text-sm text-mute">{t('game.noMessages')}</p>}
          {chat.map((msg, i) => (
            <p key={i} className="mb-1 text-sm text-paper">
              <span className="font-medium text-mute">{msg.playerId}: </span>
              {msg.message}
            </p>
          ))}
        </div>
        <div className="mt-2 flex gap-2">
          <input
            className="flex-1 rounded-lg border border-ink-line bg-ink px-2 py-1.5 text-sm text-paper outline-none focus:border-lime"
            value={chatInput}
            onChange={(e) => setChatInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && sendChat()}
            placeholder={t('game.chatPlaceholder')}
          />
          <button
            onClick={sendChat}
            className="rounded-lg bg-ink-line px-3 py-1.5 text-sm font-medium text-paper hover:bg-felt"
          >
            {t('game.send')}
          </button>
        </div>
      </div>

      {error && (
        <button
          onClick={dismissError}
          className="fixed bottom-4 left-1/2 -translate-x-1/2 rounded-full bg-ember px-4 py-2 text-sm font-medium text-ink shadow-lg"
        >
          {error}
        </button>
      )}
    </div>
  )
}
