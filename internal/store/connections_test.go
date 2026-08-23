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

type mockDynamoDBQueryAPI struct {
	mock.Mock
}

func (m *mockDynamoDBQueryAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.GetItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBQueryAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.PutItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBQueryAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.UpdateItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBQueryAPI) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.DeleteItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBQueryAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.QueryOutput)
	return out, args.Error(1)
}

func newTestConnectionStore(client *mockDynamoDBQueryAPI) *DynamoConnectionStore {
	return &DynamoConnectionStore{client: client, tableName: "Connections"}
}

func sampleConnection() *Connection {
	return &Connection{
		ConnectionID: "conn-1",
		GameID:       "game-1",
		PlayerID:     "player-1",
		IsGuest:      false,
		Role:         RolePlayer,
	}
}

func TestNewDynamoConnectionStore(t *testing.T) {
	s := NewDynamoConnectionStore(&dynamodb.Client{}, "Connections")

	assert.Equal(t, "Connections", s.tableName)
	assert.NotNil(t, s.client)
}

func TestPutConnection_Success(t *testing.T) {
	client := new(mockDynamoDBQueryAPI)
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		return *in.TableName == "Connections"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := newTestConnectionStore(client).PutConnection(context.Background(), sampleConnection())

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestPutConnection_Error(t *testing.T) {
	client := new(mockDynamoDBQueryAPI)
	client.On("PutItem", mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, errors.New("network error"))

	err := newTestConnectionStore(client).PutConnection(context.Background(), sampleConnection())

	require.Error(t, err)
}

func TestGetConnection_Success(t *testing.T) {
	want := sampleConnection()
	item, err := attributevalue.MarshalMap(want)
	require.NoError(t, err)

	client := new(mockDynamoDBQueryAPI)
	client.On("GetItem", mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{Item: item}, nil)

	got, err := newTestConnectionStore(client).GetConnection(context.Background(), "conn-1")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetConnection_NotFound(t *testing.T) {
	client := new(mockDynamoDBQueryAPI)
	client.On("GetItem", mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{Item: nil}, nil)

	_, err := newTestConnectionStore(client).GetConnection(context.Background(), "missing")

	assert.ErrorIs(t, err, ErrConnectionNotFound)
}

func TestGetConnection_Error(t *testing.T) {
	client := new(mockDynamoDBQueryAPI)
	client.On("GetItem", mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, errors.New("network error"))

	_, err := newTestConnectionStore(client).GetConnection(context.Background(), "conn-1")

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrConnectionNotFound)
}

func TestDeleteConnection_Success(t *testing.T) {
	client := new(mockDynamoDBQueryAPI)
	client.On("DeleteItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.DeleteItemInput) bool {
		return *in.TableName == "Connections"
	})).Return(&dynamodb.DeleteItemOutput{}, nil)

	err := newTestConnectionStore(client).DeleteConnection(context.Background(), "conn-1")

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestDeleteConnection_Error(t *testing.T) {
	client := new(mockDynamoDBQueryAPI)
	client.On("DeleteItem", mock.Anything, mock.Anything).
		Return(&dynamodb.DeleteItemOutput{}, errors.New("network error"))

	err := newTestConnectionStore(client).DeleteConnection(context.Background(), "conn-1")

	require.Error(t, err)
}

func TestListConnectionsByGame_Success(t *testing.T) {
	want1 := &Connection{ConnectionID: "conn-1", GameID: "game-1", Role: RolePlayer}
	want2 := &Connection{ConnectionID: "conn-2", GameID: "game-1", Role: RoleSpectator}
	item1, err := attributevalue.MarshalMap(want1)
	require.NoError(t, err)
	item2, err := attributevalue.MarshalMap(want2)
	require.NoError(t, err)

	client := new(mockDynamoDBQueryAPI)
	client.On("Query", mock.Anything, mock.MatchedBy(func(in *dynamodb.QueryInput) bool {
		return *in.IndexName == gameConnIndex && *in.TableName == "Connections"
	})).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item1, item2}}, nil)

	got, err := newTestConnectionStore(client).ListConnectionsByGame(context.Background(), "game-1")

	require.NoError(t, err)
	assert.Equal(t, []*Connection{want1, want2}, got)
}

func TestListConnectionsByGame_Error(t *testing.T) {
	client := new(mockDynamoDBQueryAPI)
	client.On("Query", mock.Anything, mock.Anything).
		Return(&dynamodb.QueryOutput{}, errors.New("network error"))

	_, err := newTestConnectionStore(client).ListConnectionsByGame(context.Background(), "game-1")

	require.Error(t, err)
}
