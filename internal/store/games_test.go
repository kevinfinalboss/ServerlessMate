package store

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDynamoDBAPI struct {
	mock.Mock
}

func (m *mockDynamoDBAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.GetItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.PutItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.UpdateItemOutput)
	return out, args.Error(1)
}

func newTestStore(client *mockDynamoDBAPI) *DynamoGameStore {
	return &DynamoGameStore{client: client, tableName: "Games"}
}

func TestNewDynamoGameStore(t *testing.T) {
	s := NewDynamoGameStore(&dynamodb.Client{}, "Games")

	assert.Equal(t, "Games", s.tableName)
	assert.NotNil(t, s.client)
}

func sampleGame() *Game {
	return &Game{
		GameID:      "game-1",
		FEN:         "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		PGN:         "",
		Players:     Players{White: "player-1", Black: "player-2"},
		TurnOf:      "player-1",
		Status:      "in_progress",
		WhiteTimeMs: 300000,
		BlackTimeMs: 300000,
		LastMoveAt:  1700000000000,
	}
}

func TestCreateGame_Success(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		return *in.TableName == "Games" &&
			*in.ConditionExpression == "attribute_not_exists(gameId)"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := newTestStore(client).CreateGame(context.Background(), sampleGame())

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestCreateGame_AlreadyExists(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("PutItem", mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, &types.ConditionalCheckFailedException{})

	err := newTestStore(client).CreateGame(context.Background(), sampleGame())

	assert.ErrorIs(t, err, ErrGameAlreadyExists)
}

func TestCreateGame_OtherError(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("PutItem", mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, errors.New("network error"))

	err := newTestStore(client).CreateGame(context.Background(), sampleGame())

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrGameAlreadyExists)
}

func TestGetGame_Success(t *testing.T) {
	want := sampleGame()
	item, err := attributevalue.MarshalMap(want)
	require.NoError(t, err)

	client := new(mockDynamoDBAPI)
	client.On("GetItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.GetItemInput) bool {
		return *in.TableName == "Games" && in.Key["gameId"] != nil
	})).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	got, err := newTestStore(client).GetGame(context.Background(), "game-1")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetGame_NotFound(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("GetItem", mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{Item: nil}, nil)

	_, err := newTestStore(client).GetGame(context.Background(), "missing")

	assert.ErrorIs(t, err, ErrGameNotFound)
}

func TestGetGame_ClientError(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("GetItem", mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, errors.New("network error"))

	_, err := newTestStore(client).GetGame(context.Background(), "game-1")

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrGameNotFound)
}

func TestUpdateGame_Success(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		expected, ok := in.ExpressionAttributeValues[":expectedFen"].(*types.AttributeValueMemberS)
		return *in.ConditionExpression == "fen = :expectedFen" &&
			ok && expected.Value == "start-fen"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := newTestStore(client).UpdateGame(context.Background(), sampleGame(), "start-fen")

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestUpdateGame_ConcurrentUpdate(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("PutItem", mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, &types.ConditionalCheckFailedException{})

	err := newTestStore(client).UpdateGame(context.Background(), sampleGame(), "stale-fen")

	assert.ErrorIs(t, err, ErrConcurrentUpdate)
}

func TestUpdateGame_OtherError(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("PutItem", mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, errors.New("network error"))

	err := newTestStore(client).UpdateGame(context.Background(), sampleGame(), "start-fen")

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrConcurrentUpdate)
}

func TestClearDisconnect_Success(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("UpdateItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.UpdateItemInput) bool {
		expected, ok := in.ExpressionAttributeValues[":playerId"].(*types.AttributeValueMemberS)
		return *in.TableName == "Games" &&
			*in.UpdateExpression == "REMOVE disconnectedPlayerId, disconnectedAt" &&
			ok && expected.Value == "player-1"
	})).Return(&dynamodb.UpdateItemOutput{}, nil)

	err := newTestStore(client).ClearDisconnect(context.Background(), "game-1", "player-1")

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestClearDisconnect_ConditionNotMet(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("UpdateItem", mock.Anything, mock.Anything).
		Return(&dynamodb.UpdateItemOutput{}, &types.ConditionalCheckFailedException{})

	err := newTestStore(client).ClearDisconnect(context.Background(), "game-1", "player-1")

	assert.NoError(t, err)
}

func TestClearDisconnect_OtherError(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("UpdateItem", mock.Anything, mock.Anything).
		Return(&dynamodb.UpdateItemOutput{}, errors.New("network error"))

	err := newTestStore(client).ClearDisconnect(context.Background(), "game-1", "player-1")

	require.Error(t, err)
}

func TestMarkDisconnected_Success(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("UpdateItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.UpdateItemInput) bool {
		playerID, ok1 := in.ExpressionAttributeValues[":playerId"].(*types.AttributeValueMemberS)
		at, ok2 := in.ExpressionAttributeValues[":at"].(*types.AttributeValueMemberN)
		return *in.TableName == "Games" &&
			in.ExpressionAttributeNames["#status"] == "status" &&
			ok1 && playerID.Value == "player-1" &&
			ok2 && at.Value == "1700000000000"
	})).Return(&dynamodb.UpdateItemOutput{}, nil)

	err := newTestStore(client).MarkDisconnected(context.Background(), "game-1", "player-1", 1_700_000_000_000)

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestMarkDisconnected_ConditionNotMet(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("UpdateItem", mock.Anything, mock.Anything).
		Return(&dynamodb.UpdateItemOutput{}, &types.ConditionalCheckFailedException{})

	err := newTestStore(client).MarkDisconnected(context.Background(), "game-1", "player-1", 1_700_000_000_000)

	assert.NoError(t, err)
}

func TestMarkDisconnected_OtherError(t *testing.T) {
	client := new(mockDynamoDBAPI)
	client.On("UpdateItem", mock.Anything, mock.Anything).
		Return(&dynamodb.UpdateItemOutput{}, errors.New("network error"))

	err := newTestStore(client).MarkDisconnected(context.Background(), "game-1", "player-1", 1_700_000_000_000)

	require.Error(t, err)
}
