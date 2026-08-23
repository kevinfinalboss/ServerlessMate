package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("load aws config", "error", err.Error())
		os.Exit(1)
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)
	apiGwClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("WEBSOCKET_API_ENDPOINT"))
	})

	gamesTable := os.Getenv("GAMES_TABLE")
	queueTable := os.Getenv("QUEUE_TABLE")

	d := deps{
		queue:       store.NewDynamoQueueStore(dynamoClient, queueTable),
		match:       store.NewDynamoMatchStore(dynamoClient, gamesTable, queueTable),
		connections: store.NewDynamoConnectionStore(dynamoClient, os.Getenv("CONNECTIONS_TABLE")),
		broadcaster: ws.NewAPIGatewayBroadcaster(apiGwClient),
		now:         time.Now,
		newGameID:   func() string { return uuid.NewString() },
	}

	lambda.Start(func(ctx context.Context, event events.DynamoDBEvent) error {
		return handleEvent(ctx, d, event)
	})
}

func handleEvent(ctx context.Context, d deps, event events.DynamoDBEvent) error {
	for _, record := range event.Records {
		if record.EventName != "INSERT" {
			continue
		}

		self, err := parseQueueEntry(record.Change.NewImage)
		if err != nil {
			logger.Error("parse queue entry", "error", err.Error())
			continue
		}

		if err := handle(ctx, d, self); err != nil {
			logger.Error("handle match", "playerId", self.PlayerID, "matchmakingKey", self.MatchmakingKey, "error", err.Error())
		}
	}
	return nil
}

func parseQueueEntry(image map[string]events.DynamoDBAttributeValue) (*store.QueueEntry, error) {
	joinedAt, err := image["joinedAt"].Int64()
	if err != nil {
		return nil, fmt.Errorf("parse joinedAt: %w", err)
	}

	return &store.QueueEntry{
		MatchmakingKey: image["matchmakingKey"].String(),
		SortKey:        image["sortKey"].String(),
		PlayerID:       image["playerId"].String(),
		ConnectionID:   image["connectionId"].String(),
		IsGuest:        image["isGuest"].Boolean(),
		JoinedAt:       joinedAt,
	}, nil
}
