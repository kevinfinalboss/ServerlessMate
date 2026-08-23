package game

import (
	"errors"
	"fmt"

	"github.com/notnil/chess"
)

type Color string

const (
	White Color = "white"
	Black Color = "black"
)

type Status string

const (
	InProgress                 Status = "in_progress"
	Checkmate                  Status = "checkmate"
	Stalemate                  Status = "stalemate"
	DrawByAgreement            Status = "draw_agreement"
	DrawByRepetition           Status = "draw_repetition"
	DrawByMoveRule             Status = "draw_move_rule"
	DrawByInsufficientMaterial Status = "draw_insufficient_material"
	Resigned                   Status = "resigned"
)

var (
	ErrIllegalMove = errors.New("game: illegal move")
	ErrGameOver    = errors.New("game: already over")
)

type Game struct {
	engine *chess.Game
}

func New() *Game {
	return &Game{engine: chess.NewGame()}
}

func FromFEN(fen string) (*Game, error) {
	opt, err := chess.FEN(fen)
	if err != nil {
		return nil, fmt.Errorf("game: invalid fen %q: %w", fen, err)
	}
	return &Game{engine: chess.NewGame(opt)}, nil
}

func (g *Game) Move(uci string) error {
	if g.IsOver() {
		return ErrGameOver
	}
	move, err := chess.UCINotation{}.Decode(g.engine.Position(), uci)
	if err != nil {
		return ErrIllegalMove
	}
	if err := g.engine.Move(move); err != nil {
		return ErrIllegalMove
	}
	return nil
}

func (g *Game) Resign(c Color) error {
	if g.IsOver() {
		return ErrGameOver
	}
	g.engine.Resign(toEngineColor(c))
	return nil
}

func (g *Game) Draw() error {
	if g.IsOver() {
		return ErrGameOver
	}
	return g.engine.Draw(chess.DrawOffer)
}

func (g *Game) FEN() string {
	return g.engine.FEN()
}

func (g *Game) PGN() string {
	return g.engine.String()
}

func (g *Game) Turn() Color {
	return fromEngineColor(g.engine.Position().Turn())
}

func (g *Game) IsOver() bool {
	return g.engine.Outcome() != chess.NoOutcome
}

func (g *Game) Status() Status {
	switch g.engine.Method() {
	case chess.Checkmate:
		return Checkmate
	case chess.Stalemate:
		return Stalemate
	case chess.ThreefoldRepetition, chess.FivefoldRepetition:
		return DrawByRepetition
	case chess.FiftyMoveRule, chess.SeventyFiveMoveRule:
		return DrawByMoveRule
	case chess.InsufficientMaterial:
		return DrawByInsufficientMaterial
	case chess.DrawOffer:
		return DrawByAgreement
	case chess.Resignation:
		return Resigned
	default:
		return InProgress
	}
}

func (g *Game) Winner() (Color, bool) {
	switch g.engine.Outcome() {
	case chess.WhiteWon:
		return White, true
	case chess.BlackWon:
		return Black, true
	default:
		return "", false
	}
}

func (g *Game) ValidMoves() []string {
	moves := g.engine.ValidMoves()
	notation := chess.UCINotation{}
	uci := make([]string, len(moves))
	for i, m := range moves {
		uci[i] = notation.Encode(g.engine.Position(), m)
	}
	return uci
}

func fromEngineColor(c chess.Color) Color {
	if c == chess.Black {
		return Black
	}
	return White
}

func toEngineColor(c Color) chess.Color {
	if c == Black {
		return chess.Black
	}
	return chess.White
}
