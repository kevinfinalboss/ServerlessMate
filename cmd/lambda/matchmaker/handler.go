package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
const bandWidth = 200

type deps struct {
	queue       store.QueueStore
	match       store.MatchStore
	connections store.ConnectionStore
	broadcaster ws.Broadcaster
	now         func() time.Time
	newGameID   func() string
}

func handle(ctx context.Context, d deps, self *store.QueueEntry) error {
	waiting, err := findWaitingAcrossBands(ctx, d, self)
	if err != nil {
		return fmt.Errorf("matchmaker: find waiting player: %w", err)
	}
	if waiting == nil {
		return nil
	}

	now := d.now().UnixMilli()
	timeMs := parseTimeControlMs(self.MatchmakingKey)

	game := &store.Game{
		GameID:      d.newGameID(),
		FEN:         startFEN,
		Status:      "in_progress",
		Players:     store.Players{White: waiting.PlayerID, Black: self.PlayerID},
		TurnOf:      waiting.PlayerID,
		WhiteTimeMs: timeMs,
		BlackTimeMs: timeMs,
		LastMoveAt:  now,
	}

	if err := d.match.CreateMatch(ctx, game, waiting, self); err != nil {
		if errors.Is(err, store.ErrMatchClaimFailed) {
			return nil
		}
		return fmt.Errorf("matchmaker: create match: %w", err)
	}

	if err := assignConnection(ctx, d, waiting, game.GameID); err != nil {
		return fmt.Errorf("matchmaker: assign white connection: %w", err)
	}
	if err := assignConnection(ctx, d, self, game.GameID); err != nil {
		return fmt.Errorf("matchmaker: assign black connection: %w", err)
	}

	return notifyBoth(ctx, d, waiting, self, game)
}

func findWaitingAcrossBands(ctx context.Context, d deps, self *store.QueueEntry) (*store.QueueEntry, error) {
	for _, key := range candidateMatchmakingKeys(self.MatchmakingKey) {
		waiting, err := d.queue.FindWaiting(ctx, key, self.PlayerID)
		if err != nil {
			return nil, err
		}
		if waiting != nil {
			return waiting, nil
		}
	}
	return nil, nil
}

func candidateMatchmakingKeys(matchmakingKey string) []string {
	timeControl, bandStr, found := strings.Cut(matchmakingKey, "#")
	if !found {
		return []string{matchmakingKey}
	}
	band, err := strconv.ParseInt(bandStr, 10, 64)
	if err != nil {
		return []string{matchmakingKey}
	}

	keys := []string{matchmakingKey}
	if band-bandWidth >= 0 {
		keys = append(keys, fmt.Sprintf("%s#%d", timeControl, band-bandWidth))
	}
	keys = append(keys, fmt.Sprintf("%s#%d", timeControl, band+bandWidth))
	return keys
}

func assignConnection(ctx context.Context, d deps, e *store.QueueEntry, gameID string) error {
	return d.connections.PutConnection(ctx, &store.Connection{
		ConnectionID: e.ConnectionID,
		GameID:       gameID,
		PlayerID:     e.PlayerID,
		IsGuest:      e.IsGuest,
		Role:         store.RolePlayer,
	})
}

func notifyBoth(ctx context.Context, d deps, white, black *store.QueueEntry, game *store.Game) error {
	payload, err := json.Marshal(game)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	if err := d.broadcaster.Send(ctx, white.ConnectionID, payload); err != nil && !errors.Is(err, ws.ErrConnectionGone) {
		return fmt.Errorf("notify white: %w", err)
	}
	if err := d.broadcaster.Send(ctx, black.ConnectionID, payload); err != nil && !errors.Is(err, ws.ErrConnectionGone) {
		return fmt.Errorf("notify black: %w", err)
	}
	return nil
}

func parseTimeControlMs(matchmakingKey string) int64 {
	minutesPart, _, found := strings.Cut(matchmakingKey, "+")
	if !found {
		return 0
	}
	minutes, err := strconv.ParseInt(minutesPart, 10, 64)
	if err != nil {
		return 0
	}
	return minutes * 60_000
}
