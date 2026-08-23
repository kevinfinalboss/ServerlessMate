import { useEffect, useMemo, useRef, useState } from 'react'
import { Chessboard } from 'react-chessboard'
import { Chess } from 'chess.js'

const MATE_SEQUENCE = ['e4', 'e5', 'Bc4', 'Nc6', 'Qh5', 'Nf6', 'Qxf7#']
const STEP_MS = 950
const PAUSE_MS = 2600
const START_FEN = new Chess().fen()

function fenAfter(steps: number): string {
  const chess = new Chess()
  for (let i = 0; i < steps; i++) {
    chess.move(MATE_SEQUENCE[i])
  }
  return chess.fen()
}

export function AmbientBoard() {
  const prefersReducedMotion = useMemo(
    () =>
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    [],
  )
  const finalFen = useMemo(() => fenAfter(MATE_SEQUENCE.length), [])
  const [fen, setFen] = useState(prefersReducedMotion ? finalFen : START_FEN)
  const stepRef = useRef(0)

  useEffect(() => {
    if (prefersReducedMotion) return

    let cancelled = false
    let timeout: ReturnType<typeof setTimeout>

    function tick() {
      if (cancelled) return
      const nextStep = stepRef.current + 1

      if (nextStep > MATE_SEQUENCE.length) {
        stepRef.current = 0
        setFen(START_FEN)
        timeout = setTimeout(tick, STEP_MS)
        return
      }

      stepRef.current = nextStep
      setFen(fenAfter(nextStep))
      timeout = setTimeout(tick, nextStep === MATE_SEQUENCE.length ? PAUSE_MS : STEP_MS)
    }

    timeout = setTimeout(tick, STEP_MS)
    return () => {
      cancelled = true
      clearTimeout(timeout)
    }
  }, [prefersReducedMotion])

  return (
    <div className="w-full max-w-md rounded-2xl border border-ink-line bg-ink-raised p-3 shadow-2xl shadow-black/40">
      <Chessboard
        options={{
          id: 'ambient-board',
          position: fen,
          allowDragging: false,
          animationDurationInMs: prefersReducedMotion ? 0 : 400,
          showNotation: false,
          darkSquareStyle: { backgroundColor: 'var(--color-felt)' },
          lightSquareStyle: { backgroundColor: 'var(--color-paper)' },
        }}
      />
    </div>
  )
}
