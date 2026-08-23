package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
)

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

type mockHistoryStore struct{ mock.Mock }

func (m *mockHistoryStore) RecordGameEnd(ctx context.Context, g *store.Game) error {
	args := m.Called(ctx, g)
	return args.Error(0)
}

func (m *mockHistoryStore) ListHistory(ctx context.Context, playerID string, limit int32) ([]*store.HistoryEntry, error) {
	args := m.Called(ctx, playerID, limit)
	e, _ := args.Get(0).([]*store.HistoryEntry)
	return e, args.Error(1)
}

type mockBroadcaster struct{ mock.Mock }

func (m *mockBroadcaster) Send(ctx context.Context, connectionID string, payload []byte) error {
	args := m.Called(ctx, connectionID, payload)
	return args.Error(0)
}

func TestHandle_List_DefaultLimit(t *testing.T) {
	conns := new(mockConnectionStore)
	games := new(mockGameStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	history.On("ListHistory", mock.Anything, "player-1", int32(defaultLimit)).
		Return([]*store.HistoryEntry{{GameID: "game-1", Result: store.ResultWin}}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		var body map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(payload, &body))
		var entries []*store.HistoryEntry
		require.NoError(t, json.Unmarshal(body["entries"], &entries))
		return len(entries) == 1 && entries[0].GameID == "game-1"
	})).Return(nil)

	d := deps{connections: conns, games: games, history: history, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{}`))

	require.NoError(t, err)
}

func TestHandle_List_LimitCapped(t *testing.T) {
	conns := new(mockConnectionStore)
	games := new(mockGameStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	history.On("ListHistory", mock.Anything, "player-1", int32(maxLimit)).Return([]*store.HistoryEntry{}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{connections: conns, games: games, history: history, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"limit":9999}`))

	require.NoError(t, err)
}

func TestHandle_Replay_Success(t *testing.T) {
	conns := new(mockConnectionStore)
	games := new(mockGameStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").
		Return(&store.Game{GameID: "game-1", PGN: "1. e4 e5", Players: store.Players{White: "player-1", Black: "player-2"}}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "1. e4 e5")
	})).Return(nil)

	d := deps{connections: conns, games: games, history: history, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"gameId":"game-1"}`))

	require.NoError(t, err)
	history.AssertNotCalled(t, "ListHistory")
}

func TestHandle_Replay_NotYourGame(t *testing.T) {
	conns := new(mockConnectionStore)
	games := new(mockGameStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-3"}, nil)
	games.On("GetGame", mock.Anything, "game-1").
		Return(&store.Game{GameID: "game-1", Players: store.Players{White: "player-1", Black: "player-2"}}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "not your game")
	})).Return(nil)

	d := deps{connections: conns, games: games, history: history, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"gameId":"game-1"}`))

	require.NoError(t, err)
}

func TestHandle_Replay_GameNotFound(t *testing.T) {
	conns := new(mockConnectionStore)
	games := new(mockGameStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(nil, store.ErrGameNotFound)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "game not found")
	})).Return(nil)

	d := deps{connections: conns, games: games, history: history, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"gameId":"game-1"}`))

	require.NoError(t, err)
}

func TestHandle_InvalidBody(t *testing.T) {
	conns := new(mockConnectionStore)
	games := new(mockGameStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{connections: conns, games: games, history: history, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`not json`))

	require.NoError(t, err)
}

func TestHandle_GetConnectionError(t *testing.T) {
	conns := new(mockConnectionStore)
	games := new(mockGameStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(nil, errors.New("network error"))

	d := deps{connections: conns, games: games, history: history, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{}`))

	require.Error(t, err)
}

func TestHandle_ListHistoryError(t *testing.T) {
	conns := new(mockConnectionStore)
	games := new(mockGameStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	history.On("ListHistory", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("network error"))

	d := deps{connections: conns, games: games, history: history, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{}`))

	require.Error(t, err)
}

func TestHandle_GetGameError(t *testing.T) {
	conns := new(mockConnectionStore)
	games := new(mockGameStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(nil, errors.New("network error"))

	d := deps{connections: conns, games: games, history: history, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"gameId":"game-1"}`))

	require.Error(t, err)
}
