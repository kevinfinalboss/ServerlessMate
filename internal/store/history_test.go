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

type mockDynamoDBHistoryAPI struct {
	mock.Mock
}

func (m *mockDynamoDBHistoryAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.PutItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBHistoryAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.QueryOutput)
	return out, args.Error(1)
}

func newTestHistoryStore(client *mockDynamoDBHistoryAPI) *DynamoHistoryStore {
	return &DynamoHistoryStore{client: client, tableName: "GameHistory"}
}

func TestNewDynamoHistoryStore(t *testing.T) {
	s := NewDynamoHistoryStore(&dynamodb.Client{}, "GameHistory")

	assert.Equal(t, "GameHistory", s.tableName)
	assert.NotNil(t, s.client)
}

func TestRecordGameEnd_HumanVsHuman_WritesBothSides(t *testing.T) {
	client := new(mockDynamoDBHistoryAPI)
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		var e HistoryEntry
		require.NoError(t, attributevalue.UnmarshalMap(in.Item, &e))
		return e.PlayerID == "player-1" && e.OpponentID == "player-2" && e.Result == ResultWin
	})).Return(&dynamodb.PutItemOutput{}, nil).Once()
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		var e HistoryEntry
		require.NoError(t, attributevalue.UnmarshalMap(in.Item, &e))
		return e.PlayerID == "player-2" && e.OpponentID == "player-1" && e.Result == ResultLoss
	})).Return(&dynamodb.PutItemOutput{}, nil).Once()

	g := &Game{
		GameID:  "game-1",
		Players: Players{White: "player-1", Black: "player-2"},
		Winner:  "player-1",
		EndedAt: 1_700_000_000_000,
	}

	err := newTestHistoryStore(client).RecordGameEnd(context.Background(), g)

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRecordGameEnd_Draw(t *testing.T) {
	client := new(mockDynamoDBHistoryAPI)
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		var e HistoryEntry
		require.NoError(t, attributevalue.UnmarshalMap(in.Item, &e))
		return e.Result == ResultDraw
	})).Return(&dynamodb.PutItemOutput{}, nil).Twice()

	g := &Game{GameID: "game-1", Players: Players{White: "player-1", Black: "player-2"}, Winner: ""}

	err := newTestHistoryStore(client).RecordGameEnd(context.Background(), g)

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRecordGameEnd_VsAI_WritesOnlyHuman(t *testing.T) {
	client := new(mockDynamoDBHistoryAPI)
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		var e HistoryEntry
		require.NoError(t, attributevalue.UnmarshalMap(in.Item, &e))
		return e.PlayerID == "player-1" && e.VsAI
	})).Return(&dynamodb.PutItemOutput{}, nil).Once()

	g := &Game{GameID: "game-1", Players: Players{White: "player-1", Black: "AI"}, Winner: "player-1", VsAI: true}

	err := newTestHistoryStore(client).RecordGameEnd(context.Background(), g)

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestRecordGameEnd_FirstPutError(t *testing.T) {
	client := new(mockDynamoDBHistoryAPI)
	client.On("PutItem", mock.Anything, mock.Anything).Return(&dynamodb.PutItemOutput{}, errors.New("network error"))

	g := &Game{GameID: "game-1", Players: Players{White: "player-1", Black: "player-2"}, Winner: "player-1"}

	err := newTestHistoryStore(client).RecordGameEnd(context.Background(), g)

	require.Error(t, err)
}

func TestListHistory_Success(t *testing.T) {
	entry, err := attributevalue.MarshalMap(&HistoryEntry{PlayerID: "player-1", GameID: "game-1", Result: ResultWin})
	require.NoError(t, err)

	client := new(mockDynamoDBHistoryAPI)
	client.On("Query", mock.Anything, mock.MatchedBy(func(in *dynamodb.QueryInput) bool {
		return *in.TableName == "GameHistory" && !*in.ScanIndexForward && *in.Limit == int32(10)
	})).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{entry}}, nil)

	got, err := newTestHistoryStore(client).ListHistory(context.Background(), "player-1", 10)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "game-1", got[0].GameID)
}

func TestListHistory_Error(t *testing.T) {
	client := new(mockDynamoDBHistoryAPI)
	client.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{}, errors.New("network error"))

	_, err := newTestHistoryStore(client).ListHistory(context.Background(), "player-1", 10)

	require.Error(t, err)
}
