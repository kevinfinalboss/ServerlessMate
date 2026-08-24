package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ErrConnectionNotFound = errors.New("store: connection not found")

const (
	RolePlayer    = "player"
	RoleSpectator = "spectator"
	gameConnIndex = "GameConnectionsIndex"
)

type Connection struct {
	ConnectionID string `dynamodbav:"connectionId"`
	GameID       string `dynamodbav:"gameId,omitempty"`
	PlayerID     string `dynamodbav:"playerId"`
	IsGuest      bool   `dynamodbav:"isGuest"`
	Role         string `dynamodbav:"role"`
}

type ConnectionStore interface {
	PutConnection(ctx context.Context, c *Connection) error
	GetConnection(ctx context.Context, connectionID string) (*Connection, error)
	DeleteConnection(ctx context.Context, connectionID string) error
	ListConnectionsByGame(ctx context.Context, gameID string) ([]*Connection, error)
}

type dynamoDBQueryAPI interface {
	dynamoDBAPI
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type DynamoConnectionStore struct {
	client    dynamoDBQueryAPI
	tableName string
}

func NewDynamoConnectionStore(client *dynamodb.Client, tableName string) *DynamoConnectionStore {
	return &DynamoConnectionStore{client: client, tableName: tableName}
}

func (s *DynamoConnectionStore) PutConnection(ctx context.Context, c *Connection) error {
	item, err := attributevalue.MarshalMap(c)
	if err != nil {
		return fmt.Errorf("store: marshal connection: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.tableName,
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("store: put connection: %w", err)
	}
	return nil
}

func (s *DynamoConnectionStore) GetConnection(ctx context.Context, connectionID string) (*Connection, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"connectionId": &types.AttributeValueMemberS{Value: connectionID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("store: get connection: %w", err)
	}
	if out.Item == nil {
		return nil, ErrConnectionNotFound
	}

	var c Connection
	if err := attributevalue.UnmarshalMap(out.Item, &c); err != nil {
		return nil, fmt.Errorf("store: unmarshal connection: %w", err)
	}
	return &c, nil
}

func (s *DynamoConnectionStore) DeleteConnection(ctx context.Context, connectionID string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"connectionId": &types.AttributeValueMemberS{Value: connectionID},
		},
	})
	if err != nil {
		return fmt.Errorf("store: delete connection: %w", err)
	}
	return nil
}

func (s *DynamoConnectionStore) ListConnectionsByGame(ctx context.Context, gameID string) ([]*Connection, error) {
	out, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              &s.tableName,
		IndexName:              strPtr(gameConnIndex),
		KeyConditionExpression: strPtr("gameId = :gameId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gameId": &types.AttributeValueMemberS{Value: gameID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("store: list connections by game: %w", err)
	}

	connections := make([]*Connection, len(out.Items))
	for i, item := range out.Items {
		var c Connection
		if err := attributevalue.UnmarshalMap(item, &c); err != nil {
			return nil, fmt.Errorf("store: unmarshal connection: %w", err)
		}
		connections[i] = &c
	}
	return connections, nil
}
