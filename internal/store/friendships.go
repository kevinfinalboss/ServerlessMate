package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ErrFriendshipConflict = errors.New("store: friendship already accepted or blocked")

type FriendshipStatus string

const (
	FriendshipPending  FriendshipStatus = "pending"
	FriendshipAccepted FriendshipStatus = "accepted"
	FriendshipBlocked  FriendshipStatus = "blocked"
)

type Friendship struct {
	PlayerID  string           `dynamodbav:"playerId"`
	FriendID  string           `dynamodbav:"friendId"`
	Status    FriendshipStatus `dynamodbav:"status"`
	CreatedAt int64            `dynamodbav:"createdAt"`
}

type FriendshipStore interface {
	GetFriendship(ctx context.Context, playerID, friendID string) (*Friendship, error)
	ListFriendships(ctx context.Context, playerID string) ([]*Friendship, error)
	SendRequest(ctx context.Context, playerID, friendID string, at int64) error
	AcceptRequest(ctx context.Context, playerID, friendID string, at int64) error
	Block(ctx context.Context, playerID, friendID string, at int64) error
}

type dynamoDBFriendshipAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

type DynamoFriendshipStore struct {
	client    dynamoDBFriendshipAPI
	tableName string
}

func NewDynamoFriendshipStore(client *dynamodb.Client, tableName string) *DynamoFriendshipStore {
	return &DynamoFriendshipStore{client: client, tableName: tableName}
}

func (s *DynamoFriendshipStore) GetFriendship(ctx context.Context, playerID, friendID string) (*Friendship, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"playerId": &types.AttributeValueMemberS{Value: playerID},
			"friendId": &types.AttributeValueMemberS{Value: friendID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("store: get friendship: %w", err)
	}
	if out.Item == nil {
		return nil, nil
	}

	var f Friendship
	if err := attributevalue.UnmarshalMap(out.Item, &f); err != nil {
		return nil, fmt.Errorf("store: unmarshal friendship: %w", err)
	}
	return &f, nil
}

func (s *DynamoFriendshipStore) ListFriendships(ctx context.Context, playerID string) ([]*Friendship, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		KeyConditionExpression: strPtr("playerId = :playerId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":playerId": &types.AttributeValueMemberS{Value: playerID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("store: list friendships: %w", err)
	}

	friendships := make([]*Friendship, len(out.Items))
	for i, item := range out.Items {
		var f Friendship
		if err := attributevalue.UnmarshalMap(item, &f); err != nil {
			return nil, fmt.Errorf("store: unmarshal friendship: %w", err)
		}
		friendships[i] = &f
	}
	return friendships, nil
}

func (s *DynamoFriendshipStore) SendRequest(ctx context.Context, playerID, friendID string, at int64) error {
	item, err := attributevalue.MarshalMap(&Friendship{
		PlayerID: playerID, FriendID: friendID, Status: FriendshipPending, CreatedAt: at,
	})
	if err != nil {
		return fmt.Errorf("store: marshal friendship: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:                &s.tableName,
		Item:                     item,
		ConditionExpression:      strPtr("attribute_not_exists(#status) OR (#status <> :accepted AND #status <> :blocked)"),
		ExpressionAttributeNames: map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":accepted": &types.AttributeValueMemberS{Value: string(FriendshipAccepted)},
			":blocked":  &types.AttributeValueMemberS{Value: string(FriendshipBlocked)},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrFriendshipConflict
		}
		return fmt.Errorf("store: send friend request: %w", err)
	}
	return nil
}

func (s *DynamoFriendshipStore) AcceptRequest(ctx context.Context, playerID, friendID string, at int64) error {
	forward, err := attributevalue.MarshalMap(&Friendship{PlayerID: playerID, FriendID: friendID, Status: FriendshipAccepted, CreatedAt: at})
	if err != nil {
		return fmt.Errorf("store: marshal friendship: %w", err)
	}
	backward, err := attributevalue.MarshalMap(&Friendship{PlayerID: friendID, FriendID: playerID, Status: FriendshipAccepted, CreatedAt: at})
	if err != nil {
		return fmt.Errorf("store: marshal friendship: %w", err)
	}

	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{TableName: &s.tableName, Item: forward}},
			{Put: &types.Put{TableName: &s.tableName, Item: backward}},
		},
	})
	if err != nil {
		return fmt.Errorf("store: accept friend request: %w", err)
	}
	return nil
}

func (s *DynamoFriendshipStore) Block(ctx context.Context, playerID, friendID string, at int64) error {
	blockItem, err := attributevalue.MarshalMap(&Friendship{PlayerID: playerID, FriendID: friendID, Status: FriendshipBlocked, CreatedAt: at})
	if err != nil {
		return fmt.Errorf("store: marshal friendship: %w", err)
	}

	_, err = s.client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{Put: &types.Put{TableName: &s.tableName, Item: blockItem}},
			{Delete: &types.Delete{
				TableName: &s.tableName,
				Key: map[string]types.AttributeValue{
					"playerId": &types.AttributeValueMemberS{Value: friendID},
					"friendId": &types.AttributeValueMemberS{Value: playerID},
				},
			}},
		},
	})
	if err != nil {
		return fmt.Errorf("store: block friendship: %w", err)
	}
	return nil
}
