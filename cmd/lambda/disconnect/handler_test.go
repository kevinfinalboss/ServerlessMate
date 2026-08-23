package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
)

type mockGameStore struct{ mock.Mock }

func (m *mockGameStore) CreateGame(ctx context.Context, g *store.Game) error {
	args := m.Called(ctx, g)
	return args.Error(0)
}

func (m *mockGameStore) GetGame(ctx context.Context, gameID string) (*store.Game, error) {
	args := m.Called(ctx, gameID)
	g, _ := args.Get(0).(*store.Game)
	return g, args.Error(1)
}

func (m *mockGameStore) UpdateGame(ctx context.Context, g *store.Game, expectedFEN string) error {
	args := m.Called(ctx, g, expectedFEN)
	return args.Error(0)
}

func (m *mockGameStore) ClearDisconnect(ctx context.Context, gameID, playerID string) error {
	args := m.Called(ctx, gameID, playerID)
	return args.Error(0)
}

func (m *mockGameStore) MarkDisconnected(ctx context.Context, gameID, playerID string, at int64) error {
	args := m.Called(ctx, gameID, playerID, at)
	return args.Error(0)
}

type mockConnectionStore struct{ mock.Mock }

func (m *mockConnectionStore) PutConnection(ctx context.Context, c *store.Connection) error {
	args := m.Called(ctx, c)
	return args.Error(0)
}

func (m *mockConnectionStore) GetConnection(ctx context.Context, connectionID string) (*store.Connection, error) {
	args := m.Called(ctx, connectionID)
	c, _ := args.Get(0).(*store.Connection)
	return c, args.Error(1)
}

func (m *mockConnectionStore) DeleteConnection(ctx context.Context, connectionID string) error {
	args := m.Called(ctx, connectionID)
	return args.Error(0)
}

func (m *mockConnectionStore) ListConnectionsByGame(ctx context.Context, gameID string) ([]*store.Connection, error) {
	args := m.Called(ctx, gameID)
	c, _ := args.Get(0).([]*store.Connection)
	return c, args.Error(1)
}

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestHandle_PlayerDisconnect_MarksGame(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	now := time.UnixMilli(1_700_000_000_000)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", GameID: "game-1", PlayerID: "player-1", Role: store.RolePlayer}, nil)
	games.On("MarkDisconnected", mock.Anything, "game-1", "player-1", now.UnixMilli()).Return(nil)
	conns.On("DeleteConnection", mock.Anything, "conn-1").Return(nil)

	d := deps{games: games, connections: conns, now: fixedNow(now)}

	err := handle(context.Background(), d, "conn-1")

	require.NoError(t, err)
	games.AssertExpectations(t)
	conns.AssertExpectations(t)
}

func TestHandle_SpectatorDisconnect_NoMark(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", GameID: "game-1", PlayerID: "player-1", Role: store.RoleSpectator}, nil)
	conns.On("DeleteConnection", mock.Anything, "conn-1").Return(nil)

	d := deps{games: games, connections: conns, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1")

	require.NoError(t, err)
	games.AssertNotCalled(t, "MarkDisconnected")
	conns.AssertExpectations(t)
}

func TestHandle_PlayerDisconnect_NoGameID(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", GameID: "", PlayerID: "player-1", Role: store.RolePlayer}, nil)
	conns.On("DeleteConnection", mock.Anything, "conn-1").Return(nil)

	d := deps{games: games, connections: conns, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1")

	require.NoError(t, err)
	games.AssertNotCalled(t, "MarkDisconnected")
}

func TestHandle_ConnectionNotFound(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(nil, store.ErrConnectionNotFound)

	d := deps{games: games, connections: conns, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1")

	require.NoError(t, err)
	conns.AssertNotCalled(t, "DeleteConnection")
}

func TestHandle_GetConnectionError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(nil, errors.New("network error"))

	d := deps{games: games, connections: conns, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1")

	require.Error(t, err)
}

func TestHandle_MarkDisconnectedError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", GameID: "game-1", PlayerID: "player-1", Role: store.RolePlayer}, nil)
	games.On("MarkDisconnected", mock.Anything, "game-1", "player-1", mock.Anything).Return(errors.New("network error"))

	d := deps{games: games, connections: conns, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1")

	require.Error(t, err)
	conns.AssertNotCalled(t, "DeleteConnection")
}

func TestHandle_DeleteConnectionError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", GameID: "", PlayerID: "player-1", Role: store.RolePlayer}, nil)
	conns.On("DeleteConnection", mock.Anything, "conn-1").Return(errors.New("network error"))

	d := deps{games: games, connections: conns, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1")

	require.Error(t, err)
}
