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
	actionListFriends   action = "listFriends"
	actionCancelRequest action = "cancelRequest"
)

type request struct {
	Action   action `json:"action"`
	FriendID string `json:"friendId,omitempty"`
}

type deps struct {
	connections store.ConnectionStore
	friendships store.FriendshipStore
	players     store.PlayerStore
	broadcaster ws.Broadcaster
	now         func() time.Time
}

func handle(ctx context.Context, d deps, connectionID string, body []byte) error {
	conn, err := d.connections.GetConnection(ctx, connectionID)
	if err != nil {
		return fmt.Errorf("friends: load connection: %w", err)
	}
	if conn.IsGuest {
		return notify(ctx, d, connectionID, "guests cannot use friends")
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		return notify(ctx, d, connectionID, "invalid request body")
	}

	if req.Action == actionListFriends {
		return handleListFriends(ctx, d, connectionID, conn.PlayerID)
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
	case actionCancelRequest:
		return handleCancelRequest(ctx, d, connectionID, conn.PlayerID, req.FriendID)
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

func handleCancelRequest(ctx context.Context, d deps, connectionID, playerID, friendID string) error {
	if err := d.friendships.CancelRequest(ctx, playerID, friendID); err != nil {
		return fmt.Errorf("friends: cancel request: %w", err)
	}
	return reply(ctx, d, connectionID, map[string]string{"type": "friendRequestCancelled", "friendId": friendID})
}

type friendEntry struct {
	PlayerID string `json:"playerId"`
	Username string `json:"username"`
}

type friendsResponse struct {
	Type     string        `json:"type"`
	Friends  []friendEntry `json:"friends"`
	Incoming []friendEntry `json:"incomingRequests"`
	Outgoing []friendEntry `json:"outgoingRequests"`
}

func handleListFriends(ctx context.Context, d deps, connectionID, playerID string) error {
	own, err := d.friendships.ListFriendships(ctx, playerID)
	if err != nil {
		return fmt.Errorf("friends: list friendships: %w", err)
	}
	incoming, err := d.friendships.ListIncomingRequests(ctx, playerID)
	if err != nil {
		return fmt.Errorf("friends: list incoming requests: %w", err)
	}

	resp := friendsResponse{Type: "friends", Friends: []friendEntry{}, Incoming: []friendEntry{}, Outgoing: []friendEntry{}}
	for _, f := range own {
		switch f.Status {
		case store.FriendshipAccepted:
			resp.Friends = append(resp.Friends, resolveFriendEntry(ctx, d, f.FriendID))
		case store.FriendshipPending:
			resp.Outgoing = append(resp.Outgoing, resolveFriendEntry(ctx, d, f.FriendID))
		}
	}
	for _, f := range incoming {
		resp.Incoming = append(resp.Incoming, resolveFriendEntry(ctx, d, f.PlayerID))
	}

	return reply(ctx, d, connectionID, resp)
}

func resolveFriendEntry(ctx context.Context, d deps, playerID string) friendEntry {
	p, err := d.players.GetPlayer(ctx, playerID)
	if err != nil {
		return friendEntry{PlayerID: playerID, Username: playerID}
	}
	return friendEntry{PlayerID: p.PlayerID, Username: p.Username}
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
