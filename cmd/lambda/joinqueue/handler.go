package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

const (
	defaultGuestBand = "1200"
	bandWidth        = 200
)

var allowedTimeControls = map[string]bool{
	"3+0":  true,
	"5+0":  true,
	"10+0": true,
}

type request struct {
	TimeControl string `json:"timeControl"`
}

type deps struct {
	connections store.ConnectionStore
	players     store.PlayerStore
	queue       store.QueueStore
	broadcaster ws.Broadcaster
	now         func() time.Time
}

func handle(ctx context.Context, d deps, connectionID string, body []byte) error {
	conn, err := d.connections.GetConnection(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("joinqueue: load connection: %w", err)
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return notify(ctx, d, connectionID, "invalid request body")
	}
	if !allowedTimeControls[req.TimeControl] {
		return notify(ctx, d, connectionID, "unsupported time control")
	}

	band := defaultGuestBand
	if !conn.IsGuest {
		player, err := d.players.GetPlayer(ctx, conn.PlayerID)
		if err != nil {
			return fmt.Errorf("joinqueue: load player: %w", err)
		}
		band = ratingBand(player.Rating)
	}

	matchmakingKey := req.TimeControl + "#" + band
	now := d.now().UnixMilli()

	entry := store.NewQueueEntry(matchmakingKey, conn.PlayerID, connectionID, conn.IsGuest, now)
	if err := d.queue.Join(ctx, entry); err != nil {
		return fmt.Errorf("joinqueue: join queue: %w", err)
	}

	return reply(ctx, d, connectionID, map[string]string{"type": "queueJoined", "matchmakingKey": matchmakingKey})
}

func ratingBand(rating int) string {
	band := (rating / bandWidth) * bandWidth
	return strconv.Itoa(band)
}

func reply(ctx context.Context, d deps, connectionID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("joinqueue: marshal reply: %w", err)
	}
	if err := d.broadcaster.Send(ctx, connectionID, data); err != nil && !errors.Is(err, ws.ErrConnectionGone) {
		return fmt.Errorf("joinqueue: send reply: %w", err)
	}
	return nil
}

func notify(ctx context.Context, d deps, connectionID, message string) error {
	return reply(ctx, d, connectionID, map[string]string{"type": "error", "message": message})
}
