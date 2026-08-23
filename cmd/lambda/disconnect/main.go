package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
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

	d := deps{
		games:       store.NewDynamoGameStore(dynamoClient, os.Getenv("GAMES_TABLE")),
		connections: store.NewDynamoConnectionStore(dynamoClient, os.Getenv("CONNECTIONS_TABLE")),
		now:         time.Now,
	}

	lambda.Start(func(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		return handleEvent(ctx, d, event)
	})
}

func handleEvent(ctx context.Context, d deps, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := event.RequestContext.ConnectionID

	if err := handle(ctx, d, connectionID); err != nil {
		logger.Error("handle disconnect", "connectionId", connectionID, "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}
