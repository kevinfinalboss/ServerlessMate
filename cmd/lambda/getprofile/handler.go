package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

type request struct {
	PlayerID string `json:"playerId,omitempty"`
}

type response struct {
	PlayerID    string `json:"playerId"`
	Username    string `json:"username"`
	Visible     bool   `json:"visible"`
	Rating      int    `json:"rating,omitempty"`
	Wins        int    `json:"wins,omitempty"`
	Losses      int    `json:"losses,omitempty"`
	Draws       int    `json:"draws,omitempty"`
	GamesPlayed int    `json:"gamesPlayed,omitempty"`
}

type deps struct {
	connections store.ConnectionStore
	players     store.PlayerStore
	friendships store.FriendshipStore
	broadcaster ws.Broadcaster
}

func handle(ctx context.Context, d deps, connectionID string, body []byte) error {
	conn, err := d.connections.GetConnection(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("getprofile: load connection: %w", err)
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return notify(ctx, d, connectionID, "invalid request body")
	}

	targetID := req.PlayerID
	if targetID == "" {
		targetID = conn.PlayerID
	}

	target, err := d.players.GetPlayer(ctx, targetID)
	if err != nil {
		if errors.Is(err, store.ErrPlayerNotFound) {
			return notify(ctx, d, connectionID, "player not found")
		}
		return fmt.Errorf("getprofile: load player: %w", err)
	}

	visible, err := canViewFullProfile(ctx, d, conn.PlayerID, target)
	if err != nil {
		return fmt.Errorf("getprofile: check visibility: %w", err)
	}

	resp := response{PlayerID: target.PlayerID, Username: target.Username, Visible: visible}
	if visible {
		resp.Rating = target.Rating
		resp.Wins = target.Wins
		resp.Losses = target.Losses
		resp.Draws = target.Draws
		resp.GamesPlayed = target.GamesPlayed
	}

	return reply(ctx, d, connectionID, resp)
}

func canViewFullProfile(ctx context.Context, d deps, requesterID string, target *store.Player) (bool, error) {
	if requesterID == target.PlayerID {
		return true, nil
	}
	if target.Visibility != "friends" {
		return true, nil
	}

	f, err := d.friendships.GetFriendship(ctx, requesterID, target.PlayerID)
	if err != nil {
		return false, fmt.Errorf("load friendship: %w", err)
	}
	return f != nil && f.Status == store.FriendshipAccepted, nil
}

func reply(ctx context.Context, d deps, connectionID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("getprofile: marshal reply: %w", err)
	}
	if err := d.broadcaster.Send(ctx, connectionID, data); err != nil && !errors.Is(err, ws.ErrConnectionGone) {
		return fmt.Errorf("getprofile: send reply: %w", err)
	}
	return nil
}

func notify(ctx context.Context, d deps, connectionID, message string) error {
	return reply(ctx, d, connectionID, map[string]string{"type": "error", "message": message})
}
