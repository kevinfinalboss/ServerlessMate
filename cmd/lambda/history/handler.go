package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type request struct {
	GameID string `json:"gameId,omitempty"`
	Limit  int32  `json:"limit,omitempty"`
}

type deps struct {
	connections store.ConnectionStore
	games       store.GameStore
	history     store.HistoryStore
	players     store.PlayerStore
	broadcaster ws.Broadcaster
}

func handle(ctx context.Context, d deps, connectionID string, body []byte) error {
	conn, err := d.connections.GetConnection(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("history: load connection: %w", err)
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return notify(ctx, d, connectionID, "invalid request body")
	}

	if req.GameID != "" {
		return handleReplay(ctx, d, connectionID, conn.PlayerID, req.GameID)
	}
	return handleList(ctx, d, connectionID, conn.PlayerID, req.Limit)
}

func handleReplay(ctx context.Context, d deps, connectionID, playerID, gameID string) error {
	g, err := d.games.GetGame(ctx, gameID)
	if err != nil {
		if errors.Is(err, store.ErrGameNotFound) {
			return notify(ctx, d, connectionID, "game not found")
		}
		return fmt.Errorf("history: load game: %w", err)
	}
	if g.Players.White != playerID && g.Players.Black != playerID {
		return notify(ctx, d, connectionID, "not your game")
	}

	return reply(ctx, d, connectionID, map[string]string{"type": "replay", "gameId": g.GameID, "pgn": g.PGN})
}

type historyEntryResponse struct {
	store.HistoryEntry
	OpponentUsername string `json:"opponentUsername"`
}

func handleList(ctx context.Context, d deps, connectionID, playerID string, limit int32) error {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	entries, err := d.history.ListHistory(ctx, playerID, limit)
	if err != nil {
		return fmt.Errorf("history: list history: %w", err)
	}

	usernames := make(map[string]string)
	resp := make([]historyEntryResponse, len(entries))
	for i, e := range entries {
		username := ""
		if !e.VsAI {
			username = resolveUsername(ctx, d, usernames, e.OpponentID)
		}
		resp[i] = historyEntryResponse{HistoryEntry: *e, OpponentUsername: username}
	}

	return reply(ctx, d, connectionID, map[string]any{"type": "history", "entries": resp})
}

func resolveUsername(ctx context.Context, d deps, cache map[string]string, playerID string) string {
	if username, ok := cache[playerID]; ok {
		return username
	}
	username := playerID
	if p, err := d.players.GetPlayer(ctx, playerID); err == nil {
		username = p.Username
	}
	cache[playerID] = username
	return username
}

func reply(ctx context.Context, d deps, connectionID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("history: marshal reply: %w", err)
	}
	if err := d.broadcaster.Send(ctx, connectionID, data); err != nil && !errors.Is(err, ws.ErrConnectionGone) {
		return fmt.Errorf("history: send reply: %w", err)
	}
	return nil
}

func notify(ctx context.Context, d deps, connectionID, message string) error {
	return reply(ctx, d, connectionID, map[string]string{"type": "error", "message": message})
}
