package summarizer

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	"github.com/clobrano/briefly/internal/models"
)

type GeminiSummarizer struct {
	client *genai.Client
	model  string
}

func NewGeminiSummarizer(ctx context.Context, apiKey, model string) (*GeminiSummarizer, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiSummarizer{
		client: client,
		model:  model,
	}, nil
}

func (g *GeminiSummarizer) Summarize(ctx context.Context, content, customPrompt string, contentType models.ContentType) (string, error) {
	prompt := customPrompt
	if prompt == "" {
		prompt = GetDefaultPrompt(contentType)
	}

	fullPrompt := fmt.Sprintf("%s\n\n---\n\nContent to summarize:\n\n%s", prompt, content)

	result, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(fullPrompt), nil)
	if err != nil {
		return "", fmt.Errorf("gemini API error: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	// Extract text from response
	var text string
	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			text += part.Text
		}
	}

	return text, nil
}
