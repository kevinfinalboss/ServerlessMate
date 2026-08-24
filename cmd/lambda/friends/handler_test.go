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

func (m *mockFriendshipStore) ListIncomingRequests(ctx context.Context, playerID string) ([]*store.Friendship, error) {
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

func (m *mockFriendshipStore) CancelRequest(ctx context.Context, playerA, playerB string) error {
	args := m.Called(ctx, playerA, playerB)
	return args.Error(0)
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

func (m *mockPlayerStore) UpdateProfile(ctx context.Context, playerID string, update store.ProfileUpdate) (*store.Player, error) {
	args := m.Called(ctx, playerID, update)
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

type mockBroadcaster struct{ mock.Mock }

func (m *mockBroadcaster) Send(ctx context.Context, connectionID string, payload []byte) error {
	args := m.Called(ctx, connectionID, payload)
	return args.Error(0)
}

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func baseDeps(conns *mockConnectionStore, friendships *mockFriendshipStore, bc *mockBroadcaster) deps {
	return deps{connections: conns, friendships: friendships, broadcaster: bc, now: fixedNow(time.UnixMilli(1_700_000_000_000))}
}

func TestHandle_SendRequest_Success(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-2", "player-1").Return(nil, nil)
	friendships.On("SendRequest", mock.Anything, "player-1", "player-2", int64(1_700_000_000_000)).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"sendRequest","friendId":"player-2"}`))

	require.NoError(t, err)
}

func TestHandle_SendRequest_BlockedByTarget(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-2", "player-1").
		Return(&store.Friendship{Status: store.FriendshipBlocked}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "cannot send request")
	})).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"sendRequest","friendId":"player-2"}`))

	require.NoError(t, err)
	friendships.AssertNotCalled(t, "SendRequest")
}

func TestHandle_SendRequest_Conflict(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-2", "player-1").Return(nil, nil)
	friendships.On("SendRequest", mock.Anything, "player-1", "player-2", mock.Anything).Return(store.ErrFriendshipConflict)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "already friends")
	})).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"sendRequest","friendId":"player-2"}`))

	require.NoError(t, err)
}

func TestHandle_AcceptRequest_Success(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-2", "player-1").
		Return(&store.Friendship{Status: store.FriendshipPending}, nil)
	friendships.On("AcceptRequest", mock.Anything, "player-1", "player-2", int64(1_700_000_000_000)).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"acceptRequest","friendId":"player-2"}`))

	require.NoError(t, err)
}

func TestHandle_AcceptRequest_NoPending(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-2", "player-1").Return(nil, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "no pending request")
	})).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"acceptRequest","friendId":"player-2"}`))

	require.NoError(t, err)
	friendships.AssertNotCalled(t, "AcceptRequest")
}

func TestHandle_Block_Success(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("Block", mock.Anything, "player-1", "player-2", int64(1_700_000_000_000)).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"block","friendId":"player-2"}`))

	require.NoError(t, err)
}

func TestHandle_InvalidFriendID(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"block","friendId":"player-1"}`))

	require.NoError(t, err)
}

func TestHandle_UnknownAction(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"unfriend","friendId":"player-2"}`))

	require.NoError(t, err)
}

func TestHandle_InvalidBody(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`not json`))

	require.NoError(t, err)
}

func TestHandle_Guest_Rejected(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "guest-1", IsGuest: true}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "guests cannot use friends")
	})).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"listFriends"}`))

	require.NoError(t, err)
	friendships.AssertNotCalled(t, "ListFriendships")
}

func TestHandle_GetConnectionError(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(nil, errors.New("network error"))

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"block","friendId":"player-2"}`))

	require.Error(t, err)
}

func TestHandle_SendRequest_BlockCheckError(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-2", "player-1").Return(nil, errors.New("network error"))

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"sendRequest","friendId":"player-2"}`))

	require.Error(t, err)
}

func TestHandle_AcceptRequest_LoadError(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("GetFriendship", mock.Anything, "player-2", "player-1").Return(nil, errors.New("network error"))

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"acceptRequest","friendId":"player-2"}`))

	require.Error(t, err)
}

func TestHandle_Block_Error(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("Block", mock.Anything, "player-1", "player-2", mock.Anything).Return(errors.New("network error"))

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"block","friendId":"player-2"}`))

	require.Error(t, err)
}

func TestHandle_CancelRequest_Success(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("CancelRequest", mock.Anything, "player-1", "player-2").Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "friendRequestCancelled")
	})).Return(nil)

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"cancelRequest","friendId":"player-2"}`))

	require.NoError(t, err)
}

func TestHandle_CancelRequest_Error(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("CancelRequest", mock.Anything, "player-1", "player-2").Return(errors.New("network error"))

	err := handle(context.Background(), baseDeps(conns, friendships, bc), "conn-1", []byte(`{"action":"cancelRequest","friendId":"player-2"}`))

	require.Error(t, err)
}

func TestHandle_ListFriends_Success(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("ListFriendships", mock.Anything, "player-1").Return([]*store.Friendship{
		{PlayerID: "player-1", FriendID: "player-2", Status: store.FriendshipAccepted},
		{PlayerID: "player-1", FriendID: "player-3", Status: store.FriendshipPending},
	}, nil)
	friendships.On("ListIncomingRequests", mock.Anything, "player-1").Return([]*store.Friendship{
		{PlayerID: "player-4", FriendID: "player-1", Status: store.FriendshipPending},
	}, nil)
	players.On("GetPlayer", mock.Anything, "player-2").Return(&store.Player{PlayerID: "player-2", Username: "carlsen"}, nil)
	players.On("GetPlayer", mock.Anything, "player-3").Return(&store.Player{PlayerID: "player-3", Username: "nakamura"}, nil)
	players.On("GetPlayer", mock.Anything, "player-4").Return(nil, store.ErrPlayerNotFound)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		s := string(payload)
		return strings.Contains(s, `"type":"friends"`) &&
			strings.Contains(s, "carlsen") &&
			strings.Contains(s, "nakamura") &&
			strings.Contains(s, "player-4")
	})).Return(nil)

	d := deps{connections: conns, friendships: friendships, players: players, broadcaster: bc, now: fixedNow(time.UnixMilli(1_700_000_000_000))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"listFriends"}`))

	require.NoError(t, err)
}

func TestHandle_ListFriends_ListFriendshipsError(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("ListFriendships", mock.Anything, "player-1").Return(nil, errors.New("network error"))

	d := deps{connections: conns, friendships: friendships, players: players, broadcaster: bc, now: fixedNow(time.UnixMilli(1_700_000_000_000))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"listFriends"}`))

	require.Error(t, err)
}

func TestHandle_ListFriends_ListIncomingError(t *testing.T) {
	conns := new(mockConnectionStore)
	friendships := new(mockFriendshipStore)
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	friendships.On("ListFriendships", mock.Anything, "player-1").Return([]*store.Friendship{}, nil)
	friendships.On("ListIncomingRequests", mock.Anything, "player-1").Return(nil, errors.New("network error"))

	d := deps{connections: conns, friendships: friendships, players: players, broadcaster: bc, now: fixedNow(time.UnixMilli(1_700_000_000_000))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"listFriends"}`))

	require.Error(t, err)
}
