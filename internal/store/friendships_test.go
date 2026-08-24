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

type mockDynamoDBFriendshipAPI struct {
	mock.Mock
}

func (m *mockDynamoDBFriendshipAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.GetItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBFriendshipAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.PutItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBFriendshipAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.QueryOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBFriendshipAPI) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.DeleteItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBFriendshipAPI) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.TransactWriteItemsOutput)
	return out, args.Error(1)
}

func newTestFriendshipStore(client *mockDynamoDBFriendshipAPI) *DynamoFriendshipStore {
	return &DynamoFriendshipStore{client: client, tableName: "Friendships"}
}

func TestNewDynamoFriendshipStore(t *testing.T) {
	s := NewDynamoFriendshipStore(&dynamodb.Client{}, "Friendships")

	assert.Equal(t, "Friendships", s.tableName)
	assert.NotNil(t, s.client)
}

func TestGetFriendship_Found(t *testing.T) {
	want := &Friendship{PlayerID: "player-1", FriendID: "player-2", Status: FriendshipAccepted, CreatedAt: 1_700_000_000_000}
	item, err := attributevalue.MarshalMap(want)
	require.NoError(t, err)

	client := new(mockDynamoDBFriendshipAPI)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	got, err := newTestFriendshipStore(client).GetFriendship(context.Background(), "player-1", "player-2")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetFriendship_NotFound(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: nil}, nil)

	got, err := newTestFriendshipStore(client).GetFriendship(context.Background(), "player-1", "player-2")

	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGetFriendship_Error(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{}, errors.New("network error"))

	_, err := newTestFriendshipStore(client).GetFriendship(context.Background(), "player-1", "player-2")

	require.Error(t, err)
}

func TestListFriendships_Success(t *testing.T) {
	f1, err := attributevalue.MarshalMap(&Friendship{PlayerID: "player-1", FriendID: "player-2", Status: FriendshipAccepted})
	require.NoError(t, err)

	client := new(mockDynamoDBFriendshipAPI)
	client.On("Query", mock.Anything, mock.MatchedBy(func(in *dynamodb.QueryInput) bool {
		return *in.TableName == "Friendships"
	})).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{f1}}, nil)

	got, err := newTestFriendshipStore(client).ListFriendships(context.Background(), "player-1")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "player-2", got[0].FriendID)
}

func TestListFriendships_Error(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{}, errors.New("network error"))

	_, err := newTestFriendshipStore(client).ListFriendships(context.Background(), "player-1")

	require.Error(t, err)
}

func TestListIncomingRequests_Success(t *testing.T) {
	f1, err := attributevalue.MarshalMap(&Friendship{PlayerID: "player-2", FriendID: "player-1", Status: FriendshipPending})
	require.NoError(t, err)

	client := new(mockDynamoDBFriendshipAPI)
	client.On("Query", mock.Anything, mock.MatchedBy(func(in *dynamodb.QueryInput) bool {
		return *in.TableName == "Friendships" && *in.IndexName == "FriendIDIndex"
	})).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{f1}}, nil)

	got, err := newTestFriendshipStore(client).ListIncomingRequests(context.Background(), "player-1")

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "player-2", got[0].PlayerID)
}

func TestListIncomingRequests_Error(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{}, errors.New("network error"))

	_, err := newTestFriendshipStore(client).ListIncomingRequests(context.Background(), "player-1")

	require.Error(t, err)
}

func TestCancelRequest_DeletesBothDirections(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("DeleteItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.DeleteItemInput) bool {
		return in.Key["playerId"].(*types.AttributeValueMemberS).Value == "player-1" &&
			in.Key["friendId"].(*types.AttributeValueMemberS).Value == "player-2"
	})).Return(&dynamodb.DeleteItemOutput{}, &types.ConditionalCheckFailedException{})
	client.On("DeleteItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.DeleteItemInput) bool {
		return in.Key["playerId"].(*types.AttributeValueMemberS).Value == "player-2" &&
			in.Key["friendId"].(*types.AttributeValueMemberS).Value == "player-1"
	})).Return(&dynamodb.DeleteItemOutput{}, nil)

	err := newTestFriendshipStore(client).CancelRequest(context.Background(), "player-1", "player-2")

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestCancelRequest_Error(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("DeleteItem", mock.Anything, mock.Anything).
		Return(&dynamodb.DeleteItemOutput{}, errors.New("network error"))

	err := newTestFriendshipStore(client).CancelRequest(context.Background(), "player-1", "player-2")

	require.Error(t, err)
}

func TestSendRequest_Success(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		return *in.TableName == "Friendships"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := newTestFriendshipStore(client).SendRequest(context.Background(), "player-1", "player-2", 1_700_000_000_000)

	require.NoError(t, err)
}

func TestSendRequest_Conflict(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("PutItem", mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, &types.ConditionalCheckFailedException{})

	err := newTestFriendshipStore(client).SendRequest(context.Background(), "player-1", "player-2", 1_700_000_000_000)

	assert.ErrorIs(t, err, ErrFriendshipConflict)
}

func TestSendRequest_Error(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("PutItem", mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, errors.New("network error"))

	err := newTestFriendshipStore(client).SendRequest(context.Background(), "player-1", "player-2", 1_700_000_000_000)

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrFriendshipConflict)
}

func TestAcceptRequest_Success(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("TransactWriteItems", mock.Anything, mock.MatchedBy(func(in *dynamodb.TransactWriteItemsInput) bool {
		return len(in.TransactItems) == 2
	})).Return(&dynamodb.TransactWriteItemsOutput{}, nil)

	err := newTestFriendshipStore(client).AcceptRequest(context.Background(), "player-1", "player-2", 1_700_000_000_000)

	require.NoError(t, err)
}

func TestAcceptRequest_Error(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("TransactWriteItems", mock.Anything, mock.Anything).
		Return(&dynamodb.TransactWriteItemsOutput{}, errors.New("network error"))

	err := newTestFriendshipStore(client).AcceptRequest(context.Background(), "player-1", "player-2", 1_700_000_000_000)

	require.Error(t, err)
}

func TestBlock_Success(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("TransactWriteItems", mock.Anything, mock.MatchedBy(func(in *dynamodb.TransactWriteItemsInput) bool {
		return len(in.TransactItems) == 2 &&
			in.TransactItems[0].Put != nil &&
			in.TransactItems[1].Delete != nil
	})).Return(&dynamodb.TransactWriteItemsOutput{}, nil)

	err := newTestFriendshipStore(client).Block(context.Background(), "player-1", "player-2", 1_700_000_000_000)

	require.NoError(t, err)
}

func TestBlock_Error(t *testing.T) {
	client := new(mockDynamoDBFriendshipAPI)
	client.On("TransactWriteItems", mock.Anything, mock.Anything).
		Return(&dynamodb.TransactWriteItemsOutput{}, errors.New("network error"))

	err := newTestFriendshipStore(client).Block(context.Background(), "player-1", "player-2", 1_700_000_000_000)

	require.Error(t, err)
}
