import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { CSSProperties } from 'react'
import { useNavigate } from 'react-router-dom'
import { Chessboard } from 'react-chessboard'
import type { PieceDropHandlerArgs, SquareHandlerArgs } from 'react-chessboard'
import { Chess } from 'chess.js'
import type { Square } from 'chess.js'
import { useGameSocket } from '../lib/GameSocketProvider'
import { useTranslation } from '../lib/i18n'
import type { TranslationKey } from '../lib/i18n'
import { clearActiveGameId, getActiveGameId, getPlayerId } from '../lib/session'
import { playCaptureSound, playMoveSound } from '../lib/sound'
import { isProfileResponse } from '../lib/types'
import type { GameState } from '../lib/types'

const AI_PLAYER_ID = 'AI'

const DRAW_REASON_KEY: Partial<Record<string, TranslationKey>> = {
  checkmate: 'game.result.checkmate',
  stalemate: 'game.result.stalemate',
  draw_agreement: 'game.result.draw_agreement',
  draw_repetition: 'game.result.draw_repetition',
  draw_move_rule: 'game.result.draw_move_rule',
  draw_insufficient_material: 'game.result.draw_insufficient_material',
}

const LOSS_SUFFIX_KEY: Partial<Record<string, TranslationKey>> = {
  timeout: 'game.result.timeoutSuffix',
  resigned: 'game.result.resignedSuffix',
  abandoned: 'game.result.abandonedSuffix',
}

function describeOutcome(game: GameState, t: (key: TranslationKey) => string): string {
  const reasonKey = DRAW_REASON_KEY[game.status]
  if (reasonKey) return t(reasonKey)

  const suffixKey = LOSS_SUFFIX_KEY[game.status]
  if (!suffixKey || !game.winner) return game.status

  const loserIsWhite = game.winner === game.players.black
  const subject = t(loserIsWhite ? 'game.white' : 'game.black')
  return `${subject} ${t(suffixKey)}`
}

function describeWinner(game: GameState, myPlayerId: string | null, t: (key: TranslationKey) => string): string {
  if (!game.winner) return ''
  if (game.winner === myPlayerId) return t('game.you')
  if (game.winner === AI_PLAYER_ID) return t('game.ai')
  if (game.winner === game.players.white) return t('game.white')
  if (game.winner === game.players.black) return t('game.black')
  return game.winner
}

function formatClock(ms: number) {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

function pieceCount(fen: string): number {
  return fen.split(' ')[0].replace(/[^a-zA-Z]/g, '').length
}

export function Game() {
  const navigate = useNavigate()
  const { identity, game, chat, error, dismissError, send, lastMessage, leaveGame: resetGame } = useGameSocket()
  const { t } = useTranslation()
  const [chatInput, setChatInput] = useState('')
  const [selectedSquare, setSelectedSquare] = useState<Square | null>(null)
  const [opponentUsername, setOpponentUsername] = useState<string | null>(null)
  const aiTriggeredForFEN = useRef<string | null>(null)
  const previousFenRef = useRef<string | null>(null)

  const myPlayerId = identity?.playerId ?? getPlayerId()

  useEffect(() => {
    if (!game || !myPlayerId) return
    const opponentId = game.players.white === myPlayerId ? game.players.black : game.players.white
    if (!opponentId || opponentId === AI_PLAYER_ID) return
    send({ action: 'getProfile', playerId: opponentId })
  }, [game?.gameId, myPlayerId, send])

  useEffect(() => {
    if (lastMessage && isProfileResponse(lastMessage)) {
      setOpponentUsername(lastMessage.username)
    }
  }, [lastMessage])

  useEffect(() => {
    if (!getActiveGameId() && !game) {
      navigate('/play')
    }
  }, [game, navigate])

  useEffect(() => {
    if (!game) return
    const previousFen = previousFenRef.current
    previousFenRef.current = game.fen
    if (previousFen === null || previousFen === game.fen) return

    if (pieceCount(game.fen) < pieceCount(previousFen)) {
      playCaptureSound()
    } else {
      playMoveSound()
    }
  }, [game])

  const [nowTick, setNowTick] = useState(() => Date.now())

  useEffect(() => {
    if (!game || game.status !== 'in_progress') return
    const interval = setInterval(() => setNowTick(Date.now()), 250)
    return () => clearInterval(interval)
  }, [game?.status, game?.gameId])

  const whiteDisplayMs =
    game && game.status === 'in_progress' && game.turnOf === game.players.white
      ? Math.max(0, game.whiteTimeMs - (nowTick - game.lastMoveAt))
      : (game?.whiteTimeMs ?? 0)

  const blackDisplayMs =
    game && game.status === 'in_progress' && game.turnOf === game.players.black
      ? Math.max(0, game.blackTimeMs - (nowTick - game.lastMoveAt))
      : (game?.blackTimeMs ?? 0)

  useEffect(() => {
    if (!game || !game.vsAI) return
    if (game.status !== 'in_progress') return
    if (game.turnOf !== AI_PLAYER_ID) return
    if (aiTriggeredForFEN.current === game.fen) return
    aiTriggeredForFEN.current = game.fen
    send({ action: 'aiMove' })
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

  useEffect(() => {
    setSelectedSquare(null)
  }, [game?.fen])

  useEffect(() => {
    if (!game) return
    console.log('[game] turn check', {
      myPlayerId,
      identityPlayerId: identity?.playerId ?? null,
      storedPlayerId: getPlayerId(),
      players: game.players,
      turnOf: game.turnOf,
      myColor,
      isMyTurn,
      status: game.status,
    })
  }, [game, myPlayerId, identity, myColor, isMyTurn])

  const legalTargets = useMemo(() => {
    if (!game || !selectedSquare) return []
    const chess = new Chess(game.fen)
    return chess.moves({ square: selectedSquare, verbose: true }).map((move): string => move.to)
  }, [game, selectedSquare])

  const submitMove = useCallback(
    (from: string, to: string) => {
      if (!game) return false

      const chess = new Chess(game.fen)
      const piece = chess.get(from as Square)
      const isPromotion = piece?.type === 'p' && (to.endsWith('8') || to.endsWith('1'))

      try {
        chess.move({ from, to, promotion: isPromotion ? 'q' : undefined })
      } catch {
        return false
      }

      send({ action: 'move', move: `${from}${to}${isPromotion ? 'q' : ''}` })
      return true
    },
    [game, send],
  )

  const onPieceDrop = useCallback(
    ({ sourceSquare, targetSquare }: PieceDropHandlerArgs) => {
      if (!targetSquare || !isMyTurn) {
        console.log('[game] drop rejected', { sourceSquare, targetSquare, isMyTurn })
        return false
      }
      const moved = submitMove(sourceSquare, targetSquare)
      console.log('[game] drop', { sourceSquare, targetSquare, moved })
      if (moved) setSelectedSquare(null)
      return moved
    },
    [isMyTurn, submitMove],
  )

  const onSquareClick = useCallback(
    ({ piece, square }: SquareHandlerArgs) => {
      if (!game || !isMyTurn) {
        console.log('[game] square click ignored', { square, hasGame: Boolean(game), isMyTurn })
        return
      }

      if (selectedSquare && legalTargets.includes(square)) {
        const moved = submitMove(selectedSquare, square)
        console.log('[game] click move', { from: selectedSquare, to: square, moved })
        setSelectedSquare(moved ? null : selectedSquare)
        return
      }

      const ownsPiece = piece && piece.pieceType.startsWith(myColor === 'white' ? 'w' : 'b')
      setSelectedSquare(ownsPiece ? (square as Square) : null)
    },
    [game, isMyTurn, myColor, selectedSquare, legalTargets, submitMove],
  )

  const squareStyles = useMemo(() => {
    const styles: Record<string, CSSProperties> = {}
    if (selectedSquare) {
      styles[selectedSquare] = { backgroundColor: 'rgba(159, 232, 91, 0.4)' }
    }
    for (const square of legalTargets) {
      styles[square] = {
        backgroundImage: 'radial-gradient(circle, rgba(159, 232, 91, 0.55) 22%, transparent 26%)',
      }
    }
    return styles
  }, [selectedSquare, legalTargets])

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
    resetGame()
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
        {opponentUsername && <p className="mb-2 text-sm text-mute">{t('game.vs')} {opponentUsername}</p>}
        <div className="overflow-hidden rounded-2xl border border-ink-line bg-ink-raised p-3">
          <Chessboard
            options={{
              id: 'main-board',
              position: game.fen,
              boardOrientation: myColor ?? 'white',
              onPieceDrop,
              onSquareClick,
              squareStyles,
              allowDragging: isMyTurn && !isOver,
              dragActivationDistance: 12,
              darkSquareStyle: { backgroundColor: 'var(--color-felt)' },
              lightSquareStyle: { backgroundColor: 'var(--color-paper)' },
            }}
          />
        </div>
        <div className="mt-4 flex items-center justify-between font-mono font-tabular text-paper">
          <span>
            {t('game.white')}: {formatClock(whiteDisplayMs)}
          </span>
          <span>
            {t('game.black')}: {formatClock(blackDisplayMs)}
          </span>
        </div>
        {game.comment && <p className="mt-2 text-sm text-lime italic">&ldquo;{game.comment}&rdquo;</p>}
        {isOver ? (
          <div className="mt-4 rounded-2xl border border-ink-line bg-ink-raised p-4 text-center">
            <p className="font-display font-medium text-paper">{t('game.over')}</p>
            <p className="mt-1 text-sm text-paper">{describeOutcome(game, t)}</p>
            {game.winner && (
              <p className="mt-1 text-sm text-mute">
                {t('game.winner')}: {describeWinner(game, myPlayerId, t)}
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
