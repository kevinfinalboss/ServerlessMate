package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

const defaultReconnectGraceMs = 60_000

func main() {
	lambda.Start(handleEvent)
}

func handleEvent(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := event.RequestContext.ConnectionID

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("load aws config", "connectionId", connectionID, "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)
	endpoint := "https://" + event.RequestContext.DomainName + "/" + event.RequestContext.Stage
	apiGwClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	graceMs, parseErr := strconv.ParseInt(os.Getenv("RECONNECT_GRACE_MS"), 10, 64)
	if parseErr != nil || graceMs <= 0 {
		graceMs = defaultReconnectGraceMs
	}

	d := deps{
		games:       store.NewDynamoGameStore(dynamoClient, os.Getenv("GAMES_TABLE")),
		connections: store.NewDynamoConnectionStore(dynamoClient, os.Getenv("CONNECTIONS_TABLE")),
		players:     store.NewDynamoPlayerStore(dynamoClient, os.Getenv("PLAYERS_TABLE")),
		history:     store.NewDynamoHistoryStore(dynamoClient, os.Getenv("GAME_HISTORY_TABLE")),
		broadcaster: ws.NewAPIGatewayBroadcaster(apiGwClient),
		now:         time.Now,
		graceMs:     graceMs,
	}

	if err := handle(ctx, d, connectionID, []byte(event.Body)); err != nil {
		logger.Error("handle action", "connectionId", connectionID, "action", "makeMove", "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}
