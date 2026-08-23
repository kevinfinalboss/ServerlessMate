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
	"github.com/google/uuid"

	"github.com/kevinfinalboss/ServerlessMate/internal/auth"
	"github.com/kevinfinalboss/ServerlessMate/internal/store"
	"github.com/kevinfinalboss/ServerlessMate/internal/ws"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

const defaultReconnectGraceMs = 60_000

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("load aws config", "error", err.Error())
		os.Exit(1)
	}

	validator, err := auth.NewCognitoValidator(ctx, os.Getenv("COGNITO_JWKS_URL"), os.Getenv("COGNITO_ISSUER"))
	if err != nil {
		logger.Error("init jwt validator", "error", err.Error())
		os.Exit(1)
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)

	graceMs, parseErr := strconv.ParseInt(os.Getenv("RECONNECT_GRACE_MS"), 10, 64)
	if parseErr != nil || graceMs <= 0 {
		graceMs = defaultReconnectGraceMs
	}

	games := store.NewDynamoGameStore(dynamoClient, os.Getenv("GAMES_TABLE"))
	connections := store.NewDynamoConnectionStore(dynamoClient, os.Getenv("CONNECTIONS_TABLE"))

	lambda.Start(func(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		endpoint := "https://" + event.RequestContext.DomainName + "/" + event.RequestContext.Stage
		apiGwClient := apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})

		d := deps{
			games:       games,
			connections: connections,
			validator:   validator,
			broadcaster: ws.NewAPIGatewayBroadcaster(apiGwClient),
			newGuestID:  func() string { return uuid.NewString() },
			now:         time.Now,
			graceMs:     graceMs,
		}

		return handleEvent(ctx, d, event)
	})
}

func handleEvent(ctx context.Context, d deps, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	connectionID := event.RequestContext.ConnectionID
	token := event.QueryStringParameters["token"]
	gameID := event.QueryStringParameters["gameId"]

	if err := handle(ctx, d, connectionID, token, gameID); err != nil {
		logger.Error("reject connect", "connectionId", connectionID, "error", err.Error())
		return events.APIGatewayProxyResponse{StatusCode: 401}, nil
	}
	return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}
