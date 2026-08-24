package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

var validVisibilities = map[string]bool{"public": true, "friends": true}

type request struct {
	Action     string `json:"action,omitempty"`
	PlayerID   string `json:"playerId,omitempty"`
	Visibility string `json:"visibility,omitempty"`
	BirthDate  string `json:"birthDate,omitempty"`
	Country    string `json:"country,omitempty"`
	Github     string `json:"github,omitempty"`
	LinkedIn   string `json:"linkedIn,omitempty"`
}

type response struct {
	PlayerID         string `json:"playerId"`
	Username         string `json:"username"`
	Visible          bool   `json:"visible"`
	Rating           int    `json:"rating"`
	Wins             int    `json:"wins"`
	Losses           int    `json:"losses"`
	Draws            int    `json:"draws"`
	GamesPlayed      int    `json:"gamesPlayed"`
	Visibility       string `json:"visibility,omitempty"`
	BirthDate        string `json:"birthDate,omitempty"`
	Country          string `json:"country,omitempty"`
	Github           string `json:"github,omitempty"`
	LinkedIn         string `json:"linkedIn,omitempty"`
	FriendshipStatus string `json:"friendshipStatus,omitempty"`
}

type deps struct {
	connections store.ConnectionStore
	players     store.PlayerStore
	friendships store.FriendshipStore
	broadcaster ws.Broadcaster
	now         func() time.Time
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

	if req.Action == "updateProfile" {
		return handleUpdateProfile(ctx, d, connectionID, conn.PlayerID, req)
	}

	targetID := req.PlayerID
	if targetID == "" {
		targetID = conn.PlayerID
	}

	target, err := loadTarget(ctx, d, targetID, conn.PlayerID)
	if err != nil {
		if errors.Is(err, store.ErrPlayerNotFound) {
			return notify(ctx, d, connectionID, "player not found")
		}
		return fmt.Errorf("getprofile: load player: %w", err)
	}

	friendshipStatus, accepted, err := loadFriendshipStatus(ctx, d, conn.PlayerID, target.PlayerID)
	if err != nil {
		return fmt.Errorf("getprofile: load friendship status: %w", err)
	}

	visible := conn.PlayerID == target.PlayerID || target.Visibility != "friends" || accepted

	return reply(ctx, d, connectionID, buildResponse(target, visible, friendshipStatus))
}

func handleUpdateProfile(ctx context.Context, d deps, connectionID, playerID string, req request) error {
	if req.Visibility != "" && !validVisibilities[req.Visibility] {
		return notify(ctx, d, connectionID, "invalid visibility")
	}

	updated, err := d.players.UpdateProfile(ctx, playerID, store.ProfileUpdate{
		Visibility: req.Visibility,
		BirthDate:  req.BirthDate,
		Country:    req.Country,
		Github:     req.Github,
		LinkedIn:   req.LinkedIn,
	})
	if err != nil {
		if errors.Is(err, store.ErrPlayerNotFound) {
			return notify(ctx, d, connectionID, "player not found")
		}
		return fmt.Errorf("getprofile: update profile: %w", err)
	}

	return reply(ctx, d, connectionID, buildResponse(updated, true, ""))
}

func buildResponse(target *store.Player, visible bool, friendshipStatus string) response {
	resp := response{PlayerID: target.PlayerID, Username: target.Username, Visible: visible, FriendshipStatus: friendshipStatus}
	if visible {
		resp.Rating = target.Rating
		resp.Wins = target.Wins
		resp.Losses = target.Losses
		resp.Draws = target.Draws
		resp.GamesPlayed = target.GamesPlayed
		resp.Visibility = target.Visibility
		resp.BirthDate = target.BirthDate
		resp.Country = target.Country
		resp.Github = target.Github
		resp.LinkedIn = target.LinkedIn
	}
	return resp
}

func loadTarget(ctx context.Context, d deps, targetID, requesterID string) (*store.Player, error) {
	if targetID == requesterID {
		return d.players.GetOrCreatePlayer(ctx, targetID, d.now().UnixMilli())
	}
	return d.players.GetPlayer(ctx, targetID)
}

func loadFriendshipStatus(ctx context.Context, d deps, requesterID, targetID string) (status string, accepted bool, err error) {
	if requesterID == targetID {
		return "", false, nil
	}

	outgoing, err := d.friendships.GetFriendship(ctx, requesterID, targetID)
	if err != nil {
		return "", false, fmt.Errorf("load friendship: %w", err)
	}
	if outgoing != nil {
		switch outgoing.Status {
		case store.FriendshipAccepted:
			return string(store.FriendshipAccepted), true, nil
		case store.FriendshipBlocked:
			return string(store.FriendshipBlocked), false, nil
		default:
			return "pendingOutgoing", false, nil
		}
	}

	incoming, err := d.friendships.GetFriendship(ctx, targetID, requesterID)
	if err != nil {
		return "", false, fmt.Errorf("load friendship: %w", err)
	}
	if incoming != nil && incoming.Status == store.FriendshipPending {
		return "pendingIncoming", false, nil
	}
	return "", false, nil
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
