package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	g := New()

	assert.Equal(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", g.FEN())
	assert.Equal(t, White, g.Turn())
	assert.False(t, g.IsOver())
	assert.Equal(t, InProgress, g.Status())
	assert.Len(t, g.ValidMoves(), 20)
}

func TestFromFEN_Invalid(t *testing.T) {
	_, err := FromFEN("not a fen")
	assert.Error(t, err)
}

func TestFromFEN_Valid(t *testing.T) {
	fen := "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq - 0 1"
	g, err := FromFEN(fen)

	require.NoError(t, err)
	assert.Equal(t, fen, g.FEN())
	assert.Equal(t, Black, g.Turn())
}

func TestMove_Valid(t *testing.T) {
	g := New()

	err := g.Move("e2e4")

	require.NoError(t, err)
	assert.Equal(t, Black, g.Turn())
	assert.Contains(t, g.FEN(), "4P3")
}

func TestMove_Illegal(t *testing.T) {
	cases := []struct {
		name string
		uci  string
	}{
		{"pawn jumps three squares", "e2e5"},
		{"malformed uci", "zz"},
		{"moves opponent piece", "e7e5"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := New()
			startFEN := g.FEN()

			err := g.Move(tc.uci)

			assert.ErrorIs(t, err, ErrIllegalMove)
			assert.Equal(t, startFEN, g.FEN())
		})
	}
}

func TestMove_AfterGameOver(t *testing.T) {
	g, err := FromFEN("rn1qkbnr/pbpp1Qpp/1p6/4p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 1")
	require.NoError(t, err)
	require.True(t, g.IsOver())

	err = g.Move("e5e4")

	assert.ErrorIs(t, err, ErrGameOver)
}

func TestMove_Promotion(t *testing.T) {
	g, err := FromFEN("8/3P4/8/8/8/7k/7p/7K w - - 2 70")
	require.NoError(t, err)

	err = g.Move("d7d8q")

	require.NoError(t, err)
	assert.Contains(t, g.FEN(), "Q")
	assert.False(t, g.IsOver())
}

func TestMove_Castling(t *testing.T) {
	g := New()
	moves := []string{"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "g8f6"}
	for _, m := range moves {
		require.NoError(t, g.Move(m))
	}

	err := g.Move("e1g1")

	require.NoError(t, err)
	assert.Contains(t, g.FEN(), "RK1")
}

func TestMove_EnPassant(t *testing.T) {
	g := New()
	moves := []string{"e2e4", "g8f6", "e4e5", "d7d5"}
	for _, m := range moves {
		require.NoError(t, g.Move(m))
	}

	err := g.Move("e5d6")

	require.NoError(t, err)
	assert.NotContains(t, g.FEN(), "3p4")
}

func TestCheckmate(t *testing.T) {
	g, err := FromFEN("rn1qkbnr/pbpp1ppp/1p6/4p3/2B1P3/5Q2/PPPP1PPP/RNB1K1NR w KQkq - 0 1")
	require.NoError(t, err)

	err = g.Move("f3f7")
	require.NoError(t, err)

	assert.True(t, g.IsOver())
	assert.Equal(t, Checkmate, g.Status())
	winner, ok := g.Winner()
	assert.True(t, ok)
	assert.Equal(t, White, winner)
}

func TestStalemate(t *testing.T) {
	g, err := FromFEN("k1K5/8/8/8/8/8/8/1Q6 w - - 0 1")
	require.NoError(t, err)

	err = g.Move("b1b6")
	require.NoError(t, err)

	assert.True(t, g.IsOver())
	assert.Equal(t, Stalemate, g.Status())
	_, ok := g.Winner()
	assert.False(t, ok)
}

func TestDraw_ByInsufficientMaterial(t *testing.T) {
	g, err := FromFEN("8/2k5/8/8/8/3K4/8/8 w - - 1 1")
	require.NoError(t, err)

	assert.True(t, g.IsOver())
	assert.Equal(t, DrawByInsufficientMaterial, g.Status())
}

func TestResign(t *testing.T) {
	g := New()

	err := g.Resign(White)

	require.NoError(t, err)
	assert.True(t, g.IsOver())
	assert.Equal(t, Resigned, g.Status())
	winner, ok := g.Winner()
	assert.True(t, ok)
	assert.Equal(t, Black, winner)
}

func TestResign_Black(t *testing.T) {
	g := New()

	err := g.Resign(Black)

	require.NoError(t, err)
	winner, ok := g.Winner()
	assert.True(t, ok)
	assert.Equal(t, White, winner)
}

func TestDraw_ByRepetition(t *testing.T) {
	g := New()
	moves := []string{
		"g1f3", "g8f6", "f3g1", "f6g8",
		"g1f3", "g8f6", "f3g1", "f6g8",
		"g1f3", "g8f6", "f3g1", "f6g8",
		"g1f3", "g8f6", "f3g1", "f6g8",
	}
	for _, m := range moves {
		require.NoError(t, g.Move(m))
	}

	assert.True(t, g.IsOver())
	assert.Equal(t, DrawByRepetition, g.Status())
}

func TestDraw_ByMoveRule(t *testing.T) {
	g, err := FromFEN("2r3k1/1q1nbppp/r3p3/3pP3/pPpP4/P1Q2N2/2RN1PPP/2R4K b - b3 149 80")
	require.NoError(t, err)

	err = g.Move("g8f8")

	require.NoError(t, err)
	assert.True(t, g.IsOver())
	assert.Equal(t, DrawByMoveRule, g.Status())
}

func TestResign_AfterGameOver(t *testing.T) {
	g := New()
	require.NoError(t, g.Resign(White))

	err := g.Resign(Black)

	assert.ErrorIs(t, err, ErrGameOver)
}

func TestDraw_ByAgreement(t *testing.T) {
	g := New()

	err := g.Draw()

	require.NoError(t, err)
	assert.True(t, g.IsOver())
	assert.Equal(t, DrawByAgreement, g.Status())
	_, ok := g.Winner()
	assert.False(t, ok)
}

func TestDraw_AfterGameOver(t *testing.T) {
	g := New()
	require.NoError(t, g.Draw())

	err := g.Draw()

	assert.ErrorIs(t, err, ErrGameOver)
}

func TestPGN(t *testing.T) {
	g := New()
	require.NoError(t, g.Move("e2e4"))
	require.NoError(t, g.Move("e7e5"))

	assert.Contains(t, g.PGN(), "1. e4 e5")
}
