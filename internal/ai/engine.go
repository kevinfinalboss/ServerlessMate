package ai

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/kevinfinalboss/ServerlessMate/internal/game"
)

var ErrNoMoves = errors.New("ai: no legal moves")

const PlayerID = "AI"

type Level string

const (
	LevelEasy Level = "easy"
	LevelHard Level = "hard"
)

type Engine interface {
	BestMove(fen string) (string, error)
}

func NewEngine(level Level) Engine {
	if level == LevelHard {
		return &MinimaxEngine{Depth: 3}
	}
	return HeuristicEngine{}
}

type HeuristicEngine struct{}

func (HeuristicEngine) BestMove(fen string) (string, error) {
	g, err := game.FromFEN(fen)
	if err != nil {
		return "", fmt.Errorf("ai: invalid fen: %w", err)
	}

	moves := g.ValidMoves()
	if len(moves) == 0 {
		return "", ErrNoMoves
	}

	maximizing := g.Turn() == game.White
	best := moves[0]
	bestScore := initialScore(maximizing)

	for _, uci := range moves {
		candidate, err := game.FromFEN(fen)
		if err != nil {
			return "", fmt.Errorf("ai: invalid fen: %w", err)
		}
		if err := candidate.Move(uci); err != nil {
			continue
		}
		score := evaluate(candidate.FEN())
		if better(score, bestScore, maximizing) {
			bestScore = score
			best = uci
		}
	}
	return best, nil
}

type MinimaxEngine struct {
	Depth int
}

func (e *MinimaxEngine) BestMove(fen string) (string, error) {
	g, err := game.FromFEN(fen)
	if err != nil {
		return "", fmt.Errorf("ai: invalid fen: %w", err)
	}

	moves := g.ValidMoves()
	if len(moves) == 0 {
		return "", ErrNoMoves
	}

	depth := e.Depth
	if depth <= 0 {
		depth = 3
	}

	maximizing := g.Turn() == game.White
	best := moves[0]
	bestScore := initialScore(maximizing)
	alpha, beta := math.Inf(-1), math.Inf(1)

	for _, uci := range moves {
		candidate, err := game.FromFEN(fen)
		if err != nil {
			return "", fmt.Errorf("ai: invalid fen: %w", err)
		}
		if err := candidate.Move(uci); err != nil {
			continue
		}

		score := minimax(candidate, depth-1, alpha, beta, !maximizing)
		if better(score, bestScore, maximizing) {
			bestScore = score
			best = uci
		}
		if maximizing {
			alpha = math.Max(alpha, bestScore)
		} else {
			beta = math.Min(beta, bestScore)
		}
	}
	return best, nil
}

func minimax(g *game.Game, depth int, alpha, beta float64, maximizing bool) float64 {
	if depth == 0 || g.IsOver() {
		return evaluate(g.FEN())
	}

	moves := g.ValidMoves()
	if len(moves) == 0 {
		return evaluate(g.FEN())
	}

	if maximizing {
		best := math.Inf(-1)
		for _, uci := range moves {
			candidate, err := game.FromFEN(g.FEN())
			if err != nil {
				continue
			}
			if err := candidate.Move(uci); err != nil {
				continue
			}
			score := minimax(candidate, depth-1, alpha, beta, false)
			best = math.Max(best, score)
			alpha = math.Max(alpha, best)
			if beta <= alpha {
				break
			}
		}
		return best
	}

	best := math.Inf(1)
	for _, uci := range moves {
		candidate, err := game.FromFEN(g.FEN())
		if err != nil {
			continue
		}
		if err := candidate.Move(uci); err != nil {
			continue
		}
		score := minimax(candidate, depth-1, alpha, beta, true)
		best = math.Min(best, score)
		beta = math.Min(beta, best)
		if beta <= alpha {
			break
		}
	}
	return best
}

func initialScore(maximizing bool) float64 {
	if maximizing {
		return math.Inf(-1)
	}
	return math.Inf(1)
}

func better(score, best float64, maximizing bool) bool {
	if maximizing {
		return score > best
	}
	return score < best
}

var pieceValues = map[byte]float64{
	'p': 1, 'n': 3, 'b': 3, 'r': 5, 'q': 9,
	'P': 1, 'N': 3, 'B': 3, 'R': 5, 'Q': 9,
}

const centerBonus = 0.1

func evaluate(fen string) float64 {
	fields := strings.Fields(fen)
	if len(fields) == 0 {
		return 0
	}

	score := 0.0
	rank, file := 7, 0
	for _, c := range fields[0] {
		switch {
		case c == '/':
			rank--
			file = 0
		case c >= '1' && c <= '8':
			file += int(c - '0')
		default:
			if v, ok := pieceValues[byte(c)]; ok {
				sign := 1.0
				if c >= 'a' && c <= 'z' {
					sign = -1
				}
				score += sign * v
				if (rank == 3 || rank == 4) && (file == 3 || file == 4) {
					score += sign * centerBonus
				}
			}
			file++
		}
	}
	return score
}
