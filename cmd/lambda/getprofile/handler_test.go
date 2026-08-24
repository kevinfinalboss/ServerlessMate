package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
)

func fixedNow() func() time.Time {
	return func() time.Time { return time.UnixMilli(1_700_000_000_000) }
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

type mockFriendshipStore struct{ mock.Mock }

func (m *mockFriendshipStore) GetFriendship(ctx context.Context, playerID, friendID string) (*store.Friendship, error) {
	args := m.Called(ctx, playerID, friendID)
	f, _ := args.Get(0).(*store.Friendship)
	return f, args.Error(1)
}

func (m *mockFriendshipStore) ListFriendships(ctx context.Context, playerID string) ([]*store.Friendship, error) {
	args := m.Called(ctx, playerID)
	f, _ := args.Get(0).([]*store.Friendship)
	return f, args.Error(1)
}

func (m *mockFriendshipStore) SendRequest(ctx context.Context, playerID, friendID string, at int64) error {
	args := m.Called(ctx, playerID, friendID, at)
	return args.Error(0)
}

func (m *mockFriendshipStore) AcceptRequest(ctx context.Context, playerID, friendID string, at int64) error {
	args := m.Called(ctx, playerID, friendID, at)
	return args.Error(0)
}

func (m *mockFriendshipStore) Block(ctx context.Context, playerID, friendID string, at int64) error {
	args := m.Called(ctx, playerID, friendID, at)
	return args.Error(0)
}

type mockBroadcaster struct{ mock.Mock }

func (m *mockBroadcaster) Send(ctx context.Context, connectionID string, payload []byte) error {
	args := m.Called(ctx, connectionID, payload)
	return args.Error(0)
}

func TestHandle_OwnProfile_AlwaysFull(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	players.On("GetOrCreatePlayer", mock.Anything, "player-1", mock.Anything).
		Return(&store.Player{PlayerID: "player-1", Username: "kasparov", Visibility: "friends", Rating: 1200}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		var resp response
		require.NoError(t, json.Unmarshal(payload, &resp))
		return resp.Visible && resp.Rating == 1200
	})).Return(nil)

	d := deps{connections: conns, players: players, friendships: friendships, broadcaster: bc, now: fixedNow()}

	err := handle(context.Background(), d, "conn-1", []byte(`{}`))

	require.NoError(t, err)
	friendships.AssertNotCalled(t, "GetFriendship")
}

func TestHandle_OtherProfile_Public(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	players.On("GetPlayer", mock.Anything, "player-2").
		Return(&store.Player{PlayerID: "player-2", Username: "carlsen", Visibility: "public", Rating: 2800}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		var resp response
		require.NoError(t, json.Unmarshal(payload, &resp))
		return resp.Visible && resp.Rating == 2800
	})).Return(nil)

	d := deps{connections: conns, players: players, friendships: friendships, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"playerId":"player-2"}`))

	require.NoError(t, err)
}

func TestHandle_OtherProfile_FriendsOnly_NotFriends(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	players.On("GetPlayer", mock.Anything, "player-2").
		Return(&store.Player{PlayerID: "player-2", Username: "carlsen", Visibility: "friends", Rating: 2800}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-1", "player-2").Return(nil, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		var resp response
		require.NoError(t, json.Unmarshal(payload, &resp))
		return !resp.Visible && resp.Rating == 0 && resp.Username == "carlsen"
	})).Return(nil)

	d := deps{connections: conns, players: players, friendships: friendships, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"playerId":"player-2"}`))

	require.NoError(t, err)
}

func TestHandle_OtherProfile_FriendsOnly_Accepted(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	players.On("GetPlayer", mock.Anything, "player-2").
		Return(&store.Player{PlayerID: "player-2", Username: "carlsen", Visibility: "friends", Rating: 2800}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-1", "player-2").
		Return(&store.Friendship{Status: store.FriendshipAccepted}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		var resp response
		require.NoError(t, json.Unmarshal(payload, &resp))
		return resp.Visible && resp.Rating == 2800
	})).Return(nil)

	d := deps{connections: conns, players: players, friendships: friendships, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"playerId":"player-2"}`))

	require.NoError(t, err)
}

func TestHandle_PlayerNotFound(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	players.On("GetPlayer", mock.Anything, "missing").Return(nil, store.ErrPlayerNotFound)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return true
	})).Return(nil)

	d := deps{connections: conns, players: players, friendships: friendships, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"playerId":"missing"}`))

	require.NoError(t, err)
}

func TestHandle_InvalidBody(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{connections: conns, players: players, friendships: friendships, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`not json`))

	require.NoError(t, err)
	players.AssertNotCalled(t, "GetPlayer")
	players.AssertNotCalled(t, "GetOrCreatePlayer")
}

func TestHandle_GetConnectionError(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(nil, errors.New("network error"))

	d := deps{connections: conns, players: players, friendships: friendships, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{}`))

	require.Error(t, err)
}

func TestHandle_GetPlayerError(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	players.On("GetOrCreatePlayer", mock.Anything, "player-1", mock.Anything).Return(nil, errors.New("network error"))

	d := deps{connections: conns, players: players, friendships: friendships, broadcaster: bc, now: fixedNow()}

	err := handle(context.Background(), d, "conn-1", []byte(`{}`))

	require.Error(t, err)
}

func TestHandle_FriendshipLookupError(t *testing.T) {
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").
		Return(&store.Connection{ConnectionID: "conn-1", PlayerID: "player-1"}, nil)
	players.On("GetPlayer", mock.Anything, "player-2").
		Return(&store.Player{PlayerID: "player-2", Visibility: "friends"}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-1", "player-2").Return(nil, errors.New("network error"))

	d := deps{connections: conns, players: players, friendships: friendships, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"playerId":"player-2"}`))

	require.Error(t, err)
}
