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
