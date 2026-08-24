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

type mockDynamoDBUpdateAPI struct {
	mock.Mock
}

func (m *mockDynamoDBUpdateAPI) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.GetItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBUpdateAPI) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.PutItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBUpdateAPI) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.UpdateItemOutput)
	return out, args.Error(1)
}

func (m *mockDynamoDBUpdateAPI) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*dynamodb.QueryOutput)
	return out, args.Error(1)
}

func newTestPlayerStore(client *mockDynamoDBUpdateAPI) *DynamoPlayerStore {
	return &DynamoPlayerStore{client: client, tableName: "Players"}
}

func samplePlayer() *Player {
	return &Player{
		PlayerID:    "player-1",
		Username:    "kasparov",
		Rating:      1200,
		Wins:        3,
		Losses:      2,
		Draws:       1,
		GamesPlayed: 6,
		Visibility:  "public",
		CreatedAt:   1_700_000_000_000,
	}
}

func TestNewDynamoPlayerStore(t *testing.T) {
	s := NewDynamoPlayerStore(&dynamodb.Client{}, "Players")

	assert.Equal(t, "Players", s.tableName)
	assert.NotNil(t, s.client)
}

func TestGetPlayer_Success(t *testing.T) {
	want := samplePlayer()
	item, err := attributevalue.MarshalMap(want)
	require.NoError(t, err)

	client := new(mockDynamoDBUpdateAPI)
	client.On("GetItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.GetItemInput) bool {
		return *in.TableName == "Players"
	})).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	got, err := newTestPlayerStore(client).GetPlayer(context.Background(), "player-1")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetPlayer_NotFound(t *testing.T) {
	client := new(mockDynamoDBUpdateAPI)
	client.On("GetItem", mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{Item: nil}, nil)

	_, err := newTestPlayerStore(client).GetPlayer(context.Background(), "missing")

	assert.ErrorIs(t, err, ErrPlayerNotFound)
}

func TestGetPlayer_Error(t *testing.T) {
	client := new(mockDynamoDBUpdateAPI)
	client.On("GetItem", mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, errors.New("network error"))

	_, err := newTestPlayerStore(client).GetPlayer(context.Background(), "player-1")

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrPlayerNotFound)
}

func TestGetOrCreatePlayer_ExistingPlayer(t *testing.T) {
	want := samplePlayer()
	item, err := attributevalue.MarshalMap(want)
	require.NoError(t, err)

	client := new(mockDynamoDBUpdateAPI)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	got, err := newTestPlayerStore(client).GetOrCreatePlayer(context.Background(), "player-1", 1_700_000_000_000)

	require.NoError(t, err)
	assert.Equal(t, want, got)
	client.AssertNotCalled(t, "PutItem")
}

func TestGetOrCreatePlayer_CreatesDefaultWhenMissing(t *testing.T) {
	created := &Player{
		PlayerID:   "player-9",
		Username:   "Player-player-9",
		Rating:     defaultRating,
		Visibility: "public",
		CreatedAt:  1_700_000_000_000,
	}
	createdItem, err := attributevalue.MarshalMap(created)
	require.NoError(t, err)

	client := new(mockDynamoDBUpdateAPI)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: nil}, nil).Once()
	client.On("PutItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.PutItemInput) bool {
		return *in.TableName == "Players"
	})).Return(&dynamodb.PutItemOutput{}, nil)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: createdItem}, nil).Once()

	got, err := newTestPlayerStore(client).GetOrCreatePlayer(context.Background(), "player-9", 1_700_000_000_000)

	require.NoError(t, err)
	assert.Equal(t, created, got)
}

func TestGetOrCreatePlayer_RaceFallsBackToGet(t *testing.T) {
	existing := samplePlayer()
	existingItem, err := attributevalue.MarshalMap(existing)
	require.NoError(t, err)

	client := new(mockDynamoDBUpdateAPI)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: nil}, nil).Once()
	client.On("PutItem", mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, &types.ConditionalCheckFailedException{})
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: existingItem}, nil).Once()

	got, err := newTestPlayerStore(client).GetOrCreatePlayer(context.Background(), "player-1", 1_700_000_000_000)

	require.NoError(t, err)
	assert.Equal(t, existing, got)
}

func TestGetOrCreatePlayer_PutItemError(t *testing.T) {
	client := new(mockDynamoDBUpdateAPI)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: nil}, nil).Once()
	client.On("PutItem", mock.Anything, mock.Anything).Return(&dynamodb.PutItemOutput{}, errors.New("network error"))

	_, err := newTestPlayerStore(client).GetOrCreatePlayer(context.Background(), "player-1", 1_700_000_000_000)

	require.Error(t, err)
}

func TestGetOrCreatePlayer_GetItemError(t *testing.T) {
	client := new(mockDynamoDBUpdateAPI)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{}, errors.New("network error"))

	_, err := newTestPlayerStore(client).GetOrCreatePlayer(context.Background(), "player-1", 1_700_000_000_000)

	require.Error(t, err)
	client.AssertNotCalled(t, "PutItem")
}

func TestUpdateProfile_Success(t *testing.T) {
	want := samplePlayer()
	item, err := attributevalue.MarshalMap(want)
	require.NoError(t, err)

	client := new(mockDynamoDBUpdateAPI)
	client.On("UpdateItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.UpdateItemInput) bool {
		return *in.TableName == "Players" &&
			in.ExpressionAttributeValues[":visibility"].(*types.AttributeValueMemberS).Value == "friends" &&
			in.ExpressionAttributeValues[":country"].(*types.AttributeValueMemberS).Value == "BR"
	})).Return(&dynamodb.UpdateItemOutput{}, nil)
	client.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{Item: item}, nil)

	got, err := newTestPlayerStore(client).UpdateProfile(context.Background(), "player-1", ProfileUpdate{
		Visibility: "friends",
		Country:    "BR",
	})

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestUpdateProfile_NotFound(t *testing.T) {
	client := new(mockDynamoDBUpdateAPI)
	client.On("UpdateItem", mock.Anything, mock.Anything).
		Return(&dynamodb.UpdateItemOutput{}, &types.ConditionalCheckFailedException{})

	_, err := newTestPlayerStore(client).UpdateProfile(context.Background(), "player-1", ProfileUpdate{})

	assert.ErrorIs(t, err, ErrPlayerNotFound)
}

func TestUpdateProfile_Error(t *testing.T) {
	client := new(mockDynamoDBUpdateAPI)
	client.On("UpdateItem", mock.Anything, mock.Anything).
		Return(&dynamodb.UpdateItemOutput{}, errors.New("network error"))

	_, err := newTestPlayerStore(client).UpdateProfile(context.Background(), "player-1", ProfileUpdate{})

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrPlayerNotFound)
}

func TestRecordGameResult(t *testing.T) {
	cases := []struct {
		name    string
		outcome GameOutcome
		wantExp string
	}{
		{"win", OutcomeWin, "SET rating = :rating, leaderboardPK = :leaderboardPK ADD gamesPlayed :one, wins :one"},
		{"loss", OutcomeLoss, "SET rating = :rating, leaderboardPK = :leaderboardPK ADD gamesPlayed :one, losses :one"},
		{"draw", OutcomeDraw, "SET rating = :rating, leaderboardPK = :leaderboardPK ADD gamesPlayed :one, draws :one"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := new(mockDynamoDBUpdateAPI)
			client.On("UpdateItem", mock.Anything, mock.MatchedBy(func(in *dynamodb.UpdateItemInput) bool {
				return *in.TableName == "Players" && *in.UpdateExpression == tc.wantExp
			})).Return(&dynamodb.UpdateItemOutput{}, nil)

			err := newTestPlayerStore(client).RecordGameResult(context.Background(), "player-1", 1216, tc.outcome)

			require.NoError(t, err)
			client.AssertExpectations(t)
		})
	}
}

func TestRecordGameResult_Error(t *testing.T) {
	client := new(mockDynamoDBUpdateAPI)
	client.On("UpdateItem", mock.Anything, mock.Anything).
		Return(&dynamodb.UpdateItemOutput{}, errors.New("network error"))

	err := newTestPlayerStore(client).RecordGameResult(context.Background(), "player-1", 1216, OutcomeWin)

	require.Error(t, err)
}

func TestListTopByRating_Success(t *testing.T) {
	want1 := &Player{PlayerID: "player-1", Rating: 1800, LeaderboardPK: leaderboardGlobalPK}
	want2 := &Player{PlayerID: "player-2", Rating: 1600, LeaderboardPK: leaderboardGlobalPK}
	item1, err := attributevalue.MarshalMap(want1)
	require.NoError(t, err)
	item2, err := attributevalue.MarshalMap(want2)
	require.NoError(t, err)

	client := new(mockDynamoDBUpdateAPI)
	client.On("Query", mock.Anything, mock.MatchedBy(func(in *dynamodb.QueryInput) bool {
		return *in.IndexName == leaderboardIndex && *in.TableName == "Players" && !*in.ScanIndexForward && *in.Limit == int32(10)
	})).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item1, item2}}, nil)

	got, err := newTestPlayerStore(client).ListTopByRating(context.Background(), 10)

	require.NoError(t, err)
	assert.Equal(t, []*Player{want1, want2}, got)
}

func TestListTopByRating_Error(t *testing.T) {
	client := new(mockDynamoDBUpdateAPI)
	client.On("Query", mock.Anything, mock.Anything).
		Return(&dynamodb.QueryOutput{}, errors.New("network error"))

	_, err := newTestPlayerStore(client).ListTopByRating(context.Background(), 10)

	require.Error(t, err)
}
