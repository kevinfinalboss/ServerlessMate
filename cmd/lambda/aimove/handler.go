package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kevinfinalboss/ServerlessMate/internal/ai"
	"github.com/kevinfinalboss/ServerlessMate/internal/game"
	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

const (
	startFEN          = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	defaultTimeMs     = 300_000
	dailyBedrockLimit = 200
)

type actionType string

const (
	actionStart actionType = "start"
	actionMove  actionType = "move"
)

type request struct {
	Action actionType `json:"action"`
	Level  ai.Level   `json:"level,omitempty"`
}

type moveResponse struct {
	*store.Game
	Comment string `json:"comment,omitempty"`
}

type deps struct {
	games       store.GameStore
	connections store.ConnectionStore
	rateLimits  store.RateLimitStore
	history     store.HistoryStore
	commentator ai.Commentator
	broadcaster ws.Broadcaster
	newGameID   func() string
	now         func() time.Time
}

func handle(ctx context.Context, d deps, connectionID string, body []byte) error {
	conn, err := d.connections.GetConnection(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("aimove: load connection: %w", err)
	}
	if conn.IsGuest {
		return notify(ctx, d, connectionID, "guests cannot play against AI")
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return notify(ctx, d, connectionID, "invalid request body")
	}

	switch req.Action {
	case actionStart:
		return handleStart(ctx, d, connectionID, conn.PlayerID, req.Level)
	case actionMove:
		return handleMove(ctx, d, connectionID, conn)
	default:
		return notify(ctx, d, connectionID, "unknown action")
	}
}

func handleStart(ctx context.Context, d deps, connectionID, playerID string, level ai.Level) error {
	if level != ai.LevelEasy && level != ai.LevelHard {
		return notify(ctx, d, connectionID, "invalid level")
	}

	now := d.now().UnixMilli()
	g := &store.Game{
		GameID:      d.newGameID(),
		FEN:         startFEN,
		Status:      "in_progress",
		Players:     store.Players{White: playerID, Black: ai.PlayerID},
		TurnOf:      playerID,
		VsAI:        true,
		AILevel:     string(level),
		WhiteTimeMs: defaultTimeMs,
		BlackTimeMs: defaultTimeMs,
		LastMoveAt:  now,
	}

	if err := d.games.CreateGame(ctx, g); err != nil {
		return fmt.Errorf("aimove: create game: %w", err)
	}

	if err := d.connections.PutConnection(ctx, &store.Connection{
		ConnectionID: connectionID,
		GameID:       g.GameID,
		PlayerID:     playerID,
		Role:         store.RolePlayer,
	}); err != nil {
		return fmt.Errorf("aimove: assign connection: %w", err)
	}

	return reply(ctx, d, connectionID, moveResponse{Game: g})
}

func handleMove(ctx context.Context, d deps, connectionID string, conn *store.Connection) error {
	if conn.GameID == "" {
		return notify(ctx, d, connectionID, "no active game")
	}

	g, err := d.games.GetGame(ctx, conn.GameID)
	if err != nil {
		return fmt.Errorf("aimove: load game: %w", err)
	}
	if !g.VsAI || g.Status != "in_progress" {
		return notify(ctx, d, connectionID, "no active AI game")
	}

	expectedFEN := g.FEN
	gameEngine, err := game.FromFEN(g.FEN)
	if err != nil {
		return fmt.Errorf("aimove: rebuild engine: %w", err)
	}
	if gameEngine.Turn() != game.Black {
		return notify(ctx, d, connectionID, "not AI's turn")
	}

	now := d.now()
	elapsed := now.UnixMilli() - g.LastMoveAt
	g.BlackTimeMs -= elapsed
	if g.BlackTimeMs <= 0 {
		g.BlackTimeMs = 0
		g.Status = "timeout"
		g.Winner = g.Players.White
		g.EndedAt = now.UnixMilli()
		g.TurnOf = ""
		if err := d.games.UpdateGame(ctx, g, expectedFEN); err != nil {
			return fmt.Errorf("aimove: persist timeout: %w", err)
		}
		if err := d.history.RecordGameEnd(ctx, g); err != nil {
			return fmt.Errorf("aimove: record history: %w", err)
		}
		return reply(ctx, d, connectionID, moveResponse{Game: g})
	}

	engine := ai.NewEngine(ai.Level(g.AILevel))
	uci, err := engine.BestMove(g.FEN)
	if err != nil {
		return fmt.Errorf("aimove: compute best move: %w", err)
	}
	if err := gameEngine.Move(uci); err != nil {
		return fmt.Errorf("aimove: apply computed move %q: %w", uci, err)
	}

	g.FEN = gameEngine.FEN()
	g.PGN = gameEngine.PGN()
	g.Status = string(gameEngine.Status())
	g.LastMoveAt = now.UnixMilli()
	if gameEngine.IsOver() {
		g.EndedAt = now.UnixMilli()
		g.TurnOf = ""
		if winner, ok := gameEngine.Winner(); ok {
			g.Winner = playerIDForColor(g.Players, winner)
		}
	} else {
		g.TurnOf = g.Players.White
	}

	if err := d.games.UpdateGame(ctx, g, expectedFEN); err != nil {
		return fmt.Errorf("aimove: persist move: %w", err)
	}
	if g.Status != "in_progress" {
		if err := d.history.RecordGameEnd(ctx, g); err != nil {
			return fmt.Errorf("aimove: record history: %w", err)
		}
	}

	comment := commentOnMove(ctx, d, g.Players.White, g.FEN, uci)
	return reply(ctx, d, connectionID, moveResponse{Game: g, Comment: comment})
}

func commentOnMove(ctx context.Context, d deps, playerID, fen, uci string) string {
	now := d.now()
	allowed, err := d.rateLimits.IncrementAndCheck(ctx, playerID, now.Format("2006-01-02"), dailyBedrockLimit, endOfDayUnix(now))
	if err != nil || !allowed {
		return ""
	}

	comment, err := d.commentator.Comment(ctx, fen, uci)
	if err != nil {
		return ""
	}
	return comment
}

func endOfDayUnix(t time.Time) int64 {
	next := time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, time.UTC)
	return next.Unix()
}

func playerIDForColor(p store.Players, c game.Color) string {
	if c == game.White {
		return p.White
	}
	return p.Black
}

func reply(ctx context.Context, d deps, connectionID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("aimove: marshal reply: %w", err)
	}
	if err := d.broadcaster.Send(ctx, connectionID, data); err != nil && !errors.Is(err, ws.ErrConnectionGone) {
		return fmt.Errorf("aimove: send reply: %w", err)
	}
	return nil
}

func notify(ctx context.Context, d deps, connectionID, message string) error {
	return reply(ctx, d, connectionID, map[string]string{"type": "error", "message": message})
}
