package store

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type RateLimitStore interface {
	IncrementAndCheck(ctx context.Context, playerID, date string, limit int, ttl int64) (bool, error)
}

type dynamoDBRateLimitAPI interface {
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

type DynamoRateLimitStore struct {
	client    dynamoDBRateLimitAPI
	tableName string
}

func NewDynamoRateLimitStore(client *dynamodb.Client, tableName string) *DynamoRateLimitStore {
	return &DynamoRateLimitStore{client: client, tableName: tableName}
}

func (s *DynamoRateLimitStore) IncrementAndCheck(ctx context.Context, playerID, date string, limit int, ttl int64) (bool, error) {
	key := playerID + "#" + date

	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"playerDate": &types.AttributeValueMemberS{Value: key},
		},
		UpdateExpression:         strPtr("SET #ttl = if_not_exists(#ttl, :ttl) ADD #count :one"),
		ConditionExpression:      strPtr("attribute_not_exists(#count) OR #count < :limit"),
		ExpressionAttributeNames: map[string]string{"#count": "count", "#ttl": "ttl"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":one":   &types.AttributeValueMemberN{Value: "1"},
			":limit": &types.AttributeValueMemberN{Value: strconv.Itoa(limit)},
			":ttl":   &types.AttributeValueMemberN{Value: strconv.FormatInt(ttl, 10)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return false, nil
		}
		return false, fmt.Errorf("store: increment rate limit: %w", err)
	}
	return true, nil
}
