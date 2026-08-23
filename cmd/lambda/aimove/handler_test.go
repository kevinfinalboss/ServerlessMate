package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kevinfinalboss/ServerlessMate/internal/ai"
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

type mockRateLimitStore struct{ mock.Mock }

func (m *mockRateLimitStore) IncrementAndCheck(ctx context.Context, playerID, date string, limit int, ttl int64) (bool, error) {
	args := m.Called(ctx, playerID, date, limit, ttl)
	return args.Bool(0), args.Error(1)
}

type mockCommentator struct{ mock.Mock }

func (m *mockCommentator) Comment(ctx context.Context, fen, uci string) (string, error) {
	args := m.Called(ctx, fen, uci)
	return args.String(0), args.Error(1)
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

const aiCapturesFEN = "q3k3/R7/8/8/8/8/8/4K3 b - - 0 1"

func baseAIGame() *store.Game {
	return &store.Game{
		GameID:      "game-1",
		FEN:         aiCapturesFEN,
		Status:      "in_progress",
		Players:     store.Players{White: "player-1", Black: ai.PlayerID},
		VsAI:        true,
		AILevel:     "easy",
		LastMoveAt:  1_700_000_000_000,
		WhiteTimeMs: defaultTimeMs,
		BlackTimeMs: defaultTimeMs,
	}
}

func TestHandle_Guest_Rejected(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "guest-1", IsGuest: true}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "guests cannot play")
	})).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"start","level":"easy"}`))

	require.NoError(t, err)
}

func TestHandle_Start_Success(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	games.On("CreateGame", mock.Anything, mock.MatchedBy(func(g *store.Game) bool {
		return g.GameID == "game-1" && g.VsAI && g.AILevel == "hard" &&
			g.Players.White == "player-1" && g.Players.Black == ai.PlayerID &&
			g.WhiteTimeMs == defaultTimeMs && g.Status == "in_progress"
	})).Return(nil)
	conns.On("PutConnection", mock.Anything, mock.MatchedBy(func(c *store.Connection) bool {
		return c.ConnectionID == "conn-1" && c.GameID == "game-1" && c.Role == store.RolePlayer
	})).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc,
		newGameID: func() string { return "game-1" }, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"start","level":"hard"}`))

	require.NoError(t, err)
}

func TestHandle_Start_InvalidLevel(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "invalid level")
	})).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"start","level":"medium"}`))

	require.NoError(t, err)
	games.AssertNotCalled(t, "CreateGame")
}

func TestHandle_Move_Success(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	g := baseAIGame()
	now := time.UnixMilli(g.LastMoveAt)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.FEN != aiCapturesFEN && updated.TurnOf == "player-1" && updated.Status == "in_progress"
	}), aiCapturesFEN).Return(nil)
	rate.On("IncrementAndCheck", mock.Anything, "player-1", mock.Anything, dailyBedrockLimit, mock.Anything).Return(true, nil)
	commentator.On("Comment", mock.Anything, mock.Anything, "a8a7").Return("Nice grab!", nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "Nice grab!")
	})).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(now)}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.NoError(t, err)
	games.AssertExpectations(t)
}

const aiEndsGameFEN = "4k3/8/8/8/1n6/8/P7/4K3 b - - 0 1"

func TestHandle_Move_GameEnds_RecordsHistory(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	history := new(mockHistoryStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	g := baseAIGame()
	g.FEN = aiEndsGameFEN
	now := time.UnixMilli(g.LastMoveAt)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.Status == "draw_insufficient_material" && updated.EndedAt == now.UnixMilli() && updated.TurnOf == ""
	}), aiEndsGameFEN).Return(nil)
	history.On("RecordGameEnd", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.Status == "draw_insufficient_material"
	})).Return(nil)
	rate.On("IncrementAndCheck", mock.Anything, "player-1", mock.Anything, dailyBedrockLimit, mock.Anything).Return(true, nil)
	commentator.On("Comment", mock.Anything, mock.Anything, "b4a2").Return("Nice trade!", nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, history: history, commentator: commentator, broadcaster: bc, now: fixedNow(now)}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.NoError(t, err)
	history.AssertExpectations(t)
}

func TestHandle_Move_RecordHistoryError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	history := new(mockHistoryStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	g := baseAIGame()
	g.FEN = aiEndsGameFEN
	now := time.UnixMilli(g.LastMoveAt)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, aiEndsGameFEN).Return(nil)
	history.On("RecordGameEnd", mock.Anything, mock.Anything).Return(errors.New("network error"))

	d := deps{games: games, connections: conns, rateLimits: rate, history: history, commentator: commentator, broadcaster: bc, now: fixedNow(now)}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.Error(t, err)
	rate.AssertNotCalled(t, "IncrementAndCheck")
}

func TestHandle_Move_RateLimited_NoComment(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	g := baseAIGame()
	now := time.UnixMilli(g.LastMoveAt)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, aiCapturesFEN).Return(nil)
	rate.On("IncrementAndCheck", mock.Anything, "player-1", mock.Anything, dailyBedrockLimit, mock.Anything).Return(false, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return !strings.Contains(string(payload), "comment")
	})).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(now)}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.NoError(t, err)
	commentator.AssertNotCalled(t, "Comment")
}

func TestHandle_Move_CommentatorError_MoveStillSucceeds(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	g := baseAIGame()
	now := time.UnixMilli(g.LastMoveAt)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, aiCapturesFEN).Return(nil)
	rate.On("IncrementAndCheck", mock.Anything, "player-1", mock.Anything, dailyBedrockLimit, mock.Anything).Return(true, nil)
	commentator.On("Comment", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("bedrock down"))
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(now)}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.NoError(t, err)
}

func TestHandle_Move_NotAITurn(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	g := baseAIGame()
	g.FEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "not AI's turn")
	})).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.NoError(t, err)
	games.AssertNotCalled(t, "UpdateGame")
}

func TestHandle_Move_Timeout(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	history := new(mockHistoryStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	g := baseAIGame()
	g.BlackTimeMs = 1000
	now := time.UnixMilli(g.LastMoveAt + 5000)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.MatchedBy(func(updated *store.Game) bool {
		return updated.Status == "timeout" && updated.Winner == "player-1" && updated.BlackTimeMs == 0
	}), aiCapturesFEN).Return(nil)
	history.On("RecordGameEnd", mock.Anything, mock.Anything).Return(nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, history: history, commentator: commentator, broadcaster: bc, now: fixedNow(now)}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.NoError(t, err)
	rate.AssertNotCalled(t, "IncrementAndCheck")
}

func TestHandle_Move_NoActiveGame(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: ""}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "no active game")
	})).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.NoError(t, err)
}

func TestHandle_Move_NotVsAIGame(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	g := baseAIGame()
	g.VsAI = false

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.MatchedBy(func(payload []byte) bool {
		return strings.Contains(string(payload), "no active AI game")
	})).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.NoError(t, err)
}

func TestHandle_UnknownAction(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"resign"}`))

	require.NoError(t, err)
}

func TestHandle_InvalidBody(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	bc.On("Send", mock.Anything, "conn-1", mock.Anything).Return(nil)

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`not json`))

	require.NoError(t, err)
}

func TestHandle_GetConnectionError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(nil, errors.New("network error"))

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.Error(t, err)
}

func TestHandle_CreateGameError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1"}, nil)
	games.On("CreateGame", mock.Anything, mock.Anything).Return(errors.New("network error"))

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc,
		newGameID: func() string { return "game-1" }, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"start","level":"easy"}`))

	require.Error(t, err)
}

func TestHandle_GetGameError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(nil, errors.New("network error"))

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.Now())}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.Error(t, err)
}

func TestHandle_UpdateGameError(t *testing.T) {
	games := new(mockGameStore)
	conns := new(mockConnectionStore)
	rate := new(mockRateLimitStore)
	commentator := new(mockCommentator)
	bc := new(mockBroadcaster)

	g := baseAIGame()

	conns.On("GetConnection", mock.Anything, "conn-1").Return(&store.Connection{PlayerID: "player-1", GameID: "game-1"}, nil)
	games.On("GetGame", mock.Anything, "game-1").Return(g, nil)
	games.On("UpdateGame", mock.Anything, mock.Anything, aiCapturesFEN).Return(errors.New("network error"))

	d := deps{games: games, connections: conns, rateLimits: rate, commentator: commentator, broadcaster: bc, now: fixedNow(time.UnixMilli(g.LastMoveAt))}

	err := handle(context.Background(), d, "conn-1", []byte(`{"action":"move"}`))

	require.Error(t, err)
}
