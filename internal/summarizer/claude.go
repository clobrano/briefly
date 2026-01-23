package summarizer

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/clobrano/briefly/internal/models"
)

type ClaudeSummarizer struct {
	client *anthropic.Client
	model  string
}

func NewClaudeSummarizer(apiKey, model string) (*ClaudeSummarizer, error) {
	client := anthropic.NewClient()
	return &ClaudeSummarizer{
		client: &client,
		model:  model,
	}, nil
}

func (c *ClaudeSummarizer) Summarize(ctx context.Context, content, customPrompt string, contentType models.ContentType) (string, error) {
	prompt := customPrompt
	if prompt == "" {
		prompt = GetDefaultPrompt(contentType)
	}

	fullPrompt := fmt.Sprintf("%s\n\n---\n\nContent to summarize:\n\n%s", prompt, content)

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(fullPrompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude API error: %w", err)
	}

	if len(message.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}

	// Extract text from response
	var result string
	for _, block := range message.Content {
		textBlock := block.AsText()
		if textBlock.Text != "" {
			result += textBlock.Text
		}
	}

	return result, nil
}
