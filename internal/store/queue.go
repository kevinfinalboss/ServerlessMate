package store

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type QueueEntry struct {
	MatchmakingKey string `dynamodbav:"matchmakingKey"`
	SortKey        string `dynamodbav:"sortKey"`
	PlayerID       string `dynamodbav:"playerId"`
	ConnectionID   string `dynamodbav:"connectionId"`
	IsGuest        bool   `dynamodbav:"isGuest"`
	JoinedAt       int64  `dynamodbav:"joinedAt"`
}

func NewQueueEntry(matchmakingKey, playerID, connectionID string, isGuest bool, joinedAt int64) *QueueEntry {
	return &QueueEntry{
		MatchmakingKey: matchmakingKey,
		SortKey:        fmt.Sprintf("%d#%s", joinedAt, playerID),
		PlayerID:       playerID,
		ConnectionID:   connectionID,
		IsGuest:        isGuest,
		JoinedAt:       joinedAt,
	}
}

type QueueStore interface {
	Join(ctx context.Context, e *QueueEntry) error
	FindWaiting(ctx context.Context, matchmakingKey, excludePlayerID string) (*QueueEntry, error)
	Leave(ctx context.Context, matchmakingKey, playerID string) error
}

type dynamoDBQueueAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

type DynamoQueueStore struct {
	client    dynamoDBQueueAPI
	tableName string
}

func NewDynamoQueueStore(client *dynamodb.Client, tableName string) *DynamoQueueStore {
	return &DynamoQueueStore{client: client, tableName: tableName}
}

func (s *DynamoQueueStore) Join(ctx context.Context, e *QueueEntry) error {
	item, err := attributevalue.MarshalMap(e)
	if err != nil {
		return fmt.Errorf("store: marshal queue entry: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("store: join queue: %w", err)
	}
	return nil
}

func (s *DynamoQueueStore) FindWaiting(ctx context.Context, matchmakingKey, excludePlayerID string) (*QueueEntry, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		KeyConditionExpression: strPtr("matchmakingKey = :key"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":key": &types.AttributeValueMemberS{Value: matchmakingKey},
		},
		ScanIndexForward: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("store: find waiting player: %w", err)
	}

	for _, item := range out.Items {
		var e QueueEntry
		if err := attributevalue.UnmarshalMap(item, &e); err != nil {
			return nil, fmt.Errorf("store: unmarshal queue entry: %w", err)
		}
		if e.PlayerID != excludePlayerID {
			return &e, nil
		}
	}
	return nil, nil
}

func (s *DynamoQueueStore) Leave(ctx context.Context, matchmakingKey, playerID string) error {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		KeyConditionExpression: strPtr("matchmakingKey = :key"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":key": &types.AttributeValueMemberS{Value: matchmakingKey},
		},
	})
	if err != nil {
		return fmt.Errorf("store: find queue entry to leave: %w", err)
	}

	for _, item := range out.Items {
		var e QueueEntry
		if err := attributevalue.UnmarshalMap(item, &e); err != nil {
			return fmt.Errorf("store: unmarshal queue entry: %w", err)
		}
		if e.PlayerID != playerID {
			continue
		}

		_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName:           &s.tableName,
			Key:                 queueKey(&e),
			ConditionExpression: strPtr("attribute_exists(matchmakingKey)"),
		})
		if err != nil {
			if isConditionalCheckFailed(err) {
				return nil
			}
			return fmt.Errorf("store: leave queue: %w", err)
		}
		return nil
	}
	return nil
}
