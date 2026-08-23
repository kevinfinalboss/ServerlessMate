package ai

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

var ErrEmptyResponse = errors.New("ai: empty bedrock response")

type Commentator interface {
	Comment(ctx context.Context, fen, uci string) (string, error)
}

type bedrockConverseAPI interface {
	Converse(ctx context.Context, params *bedrockruntime.ConverseInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.ConverseOutput, error)
}

type BedrockCommentator struct {
	client  bedrockConverseAPI
	modelID string
}

func NewBedrockCommentator(client *bedrockruntime.Client, modelID string) *BedrockCommentator {
	return &BedrockCommentator{client: client, modelID: modelID}
}

func (c *BedrockCommentator) Comment(ctx context.Context, fen, uci string) (string, error) {
	prompt := fmt.Sprintf(
		"Você é um oponente de xadrez com personalidade sarcástica mas simpática. "+
			"O motor de xadrez acabou de jogar %s na posição FEN %s. "+
			"Comente o lance em até 2 frases curtas, em português, sem sugerir jogadas futuras nem analisar profundamente — só uma reação de personagem.",
		uci, fen,
	)

	out, err := c.client.Converse(ctx, &bedrockruntime.ConverseInput{
		ModelId: aws.String(c.modelID),
		Messages: []types.Message{
			{
				Role:    types.ConversationRoleUser,
				Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: prompt}},
			},
		},
		InferenceConfig: &types.InferenceConfiguration{
			MaxTokens: aws.Int32(120),
		},
	})
	if err != nil {
		return "", fmt.Errorf("ai: bedrock converse: %w", err)
	}

	msg, ok := out.Output.(*types.ConverseOutputMemberMessage)
	if !ok || len(msg.Value.Content) == 0 {
		return "", ErrEmptyResponse
	}
	text, ok := msg.Value.Content[0].(*types.ContentBlockMemberText)
	if !ok {
		return "", ErrEmptyResponse
	}
	return text.Value, nil
}
