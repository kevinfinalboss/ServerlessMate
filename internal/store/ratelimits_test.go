package store

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDynamoDBRateLimitAPI struct {
	mock.Mock
}

func (m *mockDynamoDBRateLimitAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.UpdateItemOutput)
	return out, args.Error(1)
}

func newTestRateLimitStore(client *mockDynamoDBRateLimitAPI) *DynamoRateLimitStore {
	return &DynamoRateLimitStore{client: client, tableName: "RateLimits"}
}

func TestNewDynamoRateLimitStore(t *testing.T) {
	s := NewDynamoRateLimitStore(&dynamodb.Client{}, "RateLimits")

	assert.Equal(t, "RateLimits", s.tableName)
	assert.NotNil(t, s.client)
}

func TestIncrementAndCheck_Allowed(t *testing.T) {
	client := new(mockDynamoDBRateLimitAPI)
	client.On("UpdateItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.UpdateItemInput) bool {
		key, ok := in.Key["playerDate"].(*types.AttributeValueMemberS)
		return *in.TableName == "RateLimits" && ok && key.Value == "player-1#2026-08-23"
	})).Return(&dynamodb.UpdateItemOutput{}, nil)

	allowed, err := newTestRateLimitStore(client).IncrementAndCheck(context.Background(), "player-1", "2026-08-23", 200, 1_700_000_000)

	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIncrementAndCheck_LimitExceeded(t *testing.T) {
	client := new(mockDynamoDBRateLimitAPI)
	client.On("UpdateItem", mock.Anything, mock.Anything).
		Return(&dynamodb.UpdateItemOutput{}, &types.ConditionalCheckFailedException{})

	allowed, err := newTestRateLimitStore(client).IncrementAndCheck(context.Background(), "player-1", "2026-08-23", 200, 1_700_000_000)

	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestIncrementAndCheck_Error(t *testing.T) {
	client := new(mockDynamoDBRateLimitAPI)
	client.On("UpdateItem", mock.Anything, mock.Anything).
		Return(&dynamodb.UpdateItemOutput{}, errors.New("network error"))

	_, err := newTestRateLimitStore(client).IncrementAndCheck(context.Background(), "player-1", "2026-08-23", 200, 1_700_000_000)

	require.Error(t, err)
}
