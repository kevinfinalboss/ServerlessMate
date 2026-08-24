package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

const startFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

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

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func baseGame() *store.Game {
	return &store.Game{
		GameID:      "game-1",
		FEN:         startFEN,
		Status:      "in_progress",
		Players:     store.Players{White: "player-1", Black: "player-2"},
		LastMoveAt:  1_700_000_000_000,
		WhiteTimeMs: 300000,
		BlackTimeMs: 300000,
	}
}

func mockPlayers(games *mockPlayerStore) {
	games.On("GetOrCreatePlayer", mock.Anything, "player-1", mock.Anything).Return(&store.Player{PlayerID: "player-1", Rating: 1200}, nil)
	games.On("GetOrCreatePlayer", mock.Anything, "player-2", mock.Anything).Return(&store.Player{PlayerID: "player-2", Rating: 1200}, nil)
}

func playerConn(playerID string) *store.Connection {
	return &store.Connection{ConnectionID: "conn-1", GameID: "game-1", PlayerID: playerID, Role: store.RolePlayer}
}

func TestHandle_ClientErrors(t *testing.T) {
	cases := []struct {
		name        string
		conn        *store.Connection
		game        *store.Game
		body        string
		wantMessage string
	}{
		{
			name:        "spectator cannot act",
			conn:        &store.Connection{ConnectionID: "conn-1", GameID: "game-1", PlayerID: "player-1", Role: store.RoleSpectator},
			body:        `{"action":"move","move":"e2e4"}`,
			wantMessage: "spectators cannot act",
		},
		{
			name:        "invalid body",
			conn:        playerConn("player-1"),
			game:        baseGame(),
			body:        `not json`,
			wantMessage: "invalid request body",
		},
		{
			name: "game already over",
			conn: playerConn("player-1"),
			game: func() *store.Game {
				g := baseGame()
				g.Status = "checkmate"
				return g
			}(),
			body:        `{"action":"move","move":"e2e4"}`,
			wantMessage: "game is already over",
		},
		{
			name:        "unknown action",
			conn:        playerConn("player-1"),
			game:        baseGame(),
			body:        `{"action":"castle"}`,
			wantMessage: "unknown action",
		},
		{
			name:        "not your turn",
			conn:        playerConn("player-2"),
			game:        baseGame(),
			body:        `{"action":"move","move":"e7e5"}`,
			wantMessage: "not your turn",
		},
		{
			name:        "illegal move",
			conn:        playerConn("player-1"),
			game:        baseGame(),
			body:        `{"action":"move","move":"e2e5"}`,
			wantMessage: "illegal move",
		},
		{
			name:        "no draw offer to accept",
			conn:        playerConn("player-1"),
			game:        baseGame(),
			body:        `{"action":"acceptDraw"}`,
			wantMessage: "no draw offer to accept",
		},
		{
			name: "cannot accept own draw offer",
			conn: playerConn("player-1"),
			game: func() *store.Game {
				g := baseGame()
				g.DrawOfferedBy = "player-1"
				return g
			}(),
			body:        `{"action":"acceptDraw"}`,
			wantMessage: "cannot accept your own draw offer",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			games := new(mockGameStore)
			conns := new(mockConnectionStore)
			bc := new(mockBroadcaster)

			conns.On("GetConnection", mock.Anything, "conn-1").Return(tc.conn, nil)
			if tc.conn.Role == store.RolePlayer {
				games.On("GetGame", mock.Anything, "game-1").Return(tc.game, nil)
			}
			bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
				return strings.Contains(string(payload), tc.wantMessage)
			})).Return(nil)

			d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(1_700_000_000_000))}

			err := handle(context.Background(), d, "conn-1", []byte(tc.body))

			require.NoError(t, err)
			bc.AssertExpectations(t)
		})
	}
}

func TestHandle_Move_Success(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()
	now := time.UnixMilli(g.LastMoveAt + 5000)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return strings.Contains(updated.FEN, "4P3") &&
			updated.TurnOf == "player-2" &&
			updated.Status == "in_progress" &&
			updated.WhiteTimeMs == 295000 &&
			updated.LastMoveAt == now.UnixMilli()
	}), startFEN).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").
		Return([]*store.Connection{{ConnectionID: "conn-1"}, {ConnectionID: "conn-2"}}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)
	bc.On("Send", mock.Anything, "conn-2", mock.Anything).Return(nil)

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(now)}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move","move":"e2e4"}`))

	require.NoError(t, err)
	games.AssertExpectations(t)
	conns.AssertExpectations(t)
	bc.AssertExpectations(t)
}

func TestHandle_Timeout(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	g := baseGame()
	g.WhiteTimeMs = 1000
	now := time.UnixMilli(g.LastMoveAt + 5000)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.Status == "timeout" &&
			updated.Winner == "player-2" &&
			updated.WhiteTimeMs == 0 &&
			updated.EndedAt == now.UnixMilli()
	}), startFEN).Return(nil)
	mockPlayers(players)
	players.On("RecordGameResult", mock.Anything, "player-1", 1184, store.OutcomeLoss).Return(nil)
	players.On("RecordGameResult", mock.Anything, "player-2", 1216, store.OutcomeWin).Return(nil)
	history.On("RecordGameEnd", mock.Anything, mock.Anything).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").Return([]*store.Connection{}, nil)

	d := deps{games: games, connections: conns, players: players, history: history, broadcaster: bc, now: fixedNow(now)}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move","move":"e2e4"}`))

	require.NoError(t, err)
	games.AssertExpectations(t)
	players.AssertExpectations(t)
}

func TestHandle_Resign(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-2"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.Status == "resigned" && updated.Winner == "player-1"
	}), startFEN).Return(nil)
	mockPlayers(players)
	players.On("RecordGameResult", mock.Anything, "player-1", 1216, store.OutcomeWin).Return(nil)
	players.On("RecordGameResult", mock.Anything, "player-2", 1184, store.OutcomeLoss).Return(nil)
	history.On("RecordGameEnd", mock.Anything, mock.Anything).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").Return([]*store.Connection{}, nil)

	d := deps{games: games, connections: conns, players: players, history: history, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.NoError(t, err)
	games.AssertExpectations(t)
	players.AssertExpectations(t)
}

func TestHandle_Resign_VsAI_SkipsRating(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	g := baseGame()
	g.VsAI = true

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-2"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, startFEN).Return(nil)
	history.On("RecordGameEnd", mock.Anything, mock.Anything).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").Return([]*store.Connection{}, nil)

	d := deps{games: games, connections: conns, players: players, history: history, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.NoError(t, err)
	players.AssertNotCalled(t, "GetOrCreatePlayer")
	players.AssertNotCalled(t, "RecordGameResult")
}

func TestHandle_Abandonment_GraceExpired(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	g := baseGame()
	g.DisconnectedPlayerID = "player-1"
	g.DisconnectedAt = g.LastMoveAt
	now := time.UnixMilli(g.LastMoveAt + 90_000)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-2"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.Status == "abandoned" &&
			updated.Winner == "player-2" &&
			updated.DisconnectedPlayerID == "" &&
			updated.DisconnectedAt == 0 &&
			updated.EndedAt == now.UnixMilli()
	}), startFEN).Return(nil)
	mockPlayers(players)
	players.On("RecordGameResult", mock.Anything, "player-2", 1208, store.OutcomeWin).Return(nil)
	players.On("RecordGameResult", mock.Anything, "player-1", 1184, store.OutcomeLoss).Return(nil)
	history.On("RecordGameEnd", mock.Anything, mock.Anything).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").Return([]*store.Connection{}, nil)

	d := deps{games: games, connections: conns, players: players, history: history, broadcaster: bc, now: fixedNow(now), graceMs: 60_000}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move","move":"e7e5"}`))

	require.NoError(t, err)
	games.AssertExpectations(t)
	players.AssertExpectations(t)
}

func TestHandle_Abandonment_StillWithinGrace(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	g := baseGame()
	g.DisconnectedPlayerID = "player-1"
	g.DisconnectedAt = g.LastMoveAt
	now := time.UnixMilli(g.LastMoveAt + 30_000)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-2"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.Status == "in_progress" &&
			updated.DrawOfferedBy == "player-2" &&
			updated.DisconnectedPlayerID == "player-1"
	}), startFEN).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").Return([]*store.Connection{}, nil)

	d := deps{games: games, connections: conns, players: players, broadcaster: bc, now: fixedNow(now), graceMs: 60_000}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"offerDraw"}`))

	require.NoError(t, err)
	games.AssertExpectations(t)
	players.AssertNotCalled(t, "GetOrCreatePlayer")
}

func TestHandle_Chat_Success(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").
		Return([]*store.Connection{{ConnectionID: "conn-1"}, {ConnectionID: "conn-2"}}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "gg") && strings.Contains(string(payload), "player-1")
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-2", mock.Anything).Return(nil)

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"chat","message":"gg"}`))

	require.NoError(t, err)
	games.AssertNotCalled(t, "UpdateGame")
}

func TestHandle_Chat_EmptyMessage(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "empty message")
	})).Return(nil)

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"chat","message":"   "}`))

	require.NoError(t, err)
	conns.AssertNotCalled(t, "ListConnectionsByGame")
}

func TestHandle_Chat_ListConnectionsError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").Return(nil, errors.New("network error"))

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"chat","message":"gg"}`))

	require.Error(t, err)
}

func TestHandle_OfferDraw(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.DrawOfferedBy == "player-1" && updated.Status == "in_progress"
	}), startFEN).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").Return([]*store.Connection{}, nil)

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"offerDraw"}`))

	require.NoError(t, err)
	games.AssertExpectations(t)
}

func TestHandle_AcceptDraw_Success(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	g := baseGame()
	g.DrawOfferedBy = "player-2"

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.Status == "draw_agreement" && updated.DrawOfferedBy == "" && updated.Winner == ""
	}), startFEN).Return(nil)
	mockPlayers(players)
	players.On("RecordGameResult", mock.Anything, "player-1", 1200, store.OutcomeDraw).Return(nil)
	players.On("RecordGameResult", mock.Anything, "player-2", 1200, store.OutcomeDraw).Return(nil)
	history.On("RecordGameEnd", mock.Anything, mock.Anything).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").Return([]*store.Connection{}, nil)

	d := deps{games: games, connections: conns, players: players, history: history, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"acceptDraw"}`))

	require.NoError(t, err)
	games.AssertExpectations(t)
	players.AssertExpectations(t)
}

func TestHandle_UpdateRatings_GetPlayerError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-2"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, startFEN).Return(nil)
	players.On("GetOrCreatePlayer", mock.Anything, "player-1", mock.Anything).Return(nil, errors.New("network error"))

	d := deps{games: games, connections: conns, players: players, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.Error(t, err)
}

func TestHandle_UpdateRatings_RecordGameResultError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-2"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, startFEN).Return(nil)
	mockPlayers(players)
	players.On("RecordGameResult", mock.Anything, "player-1", 1216, store.OutcomeWin).Return(errors.New("network error"))

	d := deps{games: games, connections: conns, players: players, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.Error(t, err)
}

func TestHandle_RecordHistoryError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	players := new(mockPlayerStore)
	history := new(mockHistoryStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-2"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, startFEN).Return(nil)
	mockPlayers(players)
	players.On("RecordGameResult", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	history.On("RecordGameEnd", mock.Anything, mock.Anything).Return(errors.New("network error"))

	d := deps{games: games, connections: conns, players: players, history: history, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.Error(t, err)
}

func TestHandle_ConcurrentUpdate(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, startFEN).Return(store.ErrConcurrentUpdate)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "game changed, please retry")
	})).Return(nil)

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.NoError(t, err)
	bc.AssertExpectations(t)
}

func TestHandle_UpdateGame_InfraError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, startFEN).Return(errors.New("network error"))

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.Error(t, err)
	assert.NotErrorIs(t, err, store.ErrConcurrentUpdate)
}

func TestHandle_PlayerNotInGame(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-99"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.Error(t, err)
}

func TestHandle_GetConnectionError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(nil, errors.New("network error"))

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.Error(t, err)
}

func TestHandle_GetGameError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(nil, errors.New("network error"))

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.Error(t, err)
}

func TestHandle_GameNotFound_NotifiesInsteadOfFailingSilently(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(nil, store.ErrGameNotFound)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "game no longer exists")
	})).Return(nil)

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.NoError(t, err)
	bc.AssertExpectations(t)
}

func TestHandle_ListConnectionsError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, startFEN).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").Return(nil, errors.New("network error"))

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"offerDraw"}`))

	require.Error(t, err)
}

func TestHandle_BroadcastGoneCleanup(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	bc := new(mockBroadcaster)

	g := baseGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(playerConn("player-1"), nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, startFEN).Return(nil)
	conns.On("ListConnectionsByGame", mock.Anything, "game-1").
		Return([]*store.Connection{{ConnectionID: "conn-1"}, {ConnectionID: "conn-3"}}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)
	bc.On("Send", mock.Anything, "conn-3", mock.Anything).Return(ws.ErrConnectionGone)
	conns.On("DeleteConnection", mock.Anything, "conn-3").Return(nil)

	d := deps{games: games, connections: conns, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"offerDraw"}`))

	require.NoError(t, err)
	conns.AssertExpectations(t)
}
