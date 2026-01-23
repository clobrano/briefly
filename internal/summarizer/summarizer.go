package summarizer

import (
	"context"

	"github.com/clobrano/briefly/internal/models"
)

type Summarizer interface {
	Summarize(ctx context.Context, content, customPrompt string, contentType models.ContentType) (string, error)
}

const DefaultYouTubePrompt = `You are analyzing a YouTube video transcript. Please provide a comprehensive summary that includes:

1. **Main Topic**: What is the video about?
2. **Key Points**: List the main arguments, ideas, or information presented
3. **Important Details**: Any statistics, quotes, or specific examples mentioned
4. **Conclusion**: What are the main takeaways?

Keep the summary concise but informative. Use bullet points where appropriate.`

const DefaultTextPrompt = `You are analyzing a web article. Please provide a comprehensive summary that includes:

1. **Main Topic**: What is the article about?
2. **Key Points**: List the main arguments or information presented
3. **Important Details**: Any statistics, quotes, or specific examples mentioned
4. **Conclusion**: What are the main takeaways?

Keep the summary concise but informative. Use bullet points where appropriate.`

func GetDefaultPrompt(contentType models.ContentType) string {
	switch contentType {
	case models.ContentTypeYouTube:
		return DefaultYouTubePrompt
	case models.ContentTypeText:
		return DefaultTextPrompt
	default:
		return DefaultTextPrompt
	}
}
