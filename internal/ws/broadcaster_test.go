package ws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAPIGatewayManagementAPI struct {
	mock.Mock
}

func (m *mockAPIGatewayManagementAPI) PostToConnection(ctx context.Context, params *apigatewaymanagementapi.PostToConnectionInput, optFns ...func(*apigatewaymanagementapi.Options)) (*apigatewaymanagementapi.PostToConnectionOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*apigatewaymanagementapi.PostToConnectionOutput)
	return out, args.Error(1)
}

func newTestBroadcaster(client *mockAPIGatewayManagementAPI) *APIGatewayBroadcaster {
	return &APIGatewayBroadcaster{client: client}
}

func TestNewAPIGatewayBroadcaster(t *testing.T) {
	b := NewAPIGatewayBroadcaster(&apigatewaymanagementapi.Client{})

	assert.NotNil(t, b.client)
}

func TestSend_Success(t *testing.T) {
	client := new(mockAPIGatewayManagementAPI)
	client.On("PostToConnection", mock.Anything, mock.MatchedBy(func(in *apigatewaymanagementapi.PostToConnectionInput) bool {
		return *in.ConnectionId == "conn-1" && string(in.Data) == "hello"
	})).Return(&apigatewaymanagementapi.PostToConnectionOutput{}, nil)

	err := newTestBroadcaster(client).Send(context.Background(), "conn-1", []byte("hello"))

	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestSend_Gone(t *testing.T) {
	client := new(mockAPIGatewayManagementAPI)
	client.On("PostToConnection", mock.Anything, mock.Anything).
		Return(&apigatewaymanagementapi.PostToConnectionOutput{}, &types.GoneException{})

	err := newTestBroadcaster(client).Send(context.Background(), "conn-1", []byte("hello"))

	assert.ErrorIs(t, err, ErrConnectionGone)
}

func TestSend_OtherError(t *testing.T) {
	client := new(mockAPIGatewayManagementAPI)
	client.On("PostToConnection", mock.Anything, mock.Anything).
		Return(&apigatewaymanagementapi.PostToConnectionOutput{}, errors.New("network error"))

	err := newTestBroadcaster(client).Send(context.Background(), "conn-1", []byte("hello"))

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrConnectionGone)
}

type stubBroadcaster struct {
	results map[string]error
	calls   []string
}

func (s *stubBroadcaster) Send(ctx context.Context, connectionID string, payload []byte) error {
	s.calls = append(s.calls, connectionID)
	return s.results[connectionID]
}

func TestBroadcastAll_AllSucceed(t *testing.T) {
	b := &stubBroadcaster{results: map[string]error{}}

	gone := BroadcastAll(context.Background(), b, []string{"conn-1", "conn-2"}, []byte("hello"))

	assert.Empty(t, gone)
	assert.Equal(t, []string{"conn-1", "conn-2"}, b.calls)
}

func TestBroadcastAll_SomeGone(t *testing.T) {
	b := &stubBroadcaster{results: map[string]error{
		"conn-1": ErrConnectionGone,
		"conn-3": ErrConnectionGone,
	}}

	gone := BroadcastAll(context.Background(), b, []string{"conn-1", "conn-2", "conn-3"}, []byte("hello"))

	assert.Equal(t, []string{"conn-1", "conn-3"}, gone)
}

func TestBroadcastAll_OtherErrorIgnored(t *testing.T) {
	b := &stubBroadcaster{results: map[string]error{
		"conn-1": errors.New("network error"),
	}}

	gone := BroadcastAll(context.Background(), b, []string{"conn-1"}, []byte("hello"))

	assert.Empty(t, gone)
}
