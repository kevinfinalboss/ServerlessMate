package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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

type mockPlayerStore struct{ mock.Mock }

func (m *mockPlayerStore) GetPlayer(ctx context.Context, playerID string) (*store.Player, error) {
	args := m.Called(ctx, playerID)
	p, _ := args.Get(0).(*store.Player)
	return p, args.Error(1)
}

func (m *mockPlayerStore) GetOrCreatePlayer(ctx context.Context, playerID string, now int64) (*store.Player, error) {
	args := m.Called(ctx, playerID, now)
	p, _ := args.Get(0).(*store.Player)
	return p, args.Error(1)
}

func (m *mockPlayerStore) RecordGameResult(ctx context.Context, playerID string, newRating int, outcome store.GameOutcome) error {
	args := m.Called(ctx, playerID, newRating, outcome)
	return args.Error(0)
}

func (m *mockPlayerStore) ListTopByRating(ctx context.Context, limit int32) ([]*store.Player, error) {
	args := m.Called(ctx, limit)
	p, _ := args.Get(0).([]*store.Player)
	return p, args.Error(1)
}

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

type mockBroadcaster struct{ mock.Mock }

func (m *mockBroadcaster) Send(ctx context.Context, connectionID string, payload []byte) error {
	args := m.Called(ctx, connectionID, payload)
	return args.Error(0)
}

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestHandle_AuthenticatedPlayer_JoinsWithRatingBand(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1", IsGuest: false}, nil)
	players.On("GetOrCreatePlayer", mock.Anything, "player-1", mock.Anything).Return(&store.Player{Rating: 1350}, nil)
	queue.On("Join", mock.Anything, mock.MatchedBy(func(e *store.QueueEntry) bool {
		return e.MatchmakingKey == "5+0#1200" && e.PlayerID == "player-1" && e.ConnectionID == "conn-1"
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.UnixMilli(1_700_000_000_000))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"timeControl":"5+0"}`))

	require.NoError(t, err)
}

func TestHandle_Guest_UsesDefaultBand(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "guest-1", IsGuest: true}, nil)
	queue.On("Join", mock.Anything, mock.MatchedBy(func(e *store.QueueEntry) bool {
		return e.MatchmakingKey == "5+0#1200" && e.IsGuest
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.UnixMilli(1_700_000_000_000))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"timeControl":"5+0"}`))

	require.NoError(t, err)
	players.AssertNotCalled(t, "GetOrCreatePlayer")
}

func TestHandle_UnsupportedTimeControl(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "unsupported time control")
	})).Return(nil)

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"timeControl":"1+0"}`))

	require.NoError(t, err)
	queue.AssertNotCalled(t, "Join")
}

func TestHandle_InvalidBody(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`not json`))

	require.NoError(t, err)
}

func TestHandle_GetConnectionError(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(nil, errors.New("network error"))

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"timeControl":"5+0"}`))

	require.Error(t, err)
}

func TestHandle_GetPlayerError(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	players.On("GetOrCreatePlayer", mock.Anything, "player-1", mock.Anything).Return(nil, errors.New("network error"))

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"timeControl":"5+0"}`))

	require.Error(t, err)
}

func TestHandle_JoinError(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1", IsGuest: true}, nil)
	queue.On("Join", mock.Anything, mock.Anything).Return(errors.New("network error"))

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"timeControl":"5+0"}`))

	require.Error(t, err)
}

func TestHandle_Leave_Success(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	queue.On("Leave", mock.Anything, "5+0#1200", "player-1").Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "queueLeft")
	})).Return(nil)

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"leaveQueue","matchmakingKey":"5+0#1200"}`))

	require.NoError(t, err)
	queue.AssertExpectations(t)
}

func TestHandle_Leave_MissingMatchmakingKey(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "missing matchmakingKey")
	})).Return(nil)

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"leaveQueue"}`))

	require.NoError(t, err)
	queue.AssertNotCalled(t, "Leave")
}

func TestHandle_Leave_QueueError(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	queue := new(mockQueueStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	queue.On("Leave", mock.Anything, "5+0#1200", "player-1").Return(errors.New("network error"))

	d := deps{connections: conns, players: players, queue: queue, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"leaveQueue","matchmakingKey":"5+0#1200"}`))

	require.Error(t, err)
}

func TestRatingBand(t *testing.T) {
	cases := []struct {
		rating int
		want   string
	}{
		{1000, "1000"},
		{1199, "1000"},
		{1200, "1200"},
		{1350, "1200"},
		{2800, "2800"},
	}
	for _, tc := range cases {
		if got := ratingBand(tc.rating); got != tc.want {
			t.Errorf("ratingBand(%d) = %s, want %s", tc.rating, got, tc.want)
		}
	}
}
