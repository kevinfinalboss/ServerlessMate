package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockBedrockConverseAPI struct {
	mock.Mock
}

func (m *mockBedrockConverseAPI) Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error) {
	args := m.Called(ctx, params)
	out, _ := args.Get(0).(*bedrockruntime.ConverseOutput)
	return out, args.Error(1)
}

func newTestCommentator(client *mockBedrockConverseAPI) *BedrockCommentator {
	return &BedrockCommentator{client: client, modelID: "test-model"}
}

func TestNewBedrockCommentator(t *testing.T) {
	c := NewBedrockCommentator(&bedrockruntime.Client{}, "test-model")

	assert.Equal(t, "test-model", c.modelID)
	assert.NotNil(t, c.client)
}

func TestComment_Success(t *testing.T) {
	client := new(mockBedrockConverseAPI)
	client.On("Converse", mock.Anything, mock.MatchedBy(func(in *bedrockruntime.ConverseInput) bool {
		return *in.ModelId == "test-model" && len(in.Messages) == 1
	})).Return(&bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: "Boa jogada!"}}},
		},
	}, nil)

	comment, err := newTestCommentator(client).Comment(context.Background(), "start-fen", "e2e4")

	require.NoError(t, err)
	assert.Equal(t, "Boa jogada!", comment)
}

func TestComment_ClientError(t *testing.T) {
	client := new(mockBedrockConverseAPI)
	client.On("Converse", mock.Anything, mock.Anything).Return(&bedrockruntime.ConverseOutput{}, errors.New("network error"))

	_, err := newTestCommentator(client).Comment(context.Background(), "start-fen", "e2e4")

	require.Error(t, err)
}

func TestComment_EmptyContent(t *testing.T) {
	client := new(mockBedrockConverseAPI)
	client.On("Converse", mock.Anything, mock.Anything).Return(&bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{Value: types.Message{Content: nil}},
	}, nil)

	_, err := newTestCommentator(client).Comment(context.Background(), "start-fen", "e2e4")

	assert.ErrorIs(t, err, ErrEmptyResponse)
}

func TestComment_UnexpectedContentType(t *testing.T) {
	client := new(mockBedrockConverseAPI)
	client.On("Converse", mock.Anything, mock.Anything).Return(&bedrockruntime.ConverseOutput{
		Output: &types.ConverseOutputMemberMessage{
			Value: types.Message{Content: []types.ContentBlock{&types.ContentBlockMemberImage{}}},
		},
	}, nil)

	_, err := newTestCommentator(client).Comment(context.Background(), "start-fen", "e2e4")

	assert.ErrorIs(t, err, ErrEmptyResponse)
}
