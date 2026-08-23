package ws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
)

var ErrConnectionGone = errors.New("ws: connection is gone")

type Broadcaster interface {
	Send(ctx context.Context, connectionID string, payload []byte) error
}

type apiGatewayManagementAPI interface {
	PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error)
}

type APIGatewayBroadcaster struct {
	client apiGatewayManagementAPI
}

func NewAPIGatewayBroadcaster(client *apigatewaymanagementapi.Client) *APIGatewayBroadcaster {
	return &APIGatewayBroadcaster{client: client}
}

func (b *APIGatewayBroadcaster) Send(ctx context.Context, connectionID string, payload []byte) error {
	_, err := b.client.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: &connectionID,
		Data:         payload,
	})
	if err != nil {
		var goneErr *types.GoneException
		if errors.As(err, &goneErr) {
			return ErrConnectionGone
		}
		return fmt.Errorf("ws: post to connection %s: %w", connectionID, err)
	}
	return nil
}

func BroadcastAll(ctx context.Context, b Broadcaster, connectionIDs []string, payload []byte) []string {
	var gone []string
	for _, id := range connectionIDs {
		if err := b.Send(ctx, id, payload); err != nil && errors.Is(err, ErrConnectionGone) {
			gone = append(gone, id)
		}
	}
	return gone
}
