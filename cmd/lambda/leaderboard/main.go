package main

import (
	"context"
	"log/slog"
	"os"

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

	d := deps{
		players:     store.NewDynamoPlayerStore(dynamoClient, os.Getenv("PLAYERS_TABLE")),
		broadcaster: ws.NewAPIGatewayBroadcaster(apiGwClient),
	}

	if err := handle(ctx, d, connectionID, []byte(event.Body)); err != nil {
		logger.Error("handle action", "connectionId", connectionID, "action", "leaderboard", "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}
