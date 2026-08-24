package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

type mockQueueStore struct{ mock.Mock }

func (m *mockQueueStore) Join(ctx context.Context, e *store.QueueEntry) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func (m *mockQueueStore) FindWaiting(ctx context.Context, matchmakingKey, excludePlayerID string) (*store.QueueEntry, error) {
	args := m.Called(ctx, matchmakingKey, excludePlayerID)
	e, _ := args.Get(0).(*store.QueueEntry)
	return e, args.Error(1)
}

func (m *mockQueueStore) Leave(ctx context.Context, matchmakingKey, playerID string) error {
	args := m.Called(ctx, matchmakingKey, playerID)
	return args.Error(0)
}

type mockMatchStore struct{ mock.Mock }

func (m *mockMatchStore) CreateMatch(ctx context.Context, game *store.Game, a, b *store.QueueEntry) error {
	args := m.Called(ctx, game, a, b)
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

type mockBroadcaster struct{ mock.Mock }

func (m *mockBroadcaster) Send(ctx context.Context, connectionID string, payload []byte) error {
	args := m.Called(ctx, connectionID, payload)
	return args.Error(0)
}

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func fixedGameID(id string) func() string {
	return func() string { return id }
}

func TestHandle_NoOneWaiting(t *testing.T) {
	queue := new(mockQueueStore)
	match := new(mockMatchStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	queue.On("FindWaiting", mock.Anything, "5+0#1200", "player-2").Return(nil, nil)

	d := deps{queue: queue, match: match, connections: conns, broadcaster: bc, now: fixedNow(time.Now()), newGameID: fixedGameID("game-1")}
	self := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-2", ConnectionID: "conn-2"}

	err := handle(context.Background(), d, self)

	require.NoError(t, err)
	match.AssertNotCalled(t, "CreateMatch")
}

func TestHandle_MatchFound_Success(t *testing.T) {
	queue := new(mockQueueStore)
	match := new(mockMatchStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	now := time.UnixMilli(1_700_000_000_000)
	waiting := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-1", ConnectionID: "conn-1", SortKey: "a"}
	self := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-2", ConnectionID: "conn-2", SortKey: "b"}

	queue.On("FindWaiting", mock.Anything, "5+0#1200", "player-2").Return(waiting, nil)
	match.On("CreateMatch", mock.Anything, mock.MatchedBy(func(g *store.Game) bool {
		return g.GameID == "game-1" &&
			g.Players.White == "player-1" &&
			g.Players.Black == "player-2" &&
			g.TurnOf == "player-1" &&
			g.WhiteTimeMs == 300_000 &&
			g.BlackTimeMs == 300_000 &&
			g.Status == "in_progress"
	}), waiting, self).Return(nil)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.ConnectionID == "conn-1" && c.GameID == "game-1" && c.Role == store.RolePlayer
	})).Return(nil)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.ConnectionID == "conn-2" && c.GameID == "game-1" && c.Role == store.RolePlayer
	})).Return(nil)
	isFullGameState := mock.MatchedBy(func(payload []byte) bool {
		var g store.Game
		if err := json.Unmarshal(payload, &g); err != nil {
			return false
		}
		return g.GameID == "game-1" && g.FEN == startFEN && g.TurnOf == "player-1" &&
			g.Players.White == "player-1" && g.Players.Black == "player-2"
	})
	bc.On("Send", mock.Anything, "conn-1", isFullGameState).Return(nil)
	bc.On("Send", mock.Anything, "conn-2", isFullGameState).Return(nil)

	d := deps{queue: queue, match: match, connections: conns, broadcaster: bc, now: fixedNow(now), newGameID: fixedGameID("game-1")}

	err := handle(context.Background(), d, self)

	require.NoError(t, err)
	match.AssertExpectations(t)
	conns.AssertExpectations(t)
	bc.AssertExpectations(t)
}

func TestHandle_ClaimFailed_NoOp(t *testing.T) {
	queue := new(mockQueueStore)
	match := new(mockMatchStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	waiting := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-1", ConnectionID: "conn-1"}
	self := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-2", ConnectionID: "conn-2"}

	queue.On("FindWaiting", mock.Anything, "5+0#1200", "player-2").Return(waiting, nil)
	match.On("CreateMatch", mock.Anything, mock.Anything, waiting, self).Return(store.ErrMatchClaimFailed)

	d := deps{queue: queue, match: match, connections: conns, broadcaster: bc, now: fixedNow(time.Now()), newGameID: fixedGameID("game-1")}

	err := handle(context.Background(), d, self)

	require.NoError(t, err)
	conns.AssertNotCalled(t, "PutConnection")
}

func TestHandle_FindWaitingError(t *testing.T) {
	queue := new(mockQueueStore)
	match := new(mockMatchStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	queue.On("FindWaiting", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("network error"))

	d := deps{queue: queue, match: match, connections: conns, broadcaster: bc, now: fixedNow(time.Now()), newGameID: fixedGameID("game-1")}
	self := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-2"}

	err := handle(context.Background(), d, self)

	require.Error(t, err)
}

func TestHandle_CreateMatchOtherError(t *testing.T) {
	queue := new(mockQueueStore)
	match := new(mockMatchStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	waiting := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-1"}
	self := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-2"}

	queue.On("FindWaiting", mock.Anything, mock.Anything, mock.Anything).Return(waiting, nil)
	match.On("CreateMatch", mock.Anything, mock.Anything, waiting, self).Return(errors.New("network error"))

	d := deps{queue: queue, match: match, connections: conns, broadcaster: bc, now: fixedNow(time.Now()), newGameID: fixedGameID("game-1")}

	err := handle(context.Background(), d, self)

	require.Error(t, err)
}

func TestHandle_AssignConnectionError(t *testing.T) {
	queue := new(mockQueueStore)
	match := new(mockMatchStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	waiting := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-1", ConnectionID: "conn-1"}
	self := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-2", ConnectionID: "conn-2"}

	queue.On("FindWaiting", mock.Anything, mock.Anything, mock.Anything).Return(waiting, nil)
	match.On("CreateMatch", mock.Anything, mock.Anything, waiting, self).Return(nil)
	conns.On("PutConnection", mock.Anything, mock.Anything).Return(errors.New("network error"))

	d := deps{queue: queue, match: match, connections: conns, broadcaster: bc, now: fixedNow(time.Now()), newGameID: fixedGameID("game-1")}

	err := handle(context.Background(), d, self)

	require.Error(t, err)
}

func TestHandle_NotifyGoneIsIgnored(t *testing.T) {
	queue := new(mockQueueStore)
	match := new(mockMatchStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	waiting := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-1", ConnectionID: "conn-1"}
	self := &store.QueueEntry{MatchmakingKey: "5+0#1200", PlayerID: "player-2", ConnectionID: "conn-2"}

	queue.On("FindWaiting", mock.Anything, mock.Anything, mock.Anything).Return(waiting, nil)
	match.On("CreateMatch", mock.Anything, mock.Anything, waiting, self).Return(nil)
	conns.On("PutConnection", mock.Anything, mock.Anything).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(ws.ErrConnectionGone)
	bc.On("Send", mock.Anything, "conn-2", mock.Anything).Return(nil)

	d := deps{queue: queue, match: match, connections: conns, broadcaster: bc, now: fixedNow(time.Now()), newGameID: fixedGameID("game-1")}

	err := handle(context.Background(), d, self)

	require.NoError(t, err)
}

func TestParseTimeControlMs(t *testing.T) {
	cases := []struct {
		key  string
		want int64
	}{
		{"5+0#1200", 300_000},
		{"10+0#1400", 600_000},
		{"invalid", 0},
		{"x+0#1200", 0},
	}
	for _, tc := range cases {
		if got := parseTimeControlMs(tc.key); got != tc.want {
			t.Errorf("parseTimeControlMs(%q) = %d, want %d", tc.key, got, tc.want)
		}
	}
}

func TestAssignConnection(t *testing.T) {
	conns := new(mockConnectionStore)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.ConnectionID == "conn-1" && c.GameID == "game-1" && c.PlayerID == "player-1" && c.Role == store.RolePlayer
	})).Return(nil)

	d := deps{connections: conns}
	e := &store.QueueEntry{ConnectionID: "conn-1", PlayerID: "player-1"}

	err := assignConnection(context.Background(), d, e, "game-1")

	assert.NoError(t, err)
}
