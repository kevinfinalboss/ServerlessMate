package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kevinfinalboss/ServerlessMate/internal/game"
	"github.com/kevinfinalboss/ServerlessMate/internal/rating"
	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

type action string

const (
	actionMove       action = "move"
	actionResign     action = "resign"
	actionOfferDraw  action = "offerDraw"
	actionAcceptDraw action = "acceptDraw"
	actionChat       action = "chat"
)

const statusAbandoned = "abandoned"

type request struct {
	Action  action `json:"action"`
	Move    string `json:"move,omitempty"`
	Message string `json:"message,omitempty"`
}

type deps struct {
	games       store.GameStore
	connections store.ConnectionStore
	players     store.PlayerStore
	history     store.HistoryStore
	broadcaster ws.Broadcaster
	now         func() time.Time
	graceMs     int64
}

func handle(ctx context.Context, d deps, connectionID string, body []byte) error {
	conn, err := d.connections.GetConnection(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("makemove: load connection: %w", err)
	}
	if conn.Role != store.RolePlayer {
		return notify(ctx, d, connectionID, "spectators cannot act")
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return notify(ctx, d, connectionID, "invalid request body")
	}

	g, err := d.games.GetGame(ctx, conn.GameID)
	if err != nil {
		return fmt.Errorf("makemove: load game: %w", err)
	}
	if g.Status != string(game.InProgress) {
		return notify(ctx, d, connectionID, "game is already over")
	}

	color, ok := colorForPlayer(g.Players, conn.PlayerID)
	if !ok {
		return fmt.Errorf("makemove: player %s not part of game %s", conn.PlayerID, g.GameID)
	}

	expectedFEN := g.FEN
	engine, err := game.FromFEN(g.FEN)
	if err != nil {
		return fmt.Errorf("makemove: rebuild engine from fen %q: %w", g.FEN, err)
	}

	now := d.now()

	if g.DisconnectedPlayerID != "" && now.UnixMilli()-g.DisconnectedAt > d.graceMs {
		finishByAbandonment(g, g.DisconnectedPlayerID, now)
		return commit(ctx, d, connectionID, g, expectedFEN)
	}

	if req.Action == actionMove {
		if timedOut := deductClock(g, engine.Turn(), now); timedOut {
			finishByTimeout(g, engine.Turn(), now)
			return commit(ctx, d, connectionID, g, expectedFEN)
		}
	}

	switch req.Action {
	case actionMove:
		if engine.Turn() != color {
			return notify(ctx, d, connectionID, "not your turn")
		}
		if err := engine.Move(req.Move); err != nil {
			return notify(ctx, d, connectionID, "illegal move")
		}
		applyEngineState(g, engine, now)
	case actionResign:
		if err := engine.Resign(color); err != nil {
			return notify(ctx, d, connectionID, "game is already over")
		}
		applyEngineState(g, engine, now)
	case actionOfferDraw:
		g.DrawOfferedBy = conn.PlayerID
	case actionAcceptDraw:
		if g.DrawOfferedBy == "" {
			return notify(ctx, d, connectionID, "no draw offer to accept")
		}
		if g.DrawOfferedBy == conn.PlayerID {
			return notify(ctx, d, connectionID, "cannot accept your own draw offer")
		}
		if err := engine.Draw(); err != nil {
			return notify(ctx, d, connectionID, "game is already over")
		}
		g.DrawOfferedBy = ""
		applyEngineState(g, engine, now)
	case actionChat:
		return broadcastChat(ctx, d, connectionID, g.GameID, conn.PlayerID, req.Message)
	default:
		return notify(ctx, d, connectionID, "unknown action")
	}

	return commit(ctx, d, connectionID, g, expectedFEN)
}

func broadcastChat(ctx context.Context, d deps, connectionID, gameID, playerID, message string) error {
	if strings.TrimSpace(message) == "" {
		return notify(ctx, d, connectionID, "empty message")
	}

	conns, err := d.connections.ListConnectionsByGame(ctx, gameID)
	if err != nil {
		return fmt.Errorf("makemove: list connections: %w", err)
	}

	payload, err := json.Marshal(map[string]string{"type": "chat", "playerId": playerID, "message": message})
	if err != nil {
		return fmt.Errorf("makemove: marshal chat: %w", err)
	}

	ids := make([]string, len(conns))
	for i, c := range conns {
		ids[i] = c.ConnectionID
	}
	for _, id := range ws.BroadcastAll(ctx, d.broadcaster, ids, payload) {
		_ = d.connections.DeleteConnection(ctx, id)
	}
	return nil
}

func commit(ctx context.Context, d deps, connectionID string, g *store.Game, expectedFEN string) error {
	if err := d.games.UpdateGame(ctx, g, expectedFEN); err != nil {
		if errors.Is(err, store.ErrConcurrentUpdate) {
			return notify(ctx, d, connectionID, "game changed, please retry")
		}
		return fmt.Errorf("makemove: persist game: %w", err)
	}

	if g.Status != string(game.InProgress) {
		if !g.VsAI {
			if err := updateRatings(ctx, d, g); err != nil {
				return fmt.Errorf("makemove: update ratings: %w", err)
			}
		}
		if err := d.history.RecordGameEnd(ctx, g); err != nil {
			return fmt.Errorf("makemove: record history: %w", err)
		}
	}

	conns, err := d.connections.ListConnectionsByGame(ctx, g.GameID)
	if err != nil {
		return fmt.Errorf("makemove: list connections: %w", err)
	}

	payload, err := json.Marshal(g)
	if err != nil {
		return fmt.Errorf("makemove: marshal state: %w", err)
	}

	ids := make([]string, len(conns))
	for i, c := range conns {
		ids[i] = c.ConnectionID
	}
	for _, id := range ws.BroadcastAll(ctx, d.broadcaster, ids, payload) {
		_ = d.connections.DeleteConnection(ctx, id)
	}
	return nil
}

func updateRatings(ctx context.Context, d deps, g *store.Game) error {
	white, err := d.players.GetPlayer(ctx, g.Players.White)
	if err != nil {
		return fmt.Errorf("load white player: %w", err)
	}
	black, err := d.players.GetPlayer(ctx, g.Players.Black)
	if err != nil {
		return fmt.Errorf("load black player: %w", err)
	}

	if g.Status == statusAbandoned {
		return recordAbandonment(ctx, d, g, white, black)
	}

	result := rating.Draw
	switch g.Winner {
	case g.Players.White:
		result = rating.WhiteWon
	case g.Players.Black:
		result = rating.BlackWon
	}

	newWhiteRating, newBlackRating := rating.Apply(white.Rating, black.Rating, result)

	if err := d.players.RecordGameResult(ctx, g.Players.White, newWhiteRating, outcomeFor(result, true)); err != nil {
		return fmt.Errorf("record white result: %w", err)
	}
	if err := d.players.RecordGameResult(ctx, g.Players.Black, newBlackRating, outcomeFor(result, false)); err != nil {
		return fmt.Errorf("record black result: %w", err)
	}
	return nil
}

func recordAbandonment(ctx context.Context, d deps, g *store.Game, white, black *store.Player) error {
	winnerID, winnerRating, loserID, loserRating := white.PlayerID, white.Rating, black.PlayerID, black.Rating
	if g.Winner == black.PlayerID {
		winnerID, winnerRating, loserID, loserRating = black.PlayerID, black.Rating, white.PlayerID, white.Rating
	}

	newWinnerRating, newLoserRating := rating.ApplyAbandonment(winnerRating, loserRating)

	if err := d.players.RecordGameResult(ctx, winnerID, newWinnerRating, store.OutcomeWin); err != nil {
		return fmt.Errorf("record winner result: %w", err)
	}
	if err := d.players.RecordGameResult(ctx, loserID, newLoserRating, store.OutcomeLoss); err != nil {
		return fmt.Errorf("record loser result: %w", err)
	}
	return nil
}

func outcomeFor(result rating.Result, isWhite bool) store.GameOutcome {
	switch result {
	case rating.WhiteWon:
		if isWhite {
			return store.OutcomeWin
		}
		return store.OutcomeLoss
	case rating.BlackWon:
		if isWhite {
			return store.OutcomeLoss
		}
		return store.OutcomeWin
	default:
		return store.OutcomeDraw
	}
}

func notify(ctx context.Context, d deps, connectionID, message string) error {
	payload, err := json.Marshal(map[string]string{"type": "error", "message": message})
	if err != nil {
		return fmt.Errorf("makemove: marshal notification: %w", err)
	}
	if err := d.broadcaster.Send(ctx, connectionID, payload); err != nil && !errors.Is(err, ws.ErrConnectionGone) {
		return fmt.Errorf("makemove: notify sender: %w", err)
	}
	return nil
}

func deductClock(g *store.Game, turn game.Color, now time.Time) bool {
	elapsed := now.UnixMilli() - g.LastMoveAt
	if turn == game.White {
		g.WhiteTimeMs -= elapsed
		return g.WhiteTimeMs <= 0
	}
	g.BlackTimeMs -= elapsed
	return g.BlackTimeMs <= 0
}

func finishByTimeout(g *store.Game, timedOut game.Color, now time.Time) {
	g.Status = "timeout"
	g.EndedAt = now.UnixMilli()
	g.TurnOf = ""
	if timedOut == game.White {
		g.WhiteTimeMs = 0
		g.Winner = g.Players.Black
	} else {
		g.BlackTimeMs = 0
		g.Winner = g.Players.White
	}
}

func finishByAbandonment(g *store.Game, abandonedBy string, now time.Time) {
	g.Status = statusAbandoned
	g.EndedAt = now.UnixMilli()
	g.TurnOf = ""
	g.DisconnectedPlayerID = ""
	g.DisconnectedAt = 0
	if abandonedBy == g.Players.White {
		g.Winner = g.Players.Black
	} else {
		g.Winner = g.Players.White
	}
}

func applyEngineState(g *store.Game, engine *game.Game, now time.Time) {
	g.FEN = engine.FEN()
	g.PGN = engine.PGN()
	g.Status = string(engine.Status())
	g.LastMoveAt = now.UnixMilli()
	if engine.IsOver() {
		g.EndedAt = now.UnixMilli()
		g.TurnOf = ""
		if winner, ok := engine.Winner(); ok {
			g.Winner = playerIDForColor(g.Players, winner)
		}
		return
	}
	g.TurnOf = playerIDForColor(g.Players, engine.Turn())
}

func colorForPlayer(p store.Players, playerID string) (game.Color, bool) {
	switch playerID {
	case p.White:
		return game.White, true
	case p.Black:
		return game.Black, true
	default:
		return "", false
	}
}

func playerIDForColor(p store.Players, c game.Color) string {
	if c == game.White {
		return p.White
	}
	return p.Black
}
