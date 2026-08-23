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

type mockDynamoDBQueueAPI struct {
	mock.Mock
}

func (m *mockDynamoDBQueueAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.PutItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBQueueAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.QueryOutput)
	return out, args.Error(1)
}

func newTestQueueStore(client *mockDynamoDBQueueAPI) *DynamoQueueStore {
	return &DynamoQueueStore{client: client, tableName: "Queue"}
}

func TestNewQueueEntry(t *testing.T) {
	e := NewQueueEntry("5+0#1200", "player-1", "conn-1", false, 1_700_000_000_000)

	assert.Equal(t, "5+0#1200", e.MatchmakingKey)
	assert.Equal(t, "1700000000000#player-1", e.SortKey)
	assert.Equal(t, "player-1", e.PlayerID)
	assert.Equal(t, "conn-1", e.ConnectionID)
	assert.False(t, e.IsGuest)
}

func TestNewDynamoQueueStore(t *testing.T) {
	s := NewDynamoQueueStore(&dynamodb.Client{}, "Queue")

	assert.Equal(t, "Queue", s.tableName)
	assert.NotNil(t, s.client)
}

func TestJoin_Success(t *testing.T) {
	client := new(mockDynamoDBQueueAPI)
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		return *in.TableName == "Queue"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	e := NewQueueEntry("5+0#1200", "player-1", "conn-1", false, 1_700_000_000_000)
	err := newTestQueueStore(client).Join(context.Background(), e)

	require.NoError(t, err)
}

func TestJoin_Error(t *testing.T) {
	client := new(mockDynamoDBQueueAPI)
	client.On("PutItem", mock.Anything, mock.Anything).Return(&dynamodb.PutItemOutput{}, errors.New("network error"))

	e := NewQueueEntry("5+0#1200", "player-1", "conn-1", false, 1_700_000_000_000)
	err := newTestQueueStore(client).Join(context.Background(), e)

	require.Error(t, err)
}

func TestFindWaiting_FindsOldestExcludingSelf(t *testing.T) {
	self := NewQueueEntry("5+0#1200", "player-1", "conn-1", false, 1_700_000_000_000)
	selfItem, err := attributevalue.MarshalMap(self)
	require.NoError(t, err)
	other := NewQueueEntry("5+0#1200", "player-2", "conn-2", false, 1_700_000_001_000)
	otherItem, err := attributevalue.MarshalMap(other)
	require.NoError(t, err)

	client := new(mockDynamoDBQueueAPI)
	client.On("Query", mock.Anything, mock.MatchedBy(func(in *dynamodb.QueryInput) bool {
		return *in.TableName == "Queue" && *in.ScanIndexForward
	})).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{selfItem, otherItem}}, nil)

	got, err := newTestQueueStore(client).FindWaiting(context.Background(), "5+0#1200", "player-1")

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "player-2", got.PlayerID)
}

func TestFindWaiting_NoneAvailable(t *testing.T) {
	self := NewQueueEntry("5+0#1200", "player-1", "conn-1", false, 1_700_000_000_000)
	selfItem, err := attributevalue.MarshalMap(self)
	require.NoError(t, err)

	client := new(mockDynamoDBQueueAPI)
	client.On("Query", mock.Anything, mock.Anything).
		Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{selfItem}}, nil)

	got, err := newTestQueueStore(client).FindWaiting(context.Background(), "5+0#1200", "player-1")

	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestFindWaiting_Error(t *testing.T) {
	client := new(mockDynamoDBQueueAPI)
	client.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{}, errors.New("network error"))

	_, err := newTestQueueStore(client).FindWaiting(context.Background(), "5+0#1200", "player-1")

	require.Error(t, err)
}
