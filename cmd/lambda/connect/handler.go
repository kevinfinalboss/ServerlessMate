package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kevinfinalboss/ServerlessMate/internal/auth"
	"github.com/kevinfinalboss/ServerlessMate/internal/store"
)

type deps struct {
	games       store.GameStore
	connections store.ConnectionStore
	validator   auth.Validator
	newGuestID  func() string
	now         func() time.Time
	graceMs     int64
}

func handle(ctx context.Context, d deps, connectionID, token, gameID string) error {
	playerID, isGuest, err := resolveIdentity(ctx, d, token)
	if err != nil {
		return fmt.Errorf("connect: resolve identity: %w", err)
	}

	conn := &store.Connection{
		ConnectionID: connectionID,
		PlayerID:     playerID,
		IsGuest:      isGuest,
	}

	if gameID != "" {
		role, err := resolveRole(ctx, d, gameID, playerID)
		if err != nil {
			return fmt.Errorf("connect: resolve role: %w", err)
		}
		conn.GameID = gameID
		conn.Role = role
	}

	if err := d.connections.PutConnection(ctx, conn); err != nil {
		return fmt.Errorf("connect: put connection: %w", err)
	}
	return nil
}

func resolveIdentity(ctx context.Context, d deps, token string) (playerID string, isGuest bool, err error) {
	if token == "" {
		return d.newGuestID(), true, nil
	}

	sub, err := d.validator.ValidatePlayerID(ctx, token)
	if err != nil {
		return "", false, err
	}
	return sub, false, nil
}

func resolveRole(ctx context.Context, d deps, gameID, playerID string) (string, error) {
	g, err := d.games.GetGame(ctx, gameID)
	if err != nil {
		return "", err
	}

	if g.DisconnectedPlayerID == playerID && d.now().UnixMilli()-g.DisconnectedAt <= d.graceMs {
		if err := d.games.ClearDisconnect(ctx, gameID, playerID); err != nil {
			return "", err
		}
		return store.RolePlayer, nil
	}

	if playerID == g.Players.White || playerID == g.Players.Black {
		return store.RolePlayer, nil
	}
	return store.RoleSpectator, nil
}
