package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kevinfinalboss/ServerlessMate/internal/auth"
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

type mockValidator struct{ mock.Mock }

func (m *mockValidator) ValidatePlayerID(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

type mockBroadcaster struct{ mock.Mock }

func (m *mockBroadcaster) Send(ctx context.Context, connectionID string, payload []byte) error {
	args := m.Called(ctx, connectionID, payload)
	return args.Error(0)
}

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func baseDeps(games *mockGameStore, conns *mockConnectionStore, validator *mockValidator, bc *mockBroadcaster) deps {
	return deps{
		games:       games,
		connections: conns,
		validator:   validator,
		broadcaster: bc,
		newGuestID:  func() string { return "guest-1" },
		now:         fixedNow(time.UnixMilli(1_700_000_000_000)),
		graceMs:     60_000,
	}
}

func TestHandle_Guest_NoToken(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.PlayerID == "guest-1" && c.IsGuest && c.GameID == ""
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), baseDeps(games, conns, validator, bc), "conn-1", "", "")

	require.NoError(t, err)
	conns.AssertExpectations(t)
	validator.AssertNotCalled(t, "ValidatePlayerID")
}

func TestHandle_ValidToken(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	validator.On("ValidatePlayerID", mock.Anything, "good-token").Return("player-1", nil)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.PlayerID == "player-1" && !c.IsGuest
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return true
	})).Return(nil)

	err := handle(context.Background(), baseDeps(games, conns, validator, bc), "conn-1", "good-token", "")

	require.NoError(t, err)
	conns.AssertExpectations(t)
}

func TestHandle_InvalidToken(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	validator.On("ValidatePlayerID", mock.Anything, "bad-token").Return("", auth.ErrInvalidToken)

	err := handle(context.Background(), baseDeps(games, conns, validator, bc), "conn-1", "bad-token", "")

	assert.ErrorIs(t, err, auth.ErrInvalidToken)
	conns.AssertNotCalled(t, "PutConnection")
}

func TestHandle_GameID_AsPlayer(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	validator.On("ValidatePlayerID", mock.Anything, "good-token").Return("player-1", nil)
	g := &store.Game{GameID: "game-1", FEN: "some-fen", Players: store.Players{White: "player-1", Black: "player-2"}}
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.GameID == "game-1" && c.Role == store.RolePlayer
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), `"type":"connected"`)
	})).Return(nil).Once()
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		var got store.Game
		return json.Unmarshal(payload, &got) == nil && got.GameID == "game-1" && got.FEN == "some-fen"
	})).Return(nil).Once()

	err := handle(context.Background(), baseDeps(games, conns, validator, bc), "conn-1", "good-token", "game-1")

	require.NoError(t, err)
	games.AssertNotCalled(t, "ClearDisconnect")
	bc.AssertExpectations(t)
}

func TestHandle_GameID_AsSpectator(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	g := &store.Game{GameID: "game-1", Players: store.Players{White: "player-1", Black: "player-2"}}
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.GameID == "game-1" && c.Role == store.RoleSpectator
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), baseDeps(games, conns, validator, bc), "conn-1", "", "game-1")

	require.NoError(t, err)
}

func TestHandle_GameID_RejoinWithinGrace(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	d := baseDeps(games, conns, validator, bc)
	validator.On("ValidatePlayerID", mock.Anything, "good-token").Return("player-1", nil)
	g := &store.Game{
		GameID:               "game-1",
		Players:              store.Players{White: "player-1", Black: "player-2"},
		DisconnectedPlayerID: "player-1",
		DisconnectedAt:       d.now().UnixMilli() - 30_000,
	}
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("ClearDisconnect", mock.Anything, "game-1", "player-1").Return(nil)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.Role == store.RolePlayer
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), d, "conn-1", "good-token", "game-1")

	require.NoError(t, err)
	games.AssertExpectations(t)
}

func TestHandle_GameID_RejoinGraceExpired(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	d := baseDeps(games, conns, validator, bc)
	validator.On("ValidatePlayerID", mock.Anything, "good-token").Return("player-1", nil)
	g := &store.Game{
		GameID:               "game-1",
		Players:              store.Players{White: "player-1", Black: "player-2"},
		DisconnectedPlayerID: "player-1",
		DisconnectedAt:       d.now().UnixMilli() - 90_000,
	}
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.Role == store.RolePlayer
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), d, "conn-1", "good-token", "game-1")

	require.NoError(t, err)
	games.AssertNotCalled(t, "ClearDisconnect")
}

func TestHandle_GameID_NotFound_ConnectsWithoutGame(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	validator.On("ValidatePlayerID", mock.Anything, "good-token").Return("player-1", nil)
	games.On("GetGame", mock.Anything, "stale-game").Return(nil, store.ErrGameNotFound)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.PlayerID == "player-1" && c.GameID == "" && c.Role == ""
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), baseDeps(games, conns, validator, bc), "conn-1", "good-token", "stale-game")

	require.NoError(t, err)
	conns.AssertExpectations(t)
}

func TestHandle_GameID_GetGameError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	games.On("GetGame", mock.Anything, "game-1").Return(nil, errors.New("network error"))

	err := handle(context.Background(), baseDeps(games, conns, validator, bc), "conn-1", "", "game-1")

	require.Error(t, err)
	conns.AssertNotCalled(t, "PutConnection")
}

func TestHandle_PutConnectionError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	conns.On("PutConnection", mock.Anything, mock.Anything).Return(errors.New("network error"))

	err := handle(context.Background(), baseDeps(games, conns, validator, bc), "conn-1", "", "")

	require.Error(t, err)
}

func TestHandle_SendWelcomeError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	validator := new(mockValidator)
	bc := new(mockBroadcaster)

	conns.On("PutConnection", mock.Anything, mock.Anything).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(errors.New("network error"))

	err := handle(context.Background(), baseDeps(games, conns, validator, bc), "conn-1", "", "")

	require.Error(t, err)
}
