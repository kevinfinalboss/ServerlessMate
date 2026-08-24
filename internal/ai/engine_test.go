package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinfinalboss/ServerlessMate/internal/game"
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

const mateInOneFEN = "r3k3/1q6/8/3B4/8/8/6PP/7K b - - 0 1"

func assertDeliversMate(t *testing.T, move string) {
	t.Helper()
	g, err := game.FromFEN(mateInOneFEN)
	require.NoError(t, err)
	require.NoError(t, g.Move(move))
	assert.Equal(t, game.Checkmate, g.Status(), "chosen move %q should deliver checkmate, not just grab material", move)
}

func TestHeuristicEngine_PrefersMateOverMaterial(t *testing.T) {
	move, err := HeuristicEngine{}.BestMove(mateInOneFEN)

	require.NoError(t, err)
	assertDeliversMate(t, move)
}

func TestMinimaxEngine_PrefersMateOverMaterial(t *testing.T) {
	move, err := (&MinimaxEngine{Depth: 3}).BestMove(mateInOneFEN)

	require.NoError(t, err)
	assertDeliversMate(t, move)
}

func TestEvaluatePosition_Checkmate(t *testing.T) {
	g, err := game.FromFEN(mateInOneFEN)
	require.NoError(t, err)
	require.NoError(t, g.Move("a8a1"))

	assert.Equal(t, float64(-mateScore), evaluatePosition(g))
}

func TestEvaluatePosition_Stalemate(t *testing.T) {
	fen := "7k/8/6Q1/8/8/8/8/7K b - - 0 1"
	g, err := game.FromFEN(fen)
	require.NoError(t, err)

	assert.Equal(t, 0.0, evaluatePosition(g))
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
