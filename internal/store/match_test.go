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

type mockDynamoDBTransactAPI struct {
	mock.Mock
}

func (m *mockDynamoDBTransactAPI) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.TransactWriteItemsOutput)
	return out, args.Error(1)
}

func newTestMatchStore(client *mockDynamoDBTransactAPI) *DynamoMatchStore {
	return &DynamoMatchStore{client: client, gamesTableName: "Games", queueTableName: "Queue"}
}

func TestNewDynamoMatchStore(t *testing.T) {
	s := NewDynamoMatchStore(&dynamodb.Client{}, "Games", "Queue")

	assert.Equal(t, "Games", s.gamesTableName)
	assert.Equal(t, "Queue", s.queueTableName)
	assert.NotNil(t, s.client)
}

func TestCreateMatch_Success(t *testing.T) {
	client := new(mockDynamoDBTransactAPI)
	client.On("TransactWriteItems", mock.Anything, mock.MatchedBy(func(in *dynamodb.TransactWriteItemsInput) bool {
		return len(in.TransactItems) == 3 &&
			*in.TransactItems[0].Put.TableName == "Games" &&
			*in.TransactItems[1].Delete.TableName == "Queue" &&
			*in.TransactItems[2].Delete.TableName == "Queue"
	})).Return(&dynamodb.TransactWriteItemsOutput{}, nil)

	a := NewQueueEntry("5+0#1200", "player-1", "conn-1", false, 1_700_000_000_000)
	b := NewQueueEntry("5+0#1200", "player-2", "conn-2", false, 1_700_000_001_000)
	game := &Game{GameID: "game-1", FEN: "start", Players: Players{White: "player-1", Black: "player-2"}}

	err := newTestMatchStore(client).CreateMatch(context.Background(), game, a, b)

	require.NoError(t, err)
}

func TestCreateMatch_ClaimFailed(t *testing.T) {
	client := new(mockDynamoDBTransactAPI)
	client.On("TransactWriteItems", mock.Anything, mock.Anything).
		Return(&dynamodb.TransactWriteItemsOutput{}, &types.TransactionCanceledException{})

	a := NewQueueEntry("5+0#1200", "player-1", "conn-1", false, 1_700_000_000_000)
	b := NewQueueEntry("5+0#1200", "player-2", "conn-2", false, 1_700_000_001_000)
	game := &Game{GameID: "game-1"}

	err := newTestMatchStore(client).CreateMatch(context.Background(), game, a, b)

	assert.ErrorIs(t, err, ErrMatchClaimFailed)
}

func TestCreateMatch_OtherError(t *testing.T) {
	client := new(mockDynamoDBTransactAPI)
	client.On("TransactWriteItems", mock.Anything, mock.Anything).
		Return(&dynamodb.TransactWriteItemsOutput{}, errors.New("network error"))

	a := NewQueueEntry("5+0#1200", "player-1", "conn-1", false, 1_700_000_000_000)
	b := NewQueueEntry("5+0#1200", "player-2", "conn-2", false, 1_700_000_001_000)
	game := &Game{GameID: "game-1"}

	err := newTestMatchStore(client).CreateMatch(context.Background(), game, a, b)

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrMatchClaimFailed)
}
