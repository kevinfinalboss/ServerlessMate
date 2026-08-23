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

type action string

const (
	actionSendRequest   action = "sendRequest"
	actionAcceptRequest action = "acceptRequest"
	actionBlock         action = "block"
)

type request struct {
	Action   action `json:"action"`
	FriendID string `json:"friendId"`
}

type deps struct {
	connections store.ConnectionStore
	friendships store.FriendshipStore
	broadcaster ws.Broadcaster
	now         func() time.Time
}

func handle(ctx context.Context, d deps, connectionID string, body []byte) error {
	conn, err := d.connections.GetConnection(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("friends: load connection: %w", err)
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return notify(ctx, d, connectionID, "invalid request body")
	}
	if req.FriendID == "" || req.FriendID == conn.PlayerID {
		return notify(ctx, d, connectionID, "invalid friendId")
	}

	at := d.now().UnixMilli()

	switch req.Action {
	case actionSendRequest:
		return handleSendRequest(ctx, d, connectionID, conn.PlayerID, req.FriendID, at)
	case actionAcceptRequest:
		return handleAcceptRequest(ctx, d, connectionID, conn.PlayerID, req.FriendID, at)
	case actionBlock:
		return handleBlock(ctx, d, connectionID, conn.PlayerID, req.FriendID, at)
	default:
		return notify(ctx, d, connectionID, "unknown action")
	}
}

func handleSendRequest(ctx context.Context, d deps, connectionID, playerID, friendID string, at int64) error {
	blockedByTarget, err := d.friendships.GetFriendship(ctx, friendID, playerID)
	if err != nil {
		return fmt.Errorf("friends: check block: %w", err)
	}
	if blockedByTarget != nil && blockedByTarget.Status == store.FriendshipBlocked {
		return notify(ctx, d, connectionID, "cannot send request to this player")
	}

	if err := d.friendships.SendRequest(ctx, playerID, friendID, at); err != nil {
		if errors.Is(err, store.ErrFriendshipConflict) {
			return notify(ctx, d, connectionID, "already friends or blocked")
		}
		return fmt.Errorf("friends: send request: %w", err)
	}
	return reply(ctx, d, connectionID, map[string]string{"type": "friendRequestSent", "friendId": friendID})
}

func handleAcceptRequest(ctx context.Context, d deps, connectionID, accepterID, requesterID string, at int64) error {
	pending, err := d.friendships.GetFriendship(ctx, requesterID, accepterID)
	if err != nil {
		return fmt.Errorf("friends: load request: %w", err)
	}
	if pending == nil || pending.Status != store.FriendshipPending {
		return notify(ctx, d, connectionID, "no pending request from this player")
	}

	if err := d.friendships.AcceptRequest(ctx, accepterID, requesterID, at); err != nil {
		return fmt.Errorf("friends: accept request: %w", err)
	}
	return reply(ctx, d, connectionID, map[string]string{"type": "friendRequestAccepted", "friendId": requesterID})
}

func handleBlock(ctx context.Context, d deps, connectionID, playerID, friendID string, at int64) error {
	if err := d.friendships.Block(ctx, playerID, friendID, at); err != nil {
		return fmt.Errorf("friends: block: %w", err)
	}
	return reply(ctx, d, connectionID, map[string]string{"type": "playerBlocked", "friendId": friendID})
}

func reply(ctx context.Context, d deps, connectionID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("friends: marshal reply: %w", err)
	}
	if err := d.broadcaster.Send(ctx, connectionID, data); err != nil && !errors.Is(err, ws.ErrConnectionGone) {
		return fmt.Errorf("friends: send reply: %w", err)
	}
	return nil
}

func notify(ctx context.Context, d deps, connectionID, message string) error {
	return reply(ctx, d, connectionID, map[string]string{"type": "error", "message": message})
}
