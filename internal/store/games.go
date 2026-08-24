package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	ErrGameNotFound      = errors.New("store: game not found")
	ErrGameAlreadyExists = errors.New("store: game already exists")
	ErrConcurrentUpdate  = errors.New("store: game was updated concurrently")
)

type Players struct {
	White string `json:"white" dynamodbav:"white"`
	Black string `json:"black" dynamodbav:"black"`
}

type Game struct {
	GameID               string  `json:"gameId" dynamodbav:"gameId"`
	FEN                  string  `json:"fen" dynamodbav:"fen"`
	PGN                  string  `json:"pgn" dynamodbav:"pgn"`
	Players              Players `json:"players" dynamodbav:"players"`
	TurnOf               string  `json:"turnOf" dynamodbav:"turnOf"`
	Status               string  `json:"status" dynamodbav:"status"`
	Version              int     `json:"version" dynamodbav:"version"`
	WhiteTimeMs          int64   `json:"whiteTimeMs" dynamodbav:"whiteTimeMs"`
	BlackTimeMs          int64   `json:"blackTimeMs" dynamodbav:"blackTimeMs"`
	LastMoveAt           int64   `json:"lastMoveAt" dynamodbav:"lastMoveAt"`
	EndedAt              int64   `json:"endedAt,omitempty" dynamodbav:"endedAt,omitempty"`
	Winner               string  `json:"winner,omitempty" dynamodbav:"winner,omitempty"`
	DrawOfferedBy        string  `json:"drawOfferedBy,omitempty" dynamodbav:"drawOfferedBy,omitempty"`
	VsAI                 bool    `json:"vsAI" dynamodbav:"vsAI"`
	AILevel              string  `json:"aiLevel,omitempty" dynamodbav:"aiLevel,omitempty"`
	DisconnectedPlayerID string  `json:"-" dynamodbav:"disconnectedPlayerId,omitempty"`
	DisconnectedAt       int64   `json:"-" dynamodbav:"disconnectedAt,omitempty"`
}

type GameStore interface {
	CreateGame(ctx context.Context, g *Game) error
	GetGame(ctx context.Context, gameID string) (*Game, error)
	UpdateGame(ctx context.Context, g *Game, expectedFEN string) error
	ClearDisconnect(ctx context.Context, gameID, playerID string) error
	MarkDisconnected(ctx context.Context, gameID, playerID string, at int64) error
}

type dynamoDBAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

type DynamoGameStore struct {
	client    dynamoDBAPI
	tableName string
}

func NewDynamoGameStore(client *dynamodb.Client, tableName string) *DynamoGameStore {
	return &DynamoGameStore{client: client, tableName: tableName}
}

func (s *DynamoGameStore) CreateGame(ctx context.Context, g *Game) error {
	item, err := attributevalue.MarshalMap(g)
	if err != nil {
		return fmt.Errorf("store: marshal game: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           &s.tableName,
		Item:                item,
		ConditionExpression: strPtr("attribute_not_exists(gameId)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrGameAlreadyExists
		}
		return fmt.Errorf("store: create game: %w", err)
	}
	return nil
}

func (s *DynamoGameStore) GetGame(ctx context.Context, gameID string) (*Game, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"gameId": &types.AttributeValueMemberS{Value: gameID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("store: get game: %w", err)
	}
	if out.Item == nil {
		return nil, ErrGameNotFound
	}

	var g Game
	if err := attributevalue.UnmarshalMap(out.Item, &g); err != nil {
		return nil, fmt.Errorf("store: unmarshal game: %w", err)
	}
	return &g, nil
}

func (s *DynamoGameStore) UpdateGame(ctx context.Context, g *Game, expectedFEN string) error {
	item, err := attributevalue.MarshalMap(g)
	if err != nil {
		return fmt.Errorf("store: marshal game: %w", err)
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           &s.tableName,
		Item:                item,
		ConditionExpression: strPtr("fen = :expectedFen"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":expectedFen": &types.AttributeValueMemberS{Value: expectedFEN},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return ErrConcurrentUpdate
		}
		return fmt.Errorf("store: update game: %w", err)
	}
	return nil
}

func (s *DynamoGameStore) ClearDisconnect(ctx context.Context, gameID, playerID string) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"gameId": &types.AttributeValueMemberS{Value: gameID},
		},
		UpdateExpression:    strPtr("REMOVE disconnectedPlayerId, disconnectedAt"),
		ConditionExpression: strPtr("disconnectedPlayerId = :playerId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":playerId": &types.AttributeValueMemberS{Value: playerID},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return nil
		}
		return fmt.Errorf("store: clear disconnect: %w", err)
	}
	return nil
}

func (s *DynamoGameStore) MarkDisconnected(ctx context.Context, gameID, playerID string, at int64) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"gameId": &types.AttributeValueMemberS{Value: gameID},
		},
		UpdateExpression:         strPtr("SET disconnectedPlayerId = :playerId, disconnectedAt = :at"),
		ConditionExpression:      strPtr("#status = :inProgress"),
		ExpressionAttributeNames: map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":playerId":   &types.AttributeValueMemberS{Value: playerID},
			":at":         &types.AttributeValueMemberN{Value: strconv.FormatInt(at, 10)},
			":inProgress": &types.AttributeValueMemberS{Value: "in_progress"},
		},
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return nil
		}
		return fmt.Errorf("store: mark disconnected: %w", err)
	}
	return nil
}

func isConditionalCheckFailed(err error) bool {
	var condErr *types.ConditionalCheckFailedException
	return errors.As(err, &condErr)
}

func strPtr(s string) *string {
	return &s
}
