package store

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	ResultWin  = "win"
	ResultLoss = "loss"
	ResultDraw = "draw"
)

type HistoryEntry struct {
	PlayerID   string `dynamodbav:"playerId"`
	EndedAt    int64  `dynamodbav:"endedAt"`
	GameID     string `dynamodbav:"gameId"`
	OpponentID string `dynamodbav:"opponentId"`
	Result     string `dynamodbav:"result"`
	VsAI       bool   `dynamodbav:"vsAI"`
}

type HistoryStore interface {
	RecordGameEnd(ctx context.Context, g *Game) error
	ListHistory(ctx context.Context, playerID string, limit int32) ([]*HistoryEntry, error)
}

type dynamoDBHistoryAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type DynamoHistoryStore struct {
	client    dynamoDBHistoryAPI
	tableName string
}

func NewDynamoHistoryStore(client *dynamodb.Client, tableName string) *DynamoHistoryStore {
	return &DynamoHistoryStore{client: client, tableName: tableName}
}

func (s *DynamoHistoryStore) RecordGameEnd(ctx context.Context, g *Game) error {
	if err := s.putEntry(ctx, g, g.Players.White, g.Players.Black); err != nil {
		return err
	}
	if g.VsAI {
		return nil
	}
	return s.putEntry(ctx, g, g.Players.Black, g.Players.White)
}

func (s *DynamoHistoryStore) putEntry(ctx context.Context, g *Game, playerID, opponentID string) error {
	result := ResultDraw
	if g.Winner != "" {
		if g.Winner == playerID {
			result = ResultWin
		} else {
			result = ResultLoss
		}
	}

	item, err := attributevalue.MarshalMap(&HistoryEntry{
		PlayerID:   playerID,
		EndedAt:    g.EndedAt,
		GameID:     g.GameID,
		OpponentID: opponentID,
		Result:     result,
		VsAI:       g.VsAI,
	})
	if err != nil {
		return fmt.Errorf("store: marshal history entry: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("store: record history: %w", err)
	}
	return nil
}

func (s *DynamoHistoryStore) ListHistory(ctx context.Context, playerID string, limit int32) ([]*HistoryEntry, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		KeyConditionExpression: strPtr("playerId = :playerId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":playerId": &types.AttributeValueMemberS{Value: playerID},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("store: list history: %w", err)
	}

	entries := make([]*HistoryEntry, len(out.Items))
	for i, item := range out.Items {
		var e HistoryEntry
		if err := attributevalue.UnmarshalMap(item, &e); err != nil {
			return nil, fmt.Errorf("store: unmarshal history entry: %w", err)
		}
		entries[i] = &e
	}
	return entries, nil
}
