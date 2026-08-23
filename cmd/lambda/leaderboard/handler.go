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
	defaultLimit = 10
	maxLimit     = 100
)

type request struct {
	Limit int32 `json:"limit,omitempty"`
}

type entry struct {
	PlayerID string `json:"playerId"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
}

type deps struct {
	players     store.PlayerStore
	broadcaster ws.Broadcaster
}

func handle(ctx context.Context, d deps, connectionID string, body []byte) error {
	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return notify(ctx, d, connectionID, "invalid request body")
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	players, err := d.players.ListTopByRating(ctx, limit)
	if err != nil {
		return fmt.Errorf("leaderboard: list top players: %w", err)
	}

	entries := make([]entry, len(players))
	for i, p := range players {
		entries[i] = entry{PlayerID: p.PlayerID, Username: p.Username, Rating: p.Rating}
	}

	return reply(ctx, d, connectionID, map[string]any{"type": "leaderboard", "entries": entries})
}

func reply(ctx context.Context, d deps, connectionID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("leaderboard: marshal reply: %w", err)
	}
	if err := d.broadcaster.Send(ctx, connectionID, data); err != nil && !errors.Is(err, ws.ErrConnectionGone) {
		return fmt.Errorf("leaderboard: send reply: %w", err)
	}
	return nil
}

func notify(ctx context.Context, d deps, connectionID, message string) error {
	return reply(ctx, d, connectionID, map[string]string{"type": "error", "message": message})
}
