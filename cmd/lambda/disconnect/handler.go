package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
)

type deps struct {
	games       store.GameStore
	connections store.ConnectionStore
	now         func() time.Time
}

func handle(ctx context.Context, d deps, connectionID string) error {
	conn, err := d.connections.GetConnection(ctx, connectionID)
	if err != nil {
		if errors.Is(err, store.ErrConnectionNotFound) {
			return nil
		}
		return fmt.Errorf("disconnect: load connection: %w", err)
	}

	if conn.Role == store.RolePlayer && conn.GameID != "" {
		if err := d.games.MarkDisconnected(ctx, conn.GameID, conn.PlayerID, d.now().UnixMilli()); err != nil {
			return fmt.Errorf("disconnect: mark disconnected: %w", err)
		}
	}

	if err := d.connections.DeleteConnection(ctx, connectionID); err != nil {
		return fmt.Errorf("disconnect: delete connection: %w", err)
	}
	return nil
}
