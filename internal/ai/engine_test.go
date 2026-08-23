package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func TestEvaluate_StartingPosition(t *testing.T) {
	assert.Equal(t, 0.0, evaluate(startFEN))
}

func TestEvaluate_MaterialImbalance(t *testing.T) {
	fen := "4k3/8/8/8/8/8/8/4KQ2 w - - 0 1"
	assert.Equal(t, 9.0, evaluate(fen))
}

func TestEvaluate_BlackAdvantage(t *testing.T) {
	fen := "4kq2/8/8/8/8/8/8/4K3 w - - 0 1"
	assert.Equal(t, -9.0, evaluate(fen))
}

func TestNewEngine(t *testing.T) {
	_, isHeuristic := NewEngine(LevelEasy).(HeuristicEngine)
	assert.True(t, isHeuristic)

	_, isMinimax := NewEngine(LevelHard).(*MinimaxEngine)
	assert.True(t, isMinimax)
}

func TestHeuristicEngine_CapturesFreePiece(t *testing.T) {
	fen := "4k3/8/8/8/8/8/r7/Q3K3 w - - 0 1"

	move, err := HeuristicEngine{}.BestMove(fen)

	require.NoError(t, err)
	assert.Equal(t, "a1a2", move)
}

func TestHeuristicEngine_NoLegalMoves(t *testing.T) {
	fen := "rn1qkbnr/pbpp1Qpp/1p6/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 1"

	_, err := HeuristicEngine{}.BestMove(fen)

	assert.ErrorIs(t, err, ErrNoMoves)
}

func TestHeuristicEngine_InvalidFEN(t *testing.T) {
	_, err := HeuristicEngine{}.BestMove("not a fen")

	assert.Error(t, err)
}

func TestHeuristicEngine_FallsForHangingPiece(t *testing.T) {
	fen := "4k3/8/8/2b5/3p4/8/8/3QK3 w - - 0 1"

	move, err := HeuristicEngine{}.BestMove(fen)

	require.NoError(t, err)
	assert.Equal(t, "d1d4", move, "1-ply heuristic should greedily grab the defended pawn")
}

func TestMinimaxEngine_AvoidsHangingQueen(t *testing.T) {
	fen := "4k3/8/8/2b5/3p4/8/8/3QK3 w - - 0 1"

	move, err := (&MinimaxEngine{Depth: 3}).BestMove(fen)

	require.NoError(t, err)
	assert.NotEqual(t, "d1d4", move, "minimax should see the bishop recapture and avoid losing the queen for a pawn")
}

func TestMinimaxEngine_CapturesFreePiece(t *testing.T) {
	fen := "4k3/8/8/8/8/8/r7/Q3K3 w - - 0 1"

	move, err := (&MinimaxEngine{Depth: 3}).BestMove(fen)

	require.NoError(t, err)
	assert.Equal(t, "a1a2", move)
}

func TestMinimaxEngine_DefaultDepth(t *testing.T) {
	fen := "4k3/8/8/8/8/8/r7/Q3K3 w - - 0 1"

	move, err := (&MinimaxEngine{}).BestMove(fen)

	require.NoError(t, err)
	assert.Equal(t, "a1a2", move)
}

func TestMinimaxEngine_NoLegalMoves(t *testing.T) {
	fen := "rn1qkbnr/pbpp1Qpp/1p6/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 1"

	_, err := (&MinimaxEngine{Depth: 3}).BestMove(fen)

	assert.ErrorIs(t, err, ErrNoMoves)
}

func TestMinimaxEngine_InvalidFEN(t *testing.T) {
	_, err := (&MinimaxEngine{Depth: 3}).BestMove("not a fen")

	assert.Error(t, err)
}

func TestMinimaxEngine_BlackToMove(t *testing.T) {
	fen := "4k3/7r/8/8/8/8/8/q3K3 b - - 0 1"

	move, err := (&MinimaxEngine{Depth: 2}).BestMove(fen)

	require.NoError(t, err)
	assert.NotEmpty(t, move)
}
