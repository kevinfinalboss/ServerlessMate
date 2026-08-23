package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ErrMatchClaimFailed = errors.New("store: match already claimed by another invocation")

type MatchStore interface {
	CreateMatch(ctx context.Context, game *Game, a, b *QueueEntry) error
}

type dynamoDBTransactAPI interface {
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

type DynamoMatchStore struct {
	client         dynamoDBTransactAPI
	gamesTableName string
	queueTableName string
}

func NewDynamoMatchStore(client *dynamodb.Client, gamesTableName, queueTableName string) *DynamoMatchStore {
	return &DynamoMatchStore{client: client, gamesTableName: gamesTableName, queueTableName: queueTableName}
}

func (s *DynamoMatchStore) CreateMatch(ctx context.Context, game *Game, a, b *QueueEntry) error {
	gameItem, err := attributevalue.MarshalMap(game)
	if err != nil {
		return fmt.Errorf("store: marshal game: %w", err)
	}

	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{
				TableName:           &s.gamesTableName,
				Item:                gameItem,
				ConditionExpression: strPtr("attribute_not_exists(gameId)"),
			}},
			{Delete: &types.Delete{
				TableName:           &s.queueTableName,
				Key:                 queueKey(a),
				ConditionExpression: strPtr("attribute_exists(matchmakingKey)"),
			}},
			{Delete: &types.Delete{
				TableName:           &s.queueTableName,
				Key:                 queueKey(b),
				ConditionExpression: strPtr("attribute_exists(matchmakingKey)"),
			}},
		},
	})
	if err != nil {
		var canceled *types.TransactionCanceledException
		if errors.As(err, &canceled) {
			return ErrMatchClaimFailed
		}
		return fmt.Errorf("store: create match: %w", err)
	}
	return nil
}

func queueKey(e *QueueEntry) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		"matchmakingKey": &types.AttributeValueMemberS{Value: e.MatchmakingKey},
		"sortKey":        &types.AttributeValueMemberS{Value: e.SortKey},
	}
}
