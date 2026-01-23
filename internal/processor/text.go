package processor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	readability "github.com/go-shiori/go-readability"
)

type TextExtractor struct {
	client *http.Client
}

func NewTextExtractor() *TextExtractor {
	return &TextExtractor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *TextExtractor) Extract(ctx context.Context, url string) (string, error) {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	if article.TextContent == "" {
		return "", fmt.Errorf("no text content extracted from URL")
	}

	return article.TextContent, nil
}
