package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
)

type mockPlayerStore struct{ mock.Mock }

func (m *mockPlayerStore) GetPlayer(ctx context.Context, playerID string) (*store.Player, error) {
	args := m.Called(ctx, playerID)
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

func TestHandle_DefaultLimit(t *testing.T) {
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	players.On("ListTopByRating", mock.Anything, int32(defaultLimit)).
		Return([]*store.Player{{PlayerID: "player-1", Username: "carlsen", Rating: 2800}}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		var body map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(payload, &body))
		var entries []entry
		require.NoError(t, json.Unmarshal(body["entries"], &entries))
		return len(entries) == 1 && entries[0].PlayerID == "player-1"
	})).Return(nil)

	d := deps{players: players, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{}`))

	require.NoError(t, err)
}

func TestHandle_CustomLimit(t *testing.T) {
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	players.On("ListTopByRating", mock.Anything, int32(25)).Return([]*store.Player{}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{players: players, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"limit":25}`))

	require.NoError(t, err)
}

func TestHandle_LimitCappedAtMax(t *testing.T) {
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	players.On("ListTopByRating", mock.Anything, int32(maxLimit)).Return([]*store.Player{}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{players: players, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{"limit":9999}`))

	require.NoError(t, err)
}

func TestHandle_InvalidBody(t *testing.T) {
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{players: players, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`not json`))

	require.NoError(t, err)
	players.AssertNotCalled(t, "ListTopByRating")
}

func TestHandle_ListTopByRatingError(t *testing.T) {
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	players.On("ListTopByRating", mock.Anything, mock.Anything).Return(nil, errors.New("network error"))

	d := deps{players: players, broadcaster: bc}

	err := handle(context.Background(), d, "conn-1", []byte(`{}`))

	require.Error(t, err)
}
