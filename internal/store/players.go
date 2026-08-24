package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ErrPlayerNotFound = errors.New("store: player not found")

const leaderboardGlobalPK = "GLOBAL"
const leaderboardIndex = "LeaderboardIndex"
const defaultRating = 1200

type GameOutcome int

const (
	OutcomeWin GameOutcome = iota
	OutcomeLoss
	OutcomeDraw
)

type Player struct {
	PlayerID      string `dynamodbav:"playerId"`
	Username      string `dynamodbav:"username"`
	Rating        int    `dynamodbav:"rating"`
	Wins          int    `dynamodbav:"wins"`
	Losses        int    `dynamodbav:"losses"`
	Draws         int    `dynamodbav:"draws"`
	GamesPlayed   int    `dynamodbav:"gamesPlayed"`
	Visibility    string `dynamodbav:"visibility"`
	CreatedAt     int64  `dynamodbav:"createdAt"`
	LeaderboardPK string `dynamodbav:"leaderboardPK,omitempty"`
}

type PlayerStore interface {
	GetPlayer(ctx context.Context, playerID string) (*Player, error)
	GetOrCreatePlayer(ctx context.Context, playerID string, now int64) (*Player, error)
	RecordGameResult(ctx context.Context, playerID string, newRating int, outcome GameOutcome) error
	ListTopByRating(ctx context.Context, limit int32) ([]*Player, error)
}

type dynamoDBUpdateAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type DynamoPlayerStore struct {
	client    dynamoDBUpdateAPI
	tableName string
}

func NewDynamoPlayerStore(client *dynamodb.Client, tableName string) *DynamoPlayerStore {
	return &DynamoPlayerStore{client: client, tableName: tableName}
}

func (s *DynamoPlayerStore) GetPlayer(ctx context.Context, playerID string) (*Player, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"playerId": &types.AttributeValueMemberS{Value: playerID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("store: get player: %w", err)
	}
	if out.Item == nil {
		return nil, ErrPlayerNotFound
	}

	var p Player
	if err := attributevalue.UnmarshalMap(out.Item, &p); err != nil {
		return nil, fmt.Errorf("store: unmarshal player: %w", err)
	}
	return &p, nil
}

func (s *DynamoPlayerStore) GetOrCreatePlayer(ctx context.Context, playerID string, now int64) (*Player, error) {
	p, err := s.GetPlayer(ctx, playerID)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, ErrPlayerNotFound) {
		return nil, err
	}

	item, err := attributevalue.MarshalMap(&Player{
		PlayerID:   playerID,
		Username:   defaultUsername(playerID),
		Rating:     defaultRating,
		Visibility: "public",
		CreatedAt:  now,
	})
	if err != nil {
		return nil, fmt.Errorf("store: marshal default player: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           &s.tableName,
		Item:                item,
		ConditionExpression: strPtr("attribute_not_exists(playerId)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return s.GetPlayer(ctx, playerID)
		}
		return nil, fmt.Errorf("store: create default player: %w", err)
	}

	return s.GetPlayer(ctx, playerID)
}

func defaultUsername(playerID string) string {
	if len(playerID) > 8 {
		return "Player-" + playerID[:8]
	}
	return "Player-" + playerID
}

func (s *DynamoPlayerStore) RecordGameResult(ctx context.Context, playerID string, newRating int, outcome GameOutcome) error {
	counterAttr := "draws"
	switch outcome {
	case OutcomeWin:
		counterAttr = "wins"
	case OutcomeLoss:
		counterAttr = "losses"
	}

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"playerId": &types.AttributeValueMemberS{Value: playerID},
		},
		UpdateExpression: strPtr(fmt.Sprintf(
			"SET rating = :rating, leaderboardPK = :leaderboardPK ADD gamesPlayed :one, %s :one", counterAttr,
		)),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":rating":        &types.AttributeValueMemberN{Value: strconv.Itoa(newRating)},
			":leaderboardPK": &types.AttributeValueMemberS{Value: leaderboardGlobalPK},
			":one":           &types.AttributeValueMemberN{Value: "1"},
		},
	})
	if err != nil {
		return fmt.Errorf("store: record game result: %w", err)
	}
	return nil
}

func (s *DynamoPlayerStore) ListTopByRating(ctx context.Context, limit int32) ([]*Player, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		IndexName:              strPtr(leaderboardIndex),
		KeyConditionExpression: strPtr("leaderboardPK = :leaderboardPK"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":leaderboardPK": &types.AttributeValueMemberS{Value: leaderboardGlobalPK},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("store: list top players: %w", err)
	}

	players := make([]*Player, len(out.Items))
	for i, item := range out.Items {
		var p Player
		if err := attributevalue.UnmarshalMap(item, &p); err != nil {
			return nil, fmt.Errorf("store: unmarshal player: %w", err)
		}
		players[i] = &p
	}
	return players, nil
}
